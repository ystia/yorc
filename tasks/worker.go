package tasks

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/ansible"
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

func (w Worker) setDeploymentStatus(deploymentID string, status deployments.DeploymentStatus) {
	p := &api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "status"), Value: []byte(fmt.Sprint(status))}
	kv := w.consulClient.KV()
	kv.Put(p, nil)
}

func (w Worker) processWorkflow(ctx context.Context, wfSteps []*Step, deploymentID string, isUndeploy bool) error {
	deployments.LogInConsul(w.consulClient.KV(), deploymentID, "Start processing workflow")
	uninstallerrc := make(chan error)

	g, ctx := errgroup.WithContext(ctx)
	for _, step := range wfSteps {
		log.Debugf("Running step %q", step.Name)
		// The use of the function bellow is a classic gotcha with go routines. It allows to use a fixed step as it is computed when the
		// bellow function executes (ie before g.Go or go ... in other case). Otherwise step is computed when the lambda function executes
		// ie in the other go routine and at this time step may have changed as we are in a for loop.
		func(step *Step) {
			g.Go(func() error {
				return step.run(ctx, deploymentID, w.consulClient.KV(), uninstallerrc, w.shutdownCh, w.cfg, isUndeploy)
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
		deployments.LogInConsul(w.consulClient.KV(), deploymentID, fmt.Sprintf("Error '%v' happened in workflow.", err))
		return err
	}

	if len(errors) > 0 {
		uninstallerr := fmt.Errorf("%s", strings.Join(errors, " ; "))
		deployments.LogInConsul(w.consulClient.KV(), deploymentID, fmt.Sprintf("One or more error appear in unistall workflow, please check : %v", uninstallerr))
		log.Printf("One or more error appear in uninstall workflow, please check : %v", uninstallerr)
	} else {
		deployments.LogInConsul(w.consulClient.KV(), deploymentID, "Workflow ended without error")
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
		w.setDeploymentStatus(task.TargetID, deployments.DEPLOYMENT_IN_PROGRESS)
		wf, err := readWorkFlowFromConsul(w.consulClient.KV(), path.Join(consulutil.DeploymentKVPrefix, task.TargetID, "workflows/install"))
		if err != nil {
			if task.Status() == RUNNING {
				task.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(task.TargetID, deployments.DEPLOYMENT_FAILED)
			return

		}
		for _, step := range wf {
			step.SetTaskID(task)
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.WorkflowsPrefix, task.ID, step.Name), "initial")
		}
		if err = w.processWorkflow(ctx, wf, task.TargetID, false); err != nil {
			if task.Status() == RUNNING {
				task.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(task.TargetID, deployments.DEPLOYMENT_FAILED)
			return
		}
		w.setDeploymentStatus(task.TargetID, deployments.DEPLOYED)
	case UnDeploy, Purge:
		w.setDeploymentStatus(task.TargetID, deployments.UNDEPLOYMENT_IN_PROGRESS)
		wf, err := readWorkFlowFromConsul(w.consulClient.KV(), path.Join(consulutil.DeploymentKVPrefix, task.TargetID, "workflows/uninstall"))
		if err != nil {
			if task.Status() == RUNNING {
				task.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(task.TargetID, deployments.UNDEPLOYMENT_FAILED)
			return

		}
		for _, step := range wf {
			step.SetTaskID(task)
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.WorkflowsPrefix, task.ID, step.Name), "")
		}
		if err = w.processWorkflow(ctx, wf, task.TargetID, true); err != nil {
			if task.Status() == RUNNING {
				task.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(task.TargetID, deployments.UNDEPLOYMENT_FAILED)
			return
		}
		w.setDeploymentStatus(task.TargetID, deployments.UNDEPLOYED)
		if task.TaskType == Purge {
			_, err := w.consulClient.KV().DeleteTree(path.Join(consulutil.DeploymentKVPrefix, task.TargetID), nil)
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge deployment definition: %+v", task.TargetID, task.ID, err)
				task.WithStatus(FAILED)
				return
			}
			tasks, err := GetTasksIdsForTarget(w.consulClient.KV(), task.TargetID)
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", task.TargetID, task.ID, err)
				task.WithStatus(FAILED)
				return
			}
			for _, tid := range tasks {
				if tid != task.ID {
					_, err = w.consulClient.KV().DeleteTree(path.Join(consulutil.TasksPrefix, tid), nil)
					if err != nil {
						log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", task.TargetID, task.ID, err)
						task.WithStatus(FAILED)
						return
					}
				}
				_, err = w.consulClient.KV().DeleteTree(path.Join(consulutil.WorkflowsPrefix, tid), nil)
				if err != nil {
					log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", task.TargetID, task.ID, err)
					task.WithStatus(FAILED)
					return
				}
			}
			// Now cleanup ourself: mark it as done so nobody will try to run it, clear the processing lock and finally delete the task.
			task.WithStatus(DONE)
			task.releaseLock()
			_, err = w.consulClient.KV().DeleteTree(path.Join(consulutil.TasksPrefix, task.ID), nil)
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", task.TargetID, task.ID, err)
				task.WithStatus(FAILED)
				return
			}
			return
		}
	case CustomCommand:
		eventPub := events.NewPublisher(task.kv, task.TargetID)

		nodeNameKv, _, err := w.consulClient.KV().Get(path.Join(consulutil.TasksPrefix, task.ID, "node"), nil)
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to get nodeName: %+v", task.TargetID, task.ID, err)
			task.WithStatus(FAILED)
			return
		}

		commandNameKv, _, err := w.consulClient.KV().Get(path.Join(consulutil.TasksPrefix, task.ID, "name"), nil)
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to get Custom command name: %+v", task.TargetID, task.ID, err)
			task.WithStatus(FAILED)
			return
		}

		nodeName := string(nodeNameKv.Value)
		commandName := string(commandNameKv.Value)

		exec := ansible.NewExecutor(task.kv)
		if err := exec.ExecOperation(ctx, task.TargetID, nodeName, "custom."+commandName, task.ID); err != nil {
			setNodeStatus(task.kv, eventPub, task.TargetID, nodeName, "error")
			log.Printf("Deployment %q, Step %q: Sending error %v to error channel", task.TargetID, nodeName, err)
		}
	case ScaleUp:
		//eventPub := events.NewPublisher(task.kv, task.TargetId)
		w.setDeploymentStatus(task.TargetID, deployments.SCALING_IN_PROGRESS)
		wf, err := readWorkFlowFromConsul(w.consulClient.KV(), path.Join(consulutil.DeploymentKVPrefix, task.TargetID, "workflows/install"))
		if err != nil {
			if task.Status() == RUNNING {
				task.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(task.TargetID, deployments.DEPLOYMENT_FAILED)
			return

		}
		for _, step := range wf {
			step.SetTaskID(task)
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.WorkflowsPrefix, task.ID, step.Name), "")
		}
		if err = w.processWorkflow(ctx, wf, task.TargetID, false); err != nil {
			if task.Status() == RUNNING {
				task.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(task.TargetID, deployments.DEPLOYMENT_FAILED)
			return
		}
		w.setDeploymentStatus(task.TargetID, deployments.DEPLOYED)
	case ScaleDown:
		w.setDeploymentStatus(task.TargetID, deployments.SCALING_IN_PROGRESS)
		wf, err := readWorkFlowFromConsul(w.consulClient.KV(), path.Join(consulutil.DeploymentKVPrefix, task.TargetID, "workflows/uninstall"))
		if err != nil {
			if task.Status() == RUNNING {
				task.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(task.TargetID, deployments.DEPLOYMENT_FAILED)
			return

		}
		for _, step := range wf {
			step.SetTaskID(task)
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.WorkflowsPrefix, task.ID, step.Name), "")
		}
		if err = w.processWorkflow(ctx, wf, task.TargetID, true); err != nil {
			if task.Status() == RUNNING {
				task.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(task.TargetID, deployments.DEPLOYMENT_FAILED)
			return
		}

		// Cleanup
		if err = w.cleanupScaledDownNodes(task); err != nil {
			if task.Status() == RUNNING {
				task.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(task.TargetID, deployments.DEPLOYMENT_FAILED)
			return
		}
		w.setDeploymentStatus(task.TargetID, deployments.DEPLOYED)
	default:
		deployments.LogInConsul(w.consulClient.KV(), task.TargetID, fmt.Sprintf("Unknown TaskType %d (%s) for task with id %q", task.TaskType, task.TaskType.String(), task.ID))
		log.Printf("Unknown TaskType %d (%s) for task with id %q and targetId %q", task.TaskType, task.TaskType.String(), task.ID, task.TargetID)
		if task.Status() == RUNNING {
			task.WithStatus(FAILED)
		}
		return
	}
	if task.Status() == RUNNING {
		task.WithStatus(DONE)
	}
}

// cleanupScaledDownNodes removes nodes instances from Consul
func (w Worker) cleanupScaledDownNodes(task *Task) error {
	kv := w.consulClient.KV()

	nodeNameKVP, _, err := kv.Get(path.Join(consulutil.TasksPrefix, task.ID, "node"), nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	} else if nodeNameKVP == nil || len(nodeNameKVP.Value) == 0 {
		return errors.Errorf("Missing mandatory key \"node\" for task %q", task.ID)
	}
	nodeName := string(nodeNameKVP.Value)

	nodesIdsKv, _, err := kv.Get(path.Join(consulutil.TasksPrefix, task.ID, "new_instances_ids"), nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	} else if nodesIdsKv == nil || len(nodesIdsKv.Value) == 0 {
		return errors.Errorf("Missing mandatory key \"new_instances_ids\" for task %q", task.ID)
	}
	instancesIDs := strings.Split(string(nodesIdsKv.Value), ",")
	reqKv, _, err := kv.Get(path.Join(consulutil.TasksPrefix, task.ID, "req"), nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	} else if reqKv == nil {
		return errors.Errorf("Missing mandatory key \"req\" for task %q", task.ID)
	}

	linkedNodes := strings.Split(string(reqKv.Value), ",")
	return deployments.DeleteNodeStack(kv, task.TargetID, nodeName, instancesIDs, linkedNodes)

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
				log.Printf("Worker got task with id %s", task.ID)
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
		var lastIndex uint64
		for {
			kvp, qMeta, err := w.consulClient.KV().Get(path.Join(consulutil.TasksPrefix, task.ID, ".canceledFlag"), &api.QueryOptions{WaitIndex: lastIndex})

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
