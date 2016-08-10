package tasks

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"golang.org/x/net/context"
	"novaforge.bull.com/starlings-janus/janus/commands/jconfig"
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
	cfg          jconfig.Configuration
}

func NewWorker(workerPool chan chan *Task, shutdownCh chan struct{}, consulClient *api.Client, cfg jconfig.Configuration) Worker {
	return Worker{
		workerPool:   workerPool,
		TaskChannel:  make(chan *Task),
		shutdownCh:   shutdownCh,
		consulClient: consulClient,
		cfg:          cfg}
}

func (w Worker) setDeploymentStatus(deploymentId string, status deployments.DeploymentStatus) {
	p := &api.KVPair{Key: path.Join(deployments.DeploymentKVPrefix, deploymentId, "status"), Value: []byte(fmt.Sprint(status))}
	kv := w.consulClient.KV()
	kv.Put(p, nil)
}

func (w Worker) runStep(ctx context.Context, step *Step, deploymentId string, wg *sync.WaitGroup, errc chan error, runningSteps map[string]struct{}) {
	if _, ok := runningSteps[step.Name]; ok {
		// Already running
		log.Debugf("step %q already running", step.Name)
		return
	}
	log.Debugf("Running step %q", step.Name)
	runningSteps[step.Name] = struct{}{}
	go step.run(ctx, deploymentId, wg, w.consulClient.KV(), errc, w.shutdownCh, w.cfg)
	for _, next := range step.Next {
		wg.Add(1)
		log.Debugf("Try run next step %q", next.Name)
		go w.runStep(ctx, next, deploymentId, wg, errc, runningSteps)
	}
}

func (w Worker) processWorkflow(wfRoots []*Step, deploymentId string) error {
	var wg sync.WaitGroup
	runningSteps := make(map[string]struct{})
	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)
	errc := make(chan error)
	for _, step := range wfRoots {
		wg.Add(1)
		go w.runStep(ctx, step, deploymentId, &wg, errc, runningSteps)
	}

	go func(wg *sync.WaitGroup, cancel context.CancelFunc) {
		wg.Wait()
		// Canceling context so below ctx.Done() will be closed and not considered as an error
		cancel()
	}(&wg, cancel)

	var err error
	select {
	case err = <-errc:
		log.Printf("Error '%v' happened in workflow.", err)
		log.Debug("Canceling it.")
		cancel()
	case <-ctx.Done():
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
