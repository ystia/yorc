package tasks

import (
	"context"
	"fmt"
	"github.com/hashicorp/consul/api"
	"golang.org/x/sync/errgroup"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/ansible"
	"path"
	"strings"
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
	p := &api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, deploymentId, "status"), Value: []byte(fmt.Sprint(status))}
	kv := w.consulClient.KV()
	kv.Put(p, nil)
}

func (w Worker) processWorkflow(ctx context.Context, wfSteps []*Step, deploymentId string, isUndeploy bool) error {
	deployments.LogInConsul(w.consulClient.KV(), deploymentId, "Start processing workflow")
	uninstallerrc := make(chan error)

	g, ctx := errgroup.WithContext(ctx)
	for _, step := range wfSteps {
		log.Debugf("Running step %q", step.Name)
		// The use of the function bellow is a classic gotcha with go routines. It allows to use a fixed step as it is computed when the
		// bellow function executes (ie before g.Go or go ... in other case). Otherwise step is computed when the lambda function executes
		// ie in the other go routine and at this time step may have changed as we are in a for loop.
		func(step *Step) {
			g.Go(func() error {
				return step.run(ctx, deploymentId, w.consulClient.KV(), uninstallerrc, w.shutdownCh, w.cfg, isUndeploy)
			})
		}(step)
	}

	faninErrCh := make(chan []string)
	defer close(faninErrCh)
	go func() {
		errors := make([]string, 0)
		// Reads from uninstallerrc until the channel is close
		for err := range uninstallerrc {
			errors = append(errors, err.Error())
		}
		faninErrCh <- errors
	}()

	err := g.Wait()
	// Be sure to close the uninstall errors channel before exiting or trying to read from fan-in errors channel
	close(uninstallerrc)
	errors := <-faninErrCh

	if err != nil {
		deployments.LogInConsul(w.consulClient.KV(), deploymentId, fmt.Sprintf("Error '%v' happened in workflow.", err))
		return err
	}

	if len(errors) > 0 {
		uninstallerr := fmt.Errorf("%s", strings.Join(errors, " ; "))
		deployments.LogInConsul(w.consulClient.KV(), deploymentId, fmt.Sprintf("One or more error appear in unistall workflow, please check : %v", uninstallerr))
		log.Printf("One or more error appear in uninstall workflow, please check : %v", uninstallerr)
	} else {
		deployments.LogInConsul(w.consulClient.KV(), deploymentId, "Workflow ended without error")
		log.Printf("Workflow ended without error")
	}
	return nil
}

func (w Worker) handleTask(task *Task) {
	task.WithStatus(RUNNING)
	bgCtx := context.Background()
	ctx, cancelFunc := context.WithCancel(bgCtx)
	defer task.releaseLock()
	defer cancelFunc()
	w.monitorTaskForCancellation(ctx, cancelFunc, task)
	switch task.TaskType {
	case Deploy:
		w.setDeploymentStatus(task.TargetId, deployments.DEPLOYMENT_IN_PROGRESS)
		wf, err := readWorkFlowFromConsul(w.consulClient.KV(), path.Join(consulutil.DeploymentKVPrefix, task.TargetId, "workflows/install"))
		if err != nil {
			if task.Status() == RUNNING {
				task.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(task.TargetId, deployments.DEPLOYMENT_FAILED)
			return

		}
		if err = w.processWorkflow(ctx, wf, task.TargetId, false); err != nil {
			if task.Status() == RUNNING {
				task.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(task.TargetId, deployments.DEPLOYMENT_FAILED)
			return
		}
		w.setDeploymentStatus(task.TargetId, deployments.DEPLOYED)
	case UnDeploy, Purge:
		w.setDeploymentStatus(task.TargetId, deployments.UNDEPLOYMENT_IN_PROGRESS)
		wf, err := readWorkFlowFromConsul(w.consulClient.KV(), path.Join(consulutil.DeploymentKVPrefix, task.TargetId, "workflows/uninstall"))
		if err != nil {
			if task.Status() == RUNNING {
				task.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(task.TargetId, deployments.UNDEPLOYMENT_FAILED)
			return

		}
		if err = w.processWorkflow(ctx, wf, task.TargetId, true); err != nil {
			if task.Status() == RUNNING {
				task.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(task.TargetId, deployments.UNDEPLOYMENT_FAILED)
			return
		}
		w.setDeploymentStatus(task.TargetId, deployments.UNDEPLOYED)
		if task.TaskType == Purge {
			_, err := w.consulClient.KV().DeleteTree(path.Join(consulutil.DeploymentKVPrefix, task.TargetId), nil)
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge deployment definition: %+v", task.TargetId, task.Id, err)
				task.WithStatus(FAILED)
				return
			}
			tasks, err := GetTasksIdsForTarget(w.consulClient.KV(), task.TargetId)
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", task.TargetId, task.Id, err)
				task.WithStatus(FAILED)
				return
			}
			for _, tid := range tasks {
				if tid != task.Id {
					_, err := w.consulClient.KV().DeleteTree(path.Join(consulutil.TasksPrefix, tid), nil)
					if err != nil {
						log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", task.TargetId, task.Id, err)
						task.WithStatus(FAILED)
						return
					}
				}
			}
			_, err = w.consulClient.KV().DeleteTree(path.Join(consulutil.TasksPrefix, task.Id), nil)
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", task.TargetId, task.Id, err)
				task.WithStatus(FAILED)
				return
			}
			return
		}
	case CustomCommand:
		eventPub := events.NewPublisher(task.kv, task.TargetId)

		nodeNameKv, _, err := w.consulClient.KV().Get(path.Join(consulutil.TasksPrefix, task.Id, "node"), nil)
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to get nodeName: %+v", task.TargetId, task.Id, err)
			task.WithStatus(FAILED)
			return
		}

		commandNameKv, _, err := w.consulClient.KV().Get(path.Join(consulutil.TasksPrefix, task.Id, "name"), nil)
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to get Custom command name: %+v", task.TargetId, task.Id, err)
			task.WithStatus(FAILED)
			return
		}

		nodeName := string(nodeNameKv.Value)
		commandName := string(commandNameKv.Value)

		exec := ansible.NewExecutor(task.kv)
		if err := exec.ExecOperation(ctx, task.TargetId, nodeName, "custom."+commandName, task.Id); err != nil {
			setNodeStatus(task.kv, eventPub, task.TargetId, nodeName, "error")
			log.Printf("Deployment %q, Step %q: Sending error %v to error channel", task.TargetId, nodeName, err)
		}

	default:
		deployments.LogInConsul(w.consulClient.KV(), task.TargetId, fmt.Sprintf("Unknown TaskType %d (%s) for task with id %q", task.TaskType, task.TaskType.String(), task.Id))
		log.Printf("Unknown TaskType %d (%s) for task with id %q and targetId %q", task.TaskType, task.TaskType.String(), task.Id, task.TargetId)
		if task.Status() == RUNNING {
			task.WithStatus(FAILED)
		}
		return
	}
	if task.Status() == RUNNING {
		task.WithStatus(DONE)
	}
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
				w.handleTask(task)

			case <-w.shutdownCh:
				// we have received a signal to stop
				log.Printf("Worker received shutdown signal. Exiting...")
				return
			}
		}
	}()
}

func (w Worker) monitorTaskForCancellation(ctx context.Context, cancelFunc context.CancelFunc, task *Task) {
	go func() {
		var lastIndex uint64 = 0
		for {
			kvp, qMeta, err := w.consulClient.KV().Get(path.Join(consulutil.TasksPrefix, task.Id, ".canceledFlag"), &api.QueryOptions{WaitIndex: lastIndex})

			select {
			case <-ctx.Done():
				log.Debugln("[TASK MONITOR] Task monitor exiting as task ended...")
				return
			default:
			}

			if qMeta != nil {
				lastIndex = qMeta.LastIndex
			}

			if err == nil && kvp != nil {
				if strings.ToLower(string(kvp.Value)) == "true" {
					log.Debugln("[TASK MONITOR] Task cancelation requested.")
					task.WithStatus(CANCELED)
					cancelFunc()
					return
				}
			}
		}
	}()
}
