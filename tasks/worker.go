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
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

type worker struct {
	workerPool   chan chan *task
	TaskChannel  chan *task
	shutdownCh   chan struct{}
	consulClient *api.Client
	cfg          config.Configuration
	runStepLock  sync.Locker
}

func newWorker(workerPool chan chan *task, shutdownCh chan struct{}, consulClient *api.Client, cfg config.Configuration) worker {
	return worker{
		workerPool:   workerPool,
		TaskChannel:  make(chan *task),
		shutdownCh:   shutdownCh,
		consulClient: consulClient,
		cfg:          cfg,
		runStepLock:  &sync.Mutex{}}
}

func (w worker) setDeploymentStatus(deploymentID string, status deployments.DeploymentStatus) {
	p := &api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "status"), Value: []byte(fmt.Sprint(status))}
	kv := w.consulClient.KV()
	kv.Put(p, nil)
}

func (w worker) processWorkflow(ctx context.Context, wfSteps []*step, deploymentID string, isUndeploy bool) error {
	events.LogEngineMessage(w.consulClient.KV(), deploymentID, "Start processing workflow")
	uninstallerrc := make(chan error)

	g, ctx := errgroup.WithContext(ctx)
	for _, s := range wfSteps {
		log.Debugf("Running step %q", s.Name)
		// The use of the function bellow is a classic gotcha with go routines. It allows to use a fixed step as it is computed when the
		// bellow function executes (ie before g.Go or go ... in other case). Otherwise step is computed when the lambda function executes
		// ie in the other go routine and at this time step may have changed as we are in a for loop.
		func(s *step) {
			g.Go(func() error {
				return s.run(ctx, deploymentID, w.consulClient.KV(), uninstallerrc, w.shutdownCh, w.cfg, isUndeploy)
			})
		}(s)
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
		events.LogEngineMessage(w.consulClient.KV(), deploymentID, fmt.Sprintf("Error '%v' happened in workflow.", err))
		return err
	}

	if len(errors) > 0 {
		uninstallerr := fmt.Errorf("%s", strings.Join(errors, " ; "))
		events.LogEngineMessage(w.consulClient.KV(), deploymentID, fmt.Sprintf("One or more error appear in unistall workflow, please check : %v", uninstallerr))
		log.Printf("One or more error appear in uninstall workflow, please check : %v", uninstallerr)
	} else {
		events.LogEngineMessage(w.consulClient.KV(), deploymentID, "Workflow ended without error")
		log.Printf("Workflow ended without error")
	}
	return nil
}

func (w worker) handleTask(t *task) {
	t.WithStatus(RUNNING)
	bgCtx := context.Background()
	ctx, cancelFunc := context.WithCancel(bgCtx)
	defer t.releaseLock()
	defer cancelFunc()
	w.monitorTaskForCancellation(ctx, cancelFunc, t)
	switch t.TaskType {
	case Deploy:
		w.setDeploymentStatus(t.TargetID, deployments.DEPLOYMENT_IN_PROGRESS)
		wf, err := readWorkFlowFromConsul(w.consulClient.KV(), path.Join(consulutil.DeploymentKVPrefix, t.TargetID, "workflows/install"))
		if err != nil {
			if t.Status() == RUNNING {
				t.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(t.TargetID, deployments.DEPLOYMENT_FAILED)
			return

		}
		for _, step := range wf {
			step.SetTaskID(t)
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.WorkflowsPrefix, t.ID, step.Name), "initial")
		}
		if err = w.processWorkflow(ctx, wf, t.TargetID, false); err != nil {
			if t.Status() == RUNNING {
				t.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(t.TargetID, deployments.DEPLOYMENT_FAILED)
			return
		}
		w.setDeploymentStatus(t.TargetID, deployments.DEPLOYED)
	case UnDeploy, Purge:
		w.setDeploymentStatus(t.TargetID, deployments.UNDEPLOYMENT_IN_PROGRESS)
		wf, err := readWorkFlowFromConsul(w.consulClient.KV(), path.Join(consulutil.DeploymentKVPrefix, t.TargetID, "workflows/uninstall"))
		if err != nil {
			if t.Status() == RUNNING {
				t.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(t.TargetID, deployments.UNDEPLOYMENT_FAILED)
			return

		}
		for _, step := range wf {
			step.SetTaskID(t)
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.WorkflowsPrefix, t.ID, step.Name), "")
		}
		if err = w.processWorkflow(ctx, wf, t.TargetID, true); err != nil {
			if t.Status() == RUNNING {
				t.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(t.TargetID, deployments.UNDEPLOYMENT_FAILED)
			return
		}
		w.setDeploymentStatus(t.TargetID, deployments.UNDEPLOYED)
		if t.TaskType == Purge {
			_, err := w.consulClient.KV().DeleteTree(path.Join(consulutil.DeploymentKVPrefix, t.TargetID), nil)
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge deployment definition: %+v", t.TargetID, t.ID, err)
				t.WithStatus(FAILED)
				return
			}
			tasks, err := GetTasksIdsForTarget(w.consulClient.KV(), t.TargetID)
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", t.TargetID, t.ID, err)
				t.WithStatus(FAILED)
				return
			}
			for _, tid := range tasks {
				if tid != t.ID {
					_, err = w.consulClient.KV().DeleteTree(path.Join(consulutil.TasksPrefix, tid), nil)
					if err != nil {
						log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", t.TargetID, t.ID, err)
						t.WithStatus(FAILED)
						return
					}
				}
				_, err = w.consulClient.KV().DeleteTree(path.Join(consulutil.WorkflowsPrefix, tid), nil)
				if err != nil {
					log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", t.TargetID, t.ID, err)
					t.WithStatus(FAILED)
					return
				}
			}
			// Now cleanup ourself: mark it as done so nobody will try to run it, clear the processing lock and finally delete the task.
			t.WithStatus(DONE)
			t.releaseLock()
			_, err = w.consulClient.KV().DeleteTree(path.Join(consulutil.TasksPrefix, t.ID), nil)
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", t.TargetID, t.ID, err)
				t.WithStatus(FAILED)
				return
			}
			return
		}
	case CustomCommand:
		eventPub := events.NewPublisher(t.kv, t.TargetID)

		nodeNameKv, _, err := w.consulClient.KV().Get(path.Join(consulutil.TasksPrefix, t.ID, "node"), nil)
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to get nodeName: %+v", t.TargetID, t.ID, err)
			t.WithStatus(FAILED)
			return
		}

		commandNameKv, _, err := w.consulClient.KV().Get(path.Join(consulutil.TasksPrefix, t.ID, "name"), nil)
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to get Custom command name: %+v", t.TargetID, t.ID, err)
			t.WithStatus(FAILED)
			return
		}

		nodeName := string(nodeNameKv.Value)
		commandName := string(commandNameKv.Value)

		exec := ansible.NewExecutor(t.kv)
		if err := exec.ExecOperation(ctx, t.TargetID, nodeName, "custom."+commandName, t.ID); err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Command execution failed for node %q: %+v", t.TargetID, t.ID, nodeName, err)
			err = setNodeStatus(t.kv, eventPub, t.TargetID, nodeName, tosca.NodeStateError.String())
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to set status for node %q: %+v", t.TargetID, t.ID, nodeName, err)
			}
			t.WithStatus(FAILED)
			return
		}
	case ScaleUp:
		//eventPub := events.NewPublisher(task.kv, task.TargetId)
		w.setDeploymentStatus(t.TargetID, deployments.SCALING_IN_PROGRESS)
		wf, err := readWorkFlowFromConsul(w.consulClient.KV(), path.Join(consulutil.DeploymentKVPrefix, t.TargetID, "workflows/install"))
		if err != nil {
			if t.Status() == RUNNING {
				t.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(t.TargetID, deployments.DEPLOYMENT_FAILED)
			return

		}
		for _, step := range wf {
			step.SetTaskID(t)
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.WorkflowsPrefix, t.ID, step.Name), "")
		}
		if err = w.processWorkflow(ctx, wf, t.TargetID, false); err != nil {
			if t.Status() == RUNNING {
				t.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(t.TargetID, deployments.DEPLOYMENT_FAILED)
			return
		}
		w.setDeploymentStatus(t.TargetID, deployments.DEPLOYED)
	case ScaleDown:
		w.setDeploymentStatus(t.TargetID, deployments.SCALING_IN_PROGRESS)
		wf, err := readWorkFlowFromConsul(w.consulClient.KV(), path.Join(consulutil.DeploymentKVPrefix, t.TargetID, "workflows/uninstall"))
		if err != nil {
			if t.Status() == RUNNING {
				t.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(t.TargetID, deployments.DEPLOYMENT_FAILED)
			return

		}
		for _, step := range wf {
			step.SetTaskID(t)
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.WorkflowsPrefix, t.ID, step.Name), "")
		}
		if err = w.processWorkflow(ctx, wf, t.TargetID, true); err != nil {
			if t.Status() == RUNNING {
				t.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(t.TargetID, deployments.DEPLOYMENT_FAILED)
			return
		}

		// Cleanup
		if err = w.cleanupScaledDownNodes(t); err != nil {
			if t.Status() == RUNNING {
				t.WithStatus(FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(t.TargetID, deployments.DEPLOYMENT_FAILED)
			return
		}
		w.setDeploymentStatus(t.TargetID, deployments.DEPLOYED)
	default:
		events.LogEngineMessage(w.consulClient.KV(), t.TargetID, fmt.Sprintf("Unknown TaskType %d (%s) for task with id %q", t.TaskType, t.TaskType.String(), t.ID))
		log.Printf("Unknown TaskType %d (%s) for task with id %q and targetId %q", t.TaskType, t.TaskType.String(), t.ID, t.TargetID)
		if t.Status() == RUNNING {
			t.WithStatus(FAILED)
		}
		return
	}
	if t.Status() == RUNNING {
		t.WithStatus(DONE)
	}
}

// cleanupScaledDownNodes removes nodes instances from Consul
func (w worker) cleanupScaledDownNodes(t *task) error {
	kv := w.consulClient.KV()

	nodeNameKVP, _, err := kv.Get(path.Join(consulutil.TasksPrefix, t.ID, "node"), nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	} else if nodeNameKVP == nil || len(nodeNameKVP.Value) == 0 {
		return errors.Errorf("Missing mandatory key \"node\" for task %q", t.ID)
	}
	nodeName := string(nodeNameKVP.Value)

	nodesIdsKv, _, err := kv.Get(path.Join(consulutil.TasksPrefix, t.ID, "new_instances_ids"), nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	} else if nodesIdsKv == nil || len(nodesIdsKv.Value) == 0 {
		return errors.Errorf("Missing mandatory key \"new_instances_ids\" for task %q", t.ID)
	}
	instancesIDs := strings.Split(string(nodesIdsKv.Value), ",")
	reqKv, _, err := kv.Get(path.Join(consulutil.TasksPrefix, t.ID, "req"), nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	} else if reqKv == nil {
		return errors.Errorf("Missing mandatory key \"req\" for task %q", t.ID)
	}

	linkedNodes := strings.Split(string(reqKv.Value), ",")
	return deployments.DeleteNodeStack(kv, t.TargetID, nodeName, instancesIDs, linkedNodes)

}

// Start method starts the run loop for the worker, listening for a quit channel in
// case we need to stop it
func (w worker) Start() {
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

func (w worker) monitorTaskForCancellation(ctx context.Context, cancelFunc context.CancelFunc, t *task) {
	go func() {
		var lastIndex uint64
		for {
			kvp, qMeta, err := w.consulClient.KV().Get(path.Join(consulutil.TasksPrefix, t.ID, ".canceledFlag"), &api.QueryOptions{WaitIndex: lastIndex})

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
					t.WithStatus(CANCELED)
					cancelFunc()
					return
				}
			}
		}
	}()
}
