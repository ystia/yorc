package workflow

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"

	"strconv"

	"github.com/hashicorp/consul/api"
	"golang.org/x/sync/errgroup"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/ansible"
	"novaforge.bull.com/starlings-janus/janus/tasks"
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

func (w worker) processWorkflow(ctx context.Context, workflowName string, wfSteps []*step, deploymentID string, bypassErrors bool) error {
	events.LogEngineMessage(w.consulClient.KV(), deploymentID, fmt.Sprintf("Start processing workflow %q", workflowName))
	uninstallerrc := make(chan error)

	g, ctx := errgroup.WithContext(ctx)
	for _, s := range wfSteps {
		log.Debugf("Running step %q", s.Name)
		// The use of the function bellow is a classic gotcha with go routines. It allows to use a fixed step as it is computed when the
		// bellow function executes (ie before g.Go or go ... in other case). Otherwise step is computed when the lambda function executes
		// ie in the other go routine and at this time step may have changed as we are in a for loop.
		func(s *step) {
			g.Go(func() error {
				return s.run(ctx, deploymentID, w.consulClient.KV(), uninstallerrc, w.shutdownCh, w.cfg, bypassErrors)
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
		events.LogEngineMessage(w.consulClient.KV(), deploymentID, fmt.Sprintf("Error '%v' happened in workflow %q.", err, workflowName))
		return err
	}

	if len(errors) > 0 {
		uninstallerr := fmt.Errorf("%s", strings.Join(errors, " ; "))
		events.LogEngineMessage(w.consulClient.KV(), deploymentID, fmt.Sprintf("One or more error appear in workflow %q, please check : %v", workflowName, uninstallerr))
		log.Printf("One or more error appear workflow %q, please check : %v", workflowName, uninstallerr)
	} else {
		events.LogEngineMessage(w.consulClient.KV(), deploymentID, fmt.Sprintf("Workflow %q ended without error", workflowName))
		log.Printf("Workflow %q ended without error", workflowName)
	}
	return nil
}

func (w worker) handleTask(t *task) {
	t.WithStatus(tasks.RUNNING)
	bgCtx := context.Background()
	ctx, cancelFunc := context.WithCancel(bgCtx)
	defer t.releaseLock()
	defer cancelFunc()
	w.monitorTaskForCancellation(ctx, cancelFunc, t)
	switch t.TaskType {
	case tasks.Deploy:
		w.setDeploymentStatus(t.TargetID, deployments.DEPLOYMENT_IN_PROGRESS)
		err := w.runWorkflows(ctx, t, []string{"install"}, false)
		if err != nil {
			w.setDeploymentStatus(t.TargetID, deployments.DEPLOYMENT_FAILED)
			return
		}
		w.setDeploymentStatus(t.TargetID, deployments.DEPLOYED)
	case tasks.UnDeploy, tasks.Purge:
		w.setDeploymentStatus(t.TargetID, deployments.UNDEPLOYMENT_IN_PROGRESS)
		err := w.runWorkflows(ctx, t, []string{"uninstall"}, true)
		if err != nil {
			w.setDeploymentStatus(t.TargetID, deployments.UNDEPLOYMENT_FAILED)
			return
		}
		w.setDeploymentStatus(t.TargetID, deployments.UNDEPLOYED)
		if t.TaskType == tasks.Purge {
			_, err := w.consulClient.KV().DeleteTree(path.Join(consulutil.DeploymentKVPrefix, t.TargetID), nil)
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge deployment definition: %+v", t.TargetID, t.ID, err)
				t.WithStatus(tasks.FAILED)
				return
			}
			tasksList, err := tasks.GetTasksIdsForTarget(w.consulClient.KV(), t.TargetID)
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", t.TargetID, t.ID, err)
				t.WithStatus(tasks.FAILED)
				return
			}
			for _, tid := range tasksList {
				if tid != t.ID {
					_, err = w.consulClient.KV().DeleteTree(path.Join(consulutil.TasksPrefix, tid), nil)
					if err != nil {
						log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", t.TargetID, t.ID, err)
						t.WithStatus(tasks.FAILED)
						return
					}
				}
				_, err = w.consulClient.KV().DeleteTree(path.Join(consulutil.WorkflowsPrefix, tid), nil)
				if err != nil {
					log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", t.TargetID, t.ID, err)
					t.WithStatus(tasks.FAILED)
					return
				}
			}
			// Now cleanup ourself: mark it as done so nobody will try to run it, clear the processing lock and finally delete the task.
			t.WithStatus(tasks.DONE)
			t.releaseLock()
			_, err = w.consulClient.KV().DeleteTree(path.Join(consulutil.TasksPrefix, t.ID), nil)
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", t.TargetID, t.ID, err)
				t.WithStatus(tasks.FAILED)
				return
			}
			return
		}
	case tasks.CustomCommand:
		eventPub := events.NewPublisher(t.kv, t.TargetID)

		commandNameKv, _, err := w.consulClient.KV().Get(path.Join(consulutil.TasksPrefix, t.ID, "commandName"), nil)
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to get Custom command name: %+v", t.TargetID, t.ID, err)
			t.WithStatus(tasks.FAILED)
			return
		}
		if commandNameKv == nil || len(commandNameKv.Value) == 0 {
			log.Printf("Deployment id: %q, Task id: %q, Missing commandName attribute for custom command task", t.TargetID, t.ID)
			t.WithStatus(tasks.FAILED)
			return
		}

		nodes, err := tasks.GetTaskRelatedNodes(w.consulClient.KV(), t.ID)
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to get Custom command node: %+v", t.TargetID, t.ID, err)
			t.WithStatus(tasks.FAILED)
			return
		}
		if len(nodes) != 1 {
			log.Printf("Deployment id: %q, Task id: %q, Expecting custom command task to be related to \"1\" node while it is actually related to \"%d\" nodes", t.TargetID, t.ID, len(nodes))
			t.WithStatus(tasks.FAILED)
			return
		}

		nodeName := nodes[0]
		commandName := string(commandNameKv.Value)

		exec := ansible.NewExecutor()
		if err := exec.ExecOperation(ctx, w.consulClient.KV(), w.cfg, t.ID, t.TargetID, nodeName, "custom."+commandName); err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Command execution failed for node %q: %+v", t.TargetID, t.ID, nodeName, err)
			err = setNodeStatus(t.kv, eventPub, t.ID, t.TargetID, nodeName, tosca.NodeStateError.String())
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to set status for node %q: %+v", t.TargetID, t.ID, nodeName, err)
			}
			t.WithStatus(tasks.FAILED)
			return
		}
	case tasks.ScaleUp:
		//eventPub := events.NewPublisher(task.kv, task.TargetId)
		w.setDeploymentStatus(t.TargetID, deployments.SCALING_IN_PROGRESS)

		err := w.runWorkflows(ctx, t, []string{"install"}, false)
		if err != nil {
			w.setDeploymentStatus(t.TargetID, deployments.DEPLOYMENT_FAILED)
			return
		}
		w.setDeploymentStatus(t.TargetID, deployments.DEPLOYED)
	case tasks.ScaleDown:
		w.setDeploymentStatus(t.TargetID, deployments.SCALING_IN_PROGRESS)
		err := w.runWorkflows(ctx, t, []string{"uninstall"}, true)
		if err != nil {
			w.setDeploymentStatus(t.TargetID, deployments.DEPLOYMENT_FAILED)
			return
		}

		// Cleanup
		if err = w.cleanupScaledDownNodes(t); err != nil {
			if t.Status() == tasks.RUNNING {
				t.WithStatus(tasks.FAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(t.TargetID, deployments.DEPLOYMENT_FAILED)
			return
		}
		w.setDeploymentStatus(t.TargetID, deployments.DEPLOYED)
	case tasks.CustomWorkflow:
		wfName, err := tasks.GetTaskData(w.consulClient.KV(), t.ID, "workflowName")
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q Failed: %v", t.TargetID, t.ID, err)
			log.Debugf("%+v", err)
			t.WithStatus(tasks.FAILED)
			return
		}
		continueOnError, err := tasks.GetTaskData(w.consulClient.KV(), t.ID, "continueOnError")
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q Failed: %v", t.TargetID, t.ID, err)
			log.Debugf("%+v", err)
			t.WithStatus(tasks.FAILED)
			return
		}
		bypassErrors, err := strconv.ParseBool(continueOnError)
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q Failed to parse continueOnError parameter: %v", t.TargetID, t.ID, err)
			log.Debugf("%+v", err)
			t.WithStatus(tasks.FAILED)
			return
		}
		err = w.runWorkflows(ctx, t, strings.Split(wfName, ","), bypassErrors)
		if err != nil {
			return
		}
	default:
		events.LogEngineMessage(w.consulClient.KV(), t.TargetID, fmt.Sprintf("Unknown TaskType %d (%s) for task with id %q", t.TaskType, t.TaskType.String(), t.ID))
		log.Printf("Unknown TaskType %d (%s) for task with id %q and targetId %q", t.TaskType, t.TaskType.String(), t.ID, t.TargetID)
		if t.Status() == tasks.RUNNING {
			t.WithStatus(tasks.FAILED)
		}
		return
	}
	if t.Status() == tasks.RUNNING {
		t.WithStatus(tasks.DONE)
	}
}

// cleanupScaledDownNodes removes nodes instances from Consul
func (w worker) cleanupScaledDownNodes(t *task) error {
	kv := w.consulClient.KV()

	nodes, err := tasks.GetTaskRelatedNodes(kv, t.ID)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		var instances []string
		instances, err = tasks.GetInstances(kv, t.ID, t.TargetID, node)
		if err != nil {
			return err
		}
		for _, instance := range instances {
			err = deployments.DeleteInstance(kv, t.TargetID, node, instance)
			if err != nil {
				return err
			}
		}

	}
	return nil
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
					t.WithStatus(tasks.CANCELED)
					cancelFunc()
					return
				}
			}
		}
	}()
}

func (w worker) runWorkflows(ctx context.Context, t *task, workflows []string, bypassErrors bool) error {
	kv := w.consulClient.KV()
	for _, workflow := range workflows {
		wf, err := readWorkFlowFromConsul(kv, path.Join(consulutil.DeploymentKVPrefix, t.TargetID, path.Join("workflows", workflow)))
		if err != nil {
			if t.Status() == tasks.RUNNING {
				t.WithStatus(tasks.FAILED)
			}
			log.Printf("%v. Aborting", err)
			return err

		}
		for _, step := range wf {
			step.SetTaskID(t)
			consulutil.StoreConsulKeyAsString(path.Join(consulutil.WorkflowsPrefix, t.ID, step.Name), "")
		}
		if err = w.processWorkflow(ctx, workflow, wf, t.TargetID, bypassErrors); err != nil {
			if t.Status() == tasks.RUNNING {
				t.WithStatus(tasks.FAILED)
			}
			log.Printf("%v. Aborting", err)
			return err
		}
	}
	return nil
}
