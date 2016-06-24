package tasks

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"log"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/prov/ansible"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform"
	"path"
	"sync"
)

type Worker struct {
	workerPool   chan chan *Task
	TaskChannel  chan *Task
	shutdownCh   chan struct{}
	consulClient *api.Client
}

func NewWorker(workerPool chan chan *Task, shutdownCh chan struct{}, consulClient *api.Client) Worker {
	return Worker{
		workerPool:  workerPool,
		TaskChannel: make(chan *Task),
		shutdownCh:  shutdownCh, consulClient: consulClient}
}

func (w Worker) setDeploymentStatus(deploymentId string, status deployments.DeploymentStatus) {
	p := &api.KVPair{Key: path.Join(deployments.DeploymentKVPrefix, deploymentId, "status"), Value: []byte(fmt.Sprint(status))}
	kv := w.consulClient.KV()
	kv.Put(p, nil)
}

func (w Worker) processStep(step *Step, deploymentId string, wg *sync.WaitGroup, errc chan error) {
	defer wg.Done()
	log.Printf("Processing step %s", step.Name)
	for _, activity := range step.Activities {
		actType := activity.ActivityType()
		switch {
		case actType == "delegate":
			provisioner := terraform.NewExecutor(w.consulClient.KV())
			delegateOp := activity.ActivityValue()
			switch delegateOp {
			case "install":
				if err := provisioner.ProvisionNode(deploymentId, step.Node); err != nil {
					log.Printf("Sending error %v to error channel", err)
					errc <- err
					return
				}
			case "uninstall":
				if err := provisioner.DestroyNode(deploymentId, step.Node); err != nil {
					errc <- err
					return
				}
			default:
				errc <- fmt.Errorf("Unsupported delegate operation '%s' for step '%s'", delegateOp, step.Name)
				return
			}
		case actType == "set-state":
			w.consulClient.KV().Put(&api.KVPair{Key: path.Join(deployments.DeploymentKVPrefix, deploymentId, "topology/nodes", step.Node, "status"), Value: []byte(activity.ActivityValue())}, nil)
		case actType == "call-operation":
			exec := ansible.NewExecutor(w.consulClient.KV())
			if err := exec.ExecOperation(deploymentId, step.Node, activity.ActivityValue()); err != nil {
				errc <- err
				return
			}
		}
	}
	for _, next := range step.Next {
		wg.Add(1)
		go w.processStep(next, deploymentId, wg, errc)
	}
}

func (w Worker) processWorkflow(wfRoots []*Step, deploymentId string) error {
	var wg sync.WaitGroup
	errc := make(chan error)
	for _, step := range wfRoots {
		wg.Add(1)
		go w.processStep(step, deploymentId, &wg, errc)
	}
	wg.Wait()
	log.Printf("All step done. Checking if there was an error")
	var err error
	select {
	case err = <-errc:
	default:
	}
	log.Printf("Workflow ended with error: '%v'", err)
	return err
}

// Start method starts the run loop for the worker, listening for a quit channel in
// case we need to stop it
func (w Worker) Start() {
	go func() {
		for {
			// register the current worker into the worker queue.
			w.workerPool <- w.TaskChannel

			select {
			case task := <-w.TaskChannel:
				// we have received a work request.
				log.Printf("Worker got task with id %s", task.Id)

				switch task.TaskType {

				case DEPLOY:
					w.setDeploymentStatus(task.TargetId, deployments.DEPLOYMENT_IN_PROGRESS)
					wf, err := readWorkFlowFromConsul(w.consulClient.KV(), path.Join(deployments.DeploymentKVPrefix, task.TargetId, "workflows/install"))
					if err != nil {
						task.WithStatus("failed")
						log.Printf("%v. Aborting", err)
						w.setDeploymentStatus(task.TargetId, deployments.DEPLOYMENT_FAILED)
						continue

					}
					if err = w.processWorkflow(wf, task.TargetId); err != nil {
						task.WithStatus("failed")
						log.Printf("%v. Aborting", err)
						w.setDeploymentStatus(task.TargetId, deployments.DEPLOYMENT_FAILED)
						continue
					}
					w.setDeploymentStatus(task.TargetId, deployments.DEPLOYED)
				case UNDEPLOY:
					w.setDeploymentStatus(task.TargetId, deployments.UNDEPLOYMENT_IN_PROGRESS)
					wf, err := readWorkFlowFromConsul(w.consulClient.KV(), path.Join(deployments.DeploymentKVPrefix, task.TargetId, "workflows/uninstall"))
					if err != nil {
						task.WithStatus("failed")
						log.Printf("%v. Aborting", err)
						w.setDeploymentStatus(task.TargetId, deployments.UNDEPLOYMENT_FAILED)
						continue

					}
					if err = w.processWorkflow(wf, task.TargetId); err != nil {
						task.WithStatus("failed")
						log.Printf("%v. Aborting", err)
						w.setDeploymentStatus(task.TargetId, deployments.UNDEPLOYMENT_FAILED)
						continue
					}
					w.setDeploymentStatus(task.TargetId, deployments.UNDEPLOYED)
				}
				task.WithStatus("done")
				task.TaskLock.Unlock()
				task.TaskLock.Destroy()

			case <-w.shutdownCh:
				// we have received a signal to stop
				log.Printf("Worker received shutdown signal. Exiting...")
				return
			}
		}
	}()
}
