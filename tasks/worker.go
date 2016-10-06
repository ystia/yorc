package tasks

import (
	"context"
	"fmt"
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/log"
	"path"
	"sync"
)

type Worker struct {
	workerPool   chan chan *Task
	TaskChannel  chan *Task
	shutdownCh   chan struct{}
	consulClient *api.Client
	cfg          config.Configuration
	runStepLock  sync.Locker
}

func NewWorker(workerPool chan chan *Task, shutdownCh chan struct{}, consulClient *api.Client, cfg config.Configuration) Worker {
	return Worker{
		workerPool:   workerPool,
		TaskChannel:  make(chan *Task),
		shutdownCh:   shutdownCh,
		consulClient: consulClient,
		cfg:          cfg,
		runStepLock:  &sync.Mutex{}}
}

func (w Worker) setDeploymentStatus(deploymentId string, status deployments.DeploymentStatus) {
	p := &api.KVPair{Key: path.Join(deployments.DeploymentKVPrefix, deploymentId, "status"), Value: []byte(fmt.Sprint(status))}
	kv := w.consulClient.KV()
	kv.Put(p, nil)
}

func (w Worker) processWorkflow(wfSteps []*Step, deploymentId string, isUndeploy bool) error {
	deployments.LogInConsul(w.consulClient.KV(), deploymentId, "Start processing workflow")
	var wg sync.WaitGroup
	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)
	errc := make(chan error)
	unistallerrc := make(chan error)
	for _, step := range wfSteps {
		wg.Add(1)
		log.Debugf("Running step %q", step.Name)
		go step.run(ctx, deploymentId, &wg, w.consulClient.KV(), errc, unistallerrc, w.shutdownCh, w.cfg, isUndeploy)
	}

	go func(wg *sync.WaitGroup, cancel context.CancelFunc) {
		wg.Wait()
		// Canceling context so below ctx.Done() will be closed and not considered as an error
		cancel()
	}(&wg, cancel)

	var err error
	select {
	case err = <-unistallerrc:
		deployments.LogInConsul(w.consulClient.KV(), deploymentId, fmt.Sprintf("One or more error appear in unistall workflow, please check : %v", err))
		log.Printf("One or more error appear in unistall workflow, please check : %v", err)
	case err = <-errc:
		deployments.LogInConsul(w.consulClient.KV(), deploymentId, fmt.Sprintf("Error '%v' happened in workflow.", err))
		log.Printf("Error '%v' happened in workflow.", err)
		log.Debug("Canceling it.")
		cancel()
	case <-ctx.Done():
		deployments.LogInConsul(w.consulClient.KV(), deploymentId, "Workflow ended without error")
		log.Printf("Workflow ended without error")
	}
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
				task.WithStatus(RUNNING)
				switch task.TaskType {

				case DEPLOY:
					w.setDeploymentStatus(task.TargetId, deployments.DEPLOYMENT_IN_PROGRESS)
					wf, err := readWorkFlowFromConsul(w.consulClient.KV(), path.Join(deployments.DeploymentKVPrefix, task.TargetId, "workflows/install"))
					if err != nil {
						task.WithStatus(FAILED)
						log.Printf("%v. Aborting", err)
						w.setDeploymentStatus(task.TargetId, deployments.DEPLOYMENT_FAILED)
						continue

					}
					if err = w.processWorkflow(wf, task.TargetId, false); err != nil {
						task.WithStatus(FAILED)
						log.Printf("%v. Aborting", err)
						w.setDeploymentStatus(task.TargetId, deployments.DEPLOYMENT_FAILED)
						continue
					}
					w.setDeploymentStatus(task.TargetId, deployments.DEPLOYED)
				case UNDEPLOY:
					w.setDeploymentStatus(task.TargetId, deployments.UNDEPLOYMENT_IN_PROGRESS)
					wf, err := readWorkFlowFromConsul(w.consulClient.KV(), path.Join(deployments.DeploymentKVPrefix, task.TargetId, "workflows/uninstall"))
					if err != nil {
						task.WithStatus(FAILED)
						log.Printf("%v. Aborting", err)
						w.setDeploymentStatus(task.TargetId, deployments.UNDEPLOYMENT_FAILED)
						continue

					}
					if err = w.processWorkflow(wf, task.TargetId, true); err != nil {
						task.WithStatus(FAILED)
						log.Printf("%v. Aborting", err)
						w.setDeploymentStatus(task.TargetId, deployments.UNDEPLOYMENT_FAILED)
						continue
					}
					w.setDeploymentStatus(task.TargetId, deployments.UNDEPLOYED)
				}
				task.WithStatus(DONE)
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
