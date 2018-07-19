// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/armon/go-metrics"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/helper/metricsutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
	"github.com/ystia/yorc/prov/operations"
	"github.com/ystia/yorc/registry"
		"github.com/ystia/yorc/tosca"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"github.com/ystia/yorc/tasks"
)

type worker struct {
	workerPool   chan chan *task
	TaskChannel  chan *task
	shutdownCh   chan struct{}
	consulClient *api.Client
	cfg          config.Configuration
}

func newWorker(workerPool chan chan *task, shutdownCh chan struct{}, consulClient *api.Client, cfg config.Configuration) worker {
	return worker{
		workerPool:   workerPool,
		TaskChannel:  make(chan *task),
		shutdownCh:   shutdownCh,
		consulClient: consulClient,
		cfg:          cfg,
	}
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
				log.Debugf("Worker got task with id %s", task.ID)
				w.handleTask(task)

			case <-w.shutdownCh:
				// we have received a signal to stop
				log.Printf("Worker received shutdown signal. Exiting...")
				return
			}
		}
	}()
}

func setNodeStatus(ctx context.Context, kv *api.KV, taskID, deploymentID, nodeName, status string) error {
	instancesIDs, err := tasks.GetInstances(kv, taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}

	for _, id := range instancesIDs {
		// Publish status change event
		err := deployments.SetInstanceStateStringWithContextualLogs(ctx, kv, deploymentID, nodeName, id, status)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetOperationExecutor(kv *api.KV, deploymentID, artifact string) (prov.OperationExecutor, error) {
	reg := registry.GetRegistry()

	exec, originalErr := reg.GetOperationExecutor(artifact)
	if originalErr == nil {
		return exec, nil
	}
	// Try to get an executor for artifact parent type but return the original error if we do not found any executors
	parentArt, err := deployments.GetParentType(kv, deploymentID, artifact)
	if err != nil {
		return nil, err
	}
	if parentArt != "" {
		exec, err := GetOperationExecutor(kv, deploymentID, parentArt)
		if err == nil {
			return exec, nil
		}
	}
	return nil, originalErr
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
					log.Debugln("[TASK MONITOR] Task cancellation requested.")
					t.WithStatus(ctx, tasks.TaskStatusCANCELED)
					cancelFunc()
					return
				}
			}
		}
	}()
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
			err = deployments.DeleteRelationshipInstance(kv, t.TargetID, node, instance)
			if err != nil {
				return err
			}
		}

	}
	return nil
}

func (w worker) setDeploymentStatus(ctx context.Context, deploymentID string, status deployments.DeploymentStatus) {
	p := &api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "status"), Value: []byte(fmt.Sprint(status))}
	kv := w.consulClient.KV()
	kv.Put(p, nil)
	events.PublishAndLogDeploymentStatusChange(ctx, kv, deploymentID, strings.ToLower(status.String()))
}

func (w worker) handleTask(t *task) {
	if t.Status() == tasks.TaskStatusINITIAL {
		metrics.MeasureSince([]string{"tasks", "wait"}, t.creationDate)
	}
	kv := w.consulClient.KV()

	// Fill log optional fields for log registration
	wfName, _ := tasks.GetTaskData(kv, t.ID, "workflowName")
	logOptFields := events.LogOptionalFields{
		events.WorkFlowID:  wfName,
		events.ExecutionID: t.ID,
	}
	ctx := events.NewContext(context.Background(), logOptFields)

	ctx, cancelFunc := context.WithCancel(ctx)
	defer t.releaseLock()
	defer cancelFunc()

	t.WithStatus(ctx, tasks.TaskStatusRUNNING)

	w.monitorTaskForCancellation(ctx, cancelFunc, t)
	defer func(t *task, start time.Time) {
		metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"task", t.TargetID, t.TaskType.String(), t.Status().String()}), 1)
		metrics.MeasureSince(metricsutil.CleanupMetricKey([]string{"task", t.TargetID, t.TaskType.String()}), start)
	}(t, time.Now())
	switch t.TaskType {
	case tasks.TaskTypeDeploy:
		w.setDeploymentStatus(ctx, t.TargetID, deployments.DEPLOYMENT_IN_PROGRESS)
		err := w.runWorkflowStep(ctx, t)
		if err != nil {
			w.setDeploymentStatus(ctx, t.TargetID, deployments.DEPLOYMENT_FAILED)
			return
		}
		w.setDeploymentStatus(ctx, t.TargetID, deployments.DEPLOYED)
	case tasks.TaskTypeUnDeploy, tasks.TaskTypePurge:
		status, err := deployments.GetDeploymentStatus(kv, t.TargetID)
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to get deployment status: %+v", t.TargetID, t.ID, err)
			t.WithStatus(ctx, tasks.TaskStatusFAILED)
			return
		}
		if status != deployments.UNDEPLOYED {
			w.setDeploymentStatus(ctx, t.TargetID, deployments.UNDEPLOYMENT_IN_PROGRESS)
			err := w.runWorkflowStep(ctx, t)
			if err != nil {
				w.setDeploymentStatus(ctx, t.TargetID, deployments.UNDEPLOYMENT_FAILED)
				return
			}
			w.setDeploymentStatus(ctx, t.TargetID, deployments.UNDEPLOYED)
		}

		if t.TaskType == tasks.TaskTypePurge {
			_, err := kv.DeleteTree(path.Join(consulutil.DeploymentKVPrefix, t.TargetID), nil)
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge deployment definition: %+v", t.TargetID, t.ID, err)
				t.WithStatus(ctx, tasks.TaskStatusFAILED)
				return
			}
			tasksList, err := tasks.GetTasksIdsForTarget(kv, t.TargetID)
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", t.TargetID, t.ID, err)
				t.WithStatus(ctx, tasks.TaskStatusFAILED)
				return
			}
			for _, tid := range tasksList {
				if tid != t.ID {
					_, err = kv.DeleteTree(path.Join(consulutil.TasksPrefix, tid), nil)
					if err != nil {
						log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", t.TargetID, t.ID, err)
						t.WithStatus(ctx, tasks.TaskStatusFAILED)
						return
					}
				}
				_, err = kv.DeleteTree(path.Join(consulutil.WorkflowsPrefix, tid), nil)
				if err != nil {
					log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", t.TargetID, t.ID, err)
					t.WithStatus(ctx, tasks.TaskStatusFAILED)
					return
				}
			}
			// Delete events tree corresponding to the deployment task
			_, err = kv.DeleteTree(path.Join(consulutil.EventsPrefix, t.TargetID), nil)
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge events: %+v", t.TargetID, t.ID, err)
				t.WithStatus(ctx, tasks.TaskStatusFAILED)
				return
			}
			// Delete logs tree corresponding to the deployment task
			_, err = kv.DeleteTree(path.Join(consulutil.LogsPrefix, t.TargetID), nil)
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge logs: %+v", t.TargetID, t.ID, err)
				t.WithStatus(ctx, tasks.TaskStatusFAILED)
				return
			}
			err = os.RemoveAll(filepath.Join(w.cfg.WorkingDirectory, "deployments", t.TargetID))
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", t.TargetID, t.ID, err)
				t.WithStatus(ctx, tasks.TaskStatusFAILED)
				return
			}
			// Now cleanup ourself: mark it as done so nobody will try to run it, clear the processing lock and finally delete the task.
			t.WithStatus(ctx, tasks.TaskStatusDONE)
			t.releaseLock()
			_, err = kv.DeleteTree(path.Join(consulutil.TasksPrefix, t.ID), nil)
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", t.TargetID, t.ID, err)
				t.WithStatus(ctx, tasks.TaskStatusFAILED)
				return
			}
			return
		}
	case tasks.TaskTypeCustomCommand:
		w.runCustomCommand(t, ctx)
	case tasks.TaskTypeScaleOut:
		w.setDeploymentStatus(ctx, t.TargetID, deployments.SCALING_IN_PROGRESS)
		err := w.runWorkflowStep(ctx, t)
		if err != nil {
			w.setDeploymentStatus(ctx, t.TargetID, deployments.DEPLOYMENT_FAILED)
			return
		}
		w.setDeploymentStatus(ctx, t.TargetID, deployments.DEPLOYED)
	case tasks.TaskTypeScaleIn:
		w.setDeploymentStatus(ctx, t.TargetID, deployments.SCALING_IN_PROGRESS)
		err := w.runWorkflowStep(ctx, t)
		if err != nil {
			w.setDeploymentStatus(ctx, t.TargetID, deployments.DEPLOYMENT_FAILED)
			return
		}

		// Cleanup
		if err = w.cleanupScaledDownNodes(t); err != nil {
			if t.Status() == tasks.TaskStatusRUNNING {
				t.WithStatus(ctx, tasks.TaskStatusFAILED)
			}
			log.Printf("%v. Aborting", err)
			w.setDeploymentStatus(ctx, t.TargetID, deployments.DEPLOYMENT_FAILED)
			return
		}
		w.setDeploymentStatus(ctx, t.TargetID, deployments.DEPLOYED)
	case tasks.TaskTypeCustomWorkflow:
		//continueOnError, err := tasks.GetTaskData(kv, t.ID, "continueOnError")
		//if err != nil {
		//	log.Printf("Deployment id: %q, Task id: %q Failed: %v", t.TargetID, t.ID, err)
		//	log.Debugf("%+v", err)
		//	t.WithStatus(ctx, TaskStatusFAILED)
		//	return
		//}
		//bypassErrors, err := strconv.ParseBool(continueOnError)
		//if err != nil {
		//	log.Printf("Deployment id: %q, Task id: %q Failed to parse continueOnError parameter: %v", t.TargetID, t.ID, err)
		//	log.Debugf("%+v", err)
		//	t.WithStatus(ctx, TaskStatusFAILED)
		//	return
		//}
		err := w.runWorkflowStep(ctx, t)
		if err != nil {
			return
		}
	case tasks.TaskTypeQuery:
		w.runQuery(t, ctx)
	default:
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, t.TargetID).RegisterAsString(fmt.Sprintf("Unknown TaskType %d (%s) for task with id %q", t.TaskType, t.TaskType.String(), t.ID))
		log.Printf("Unknown TaskType %d (%s) for task with id %q and targetId %q", t.TaskType, t.TaskType.String(), t.ID, t.TargetID)
		if t.Status() == tasks.TaskStatusRUNNING {
			t.WithStatus(ctx, tasks.TaskStatusFAILED)
		}
		return
	}
	if t.Status() == tasks.TaskStatusRUNNING {
		t.WithStatus(ctx, tasks.TaskStatusDONE)
	}
}

func (w worker) runCustomCommand(t *task, ctx context.Context) {
	kv := w.consulClient.KV()
	commandNameKv, _, err := kv.Get(path.Join(consulutil.TasksPrefix, t.ID, "commandName"), nil)
	if err != nil {
		log.Printf("Deployment id: %q, Task id: %q, Failed to get Custom command name: %+v", t.TargetID, t.ID, err)
		t.WithStatus(ctx, tasks.TaskStatusFAILED)
		return
	}
	if commandNameKv == nil || len(commandNameKv.Value) == 0 {
		log.Printf("Deployment id: %q, Task id: %q, Missing commandName attribute for custom command task", t.TargetID, t.ID)
		t.WithStatus(ctx, tasks.TaskStatusFAILED)
		return
	}

	nodes, err := tasks.GetTaskRelatedNodes(kv, t.ID)
	if err != nil {
		log.Printf("Deployment id: %q, Task id: %q, Failed to get Custom command node: %+v", t.TargetID, t.ID, err)
		t.WithStatus(ctx, tasks.TaskStatusFAILED)
		return
	}
	if len(nodes) != 1 {
		log.Printf("Deployment id: %q, Task id: %q, Expecting custom command task to be related to \"1\" node while it is actually related to \"%d\" nodes", t.TargetID, t.ID, len(nodes))
		t.WithStatus(ctx, tasks.TaskStatusFAILED)
		return
	}

	nodeName := nodes[0]
	commandName := string(commandNameKv.Value)
	nodeType, err := deployments.GetNodeType(w.consulClient.KV(), t.TargetID, nodeName)
	if err != nil {
		log.Printf("Deployment id: %q, Task id: %q, Failed to get Custom command node type: %+v", t.TargetID, t.ID, err)
		t.WithStatus(ctx, tasks.TaskStatusFAILED)
		return
	}
	op, err := operations.GetOperation(ctx, kv, t.TargetID, nodeName, "custom."+commandName, "", "")
	if err != nil {
		log.Printf("Deployment id: %q, Task id: %q, Command execution failed for node %q: %+v", t.TargetID, t.ID, nodeName, err)
		err = setNodeStatus(ctx, t.kv, t.ID, t.TargetID, nodeName, tosca.NodeStateError.String())
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to set status for node %q: %+v", t.TargetID, t.ID, nodeName, err)
		}
		t.WithStatus(ctx, tasks.TaskStatusFAILED)
		return
	}

	exec, err := GetOperationExecutor(kv, t.TargetID, op.ImplementationArtifact)
	if err != nil {
		log.Printf("Deployment id: %q, Task id: %q, Command execution failed for node %q: %+v", t.TargetID, t.ID, nodeName, err)
		err = setNodeStatus(ctx, t.kv, t.ID, t.TargetID, nodeName, tosca.NodeStateError.String())
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to set status for node %q: %+v", t.TargetID, t.ID, nodeName, err)
		}
		t.WithStatus(ctx, tasks.TaskStatusFAILED)
		return
	}
	err = func() error {
		defer metrics.MeasureSince(metricsutil.CleanupMetricKey([]string{"executor", "operation", t.TargetID, nodeType, op.Name}), time.Now())
		return exec.ExecOperation(ctx, w.cfg, t.ID, t.TargetID, nodeName, op)
	}()
	if err != nil {
		metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"executor", "operation", t.TargetID, nodeType, op.Name, "failures"}), 1)
		log.Printf("Deployment id: %q, Task id: %q, Command execution failed for node %q: %+v", t.TargetID, t.ID, nodeName, err)
		err = setNodeStatus(ctx, t.kv, t.ID, t.TargetID, nodeName, tosca.NodeStateError.String())
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to set status for node %q: %+v", t.TargetID, t.ID, nodeName, err)
		}
		t.WithStatus(ctx, tasks.TaskStatusFAILED)
		return
	}
	metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"executor", "operation", t.TargetID, nodeType, op.Name, "successes"}), 1)
}

func (w worker) runQuery(t *task, ctx context.Context) {
	kv := w.consulClient.KV()
	split := strings.Split(t.TargetID, ":")
	if len(split) != 2 {
		log.Printf("Query Task (id: %q): unexpected format for targetID: %q", t.ID, t.TargetID)
		t.WithStatus(ctx, tasks.TaskStatusFAILED)
		return
	}
	query := split[0]
	target := split[1]

	switch query {
	case "infra_usage":
		var reg = registry.GetRegistry()
		collector, err := reg.GetInfraUsageCollector(target)
		if err != nil {
			log.Printf("Query Task id: %q Failed to retrieve target type: %v", t.ID, err)
			log.Debugf("%+v", err)
			t.WithStatus(ctx, tasks.TaskStatusFAILED)
			return
		}
		res, err := collector.GetUsageInfo(ctx, w.cfg, t.ID, target)
		if err != nil {
			log.Printf("Query Task id: %q Failed to run query: %v", t.ID, err)
			log.Debugf("%+v", err)
			t.WithStatus(ctx, tasks.TaskStatusFAILED)
			return
		}

		// store resultSet as a JSON
		resultPrefix := path.Join(consulutil.TasksPrefix, t.ID, "resultSet")
		if res != nil {
			jsonRes, err := json.Marshal(res)
			if err != nil {
				log.Printf("Failed to marshal infra usage info [%+v]: due to error:%+v", res, err)
				log.Debugf("%+v", err)
				t.WithStatus(ctx, tasks.TaskStatusFAILED)
				return
			}
			kvPair := &api.KVPair{Key: resultPrefix, Value: jsonRes}
			if _, err := kv.Put(kvPair, nil); err != nil {
				log.Printf("Query Task id: %q Failed to store result: %v", t.ID, errors.Wrap(err, consulutil.ConsulGenericErrMsg))
				log.Debugf("%+v", err)
				t.WithStatus(ctx, tasks.TaskStatusFAILED)
				return
			}
		}
	default:
		mess := fmt.Sprintf("Unknown query: %q for Task with id %q", query, t.ID)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, t.TargetID).RegisterAsString(mess)
		log.Printf(mess)
		if t.Status() == tasks.TaskStatusRUNNING {
			t.WithStatus(ctx, tasks.TaskStatusFAILED)
		}
		return
	}
}

func (w worker) runWorkflowStep(ctx context.Context, t *task) error {
	// runWorkflows
	// processWorkflow
	// Need to retrieve the related step from Consul with stepName and workflow
	//return s.run(ctx, deploymentID, w.consulClient.KV(), uninstallerrc, w.shutdownCh, w.cfg, bypassErrors, workflowName, w)
	return nil
}
