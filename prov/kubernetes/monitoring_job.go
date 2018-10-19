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

package kubernetes

import (
	"context"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/tasks/collector"
)

type actionOperator struct {
	clientset kubernetes.Interface
}

func (o *actionOperator) ExecAction(ctx context.Context, cfg config.Configuration, taskID, deploymentID string, action *prov.Action) (bool, error) {
	opName, ok := action.Data["operationName"]
	if !ok {
		return true, errors.New(`missing mandatory parameter "operationName" in monitoring action`)
	}
	nodeName, ok := action.Data["nodeName"]
	if !ok {
		return true, errors.New(`missing mandatory parameter "nodeName" in monitoring action`)
	}

	originalWFName, ok := action.Data["originalWorkflowName"]
	if !ok {
		return true, errors.New(`missing mandatory parameter "originalWorkflowName" in monitoring action`)
	}

	originalTaskID, ok := action.Data["originalTaskID"]
	if !ok {
		return true, errors.New(`missing mandatory parameter "originalTaskID" in monitoring action`)
	}

	ctx, err := setupExecLogsContextWithWF(ctx, nodeName, opName, originalWFName, originalTaskID)
	if err != nil {
		return true, err
	}

	if o.clientset == nil {
		o.clientset, err = initClientSet(cfg)
		if err != nil {
			return false, err
		}
	}

	jobID, ok := action.Data["jobID"]
	if !ok {
		return true, errors.New(`missing mandatory parameter "jobID" in monitoring action`)
	}
	namespace, ok := action.Data["namespace"]
	if !ok {
		return true, errors.New(`missing mandatory parameter "namespace" in monitoring action`)
	}

	stepName, ok := action.Data["stepName"]
	if !ok {
		return true, errors.New(`missing mandatory parameter "stepName" in monitoring action`)
	}

	deregister, err := o.monitorJob(ctx, cfg, deploymentID, nodeName, originalTaskID, stepName, namespace, jobID, action)
	if err != nil {
		err := o.endJob(ctx, cfg, originalTaskID, jobID, stepName, action, true)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, deploymentID).
				Registerf("fatal error encountered during job monitoring, we stop to monitor it but some resources may have to be deleted manually on the Kubernetes cluster.")

		}
	}
	return deregister, err
}

func (o *actionOperator) monitorJob(ctx context.Context, cfg config.Configuration, deploymentID, nodeName, originalTaskID, stepName, namespace, jobID string, action *prov.Action) (bool, error) {
	job, err := o.clientset.BatchV1().Jobs(namespace).Get(jobID, metav1.GetOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "can not retrieve job %q from node %q", jobID, nodeName)
	}

	publishJobLogs(ctx, cfg, o.clientset, deploymentID, namespace, job.Name, action)

	if job.Status.Active == 0 && (job.Status.Succeeded != 0 || job.Status.Failed != 0) {

		deleteForeground := metav1.DeletePropagationForeground
		err = o.clientset.BatchV1().Jobs(namespace).Delete(jobID, &metav1.DeleteOptions{PropagationPolicy: &deleteForeground})
		if err != nil {
			return true, errors.Wrapf(err, "failed to delete completed job %q", jobID)
		}

		if job.Status.Succeeded < *job.Spec.Completions {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, deploymentID).Registerf("job failed: succeeded pods: %d, failed pods: %d, requested completions: %d", job.Status.Succeeded, job.Status.Failed, *job.Spec.Completions)
			// Just set the step status as error but from the action pov it is not an error
			o.endJob(ctx, cfg, originalTaskID, jobID, stepName, action, true)
			return true, nil
		}
		o.endJob(ctx, cfg, originalTaskID, jobID, stepName, action, false)
		return true, nil
	}
	return false, nil
}

func (o *actionOperator) endJob(ctx context.Context, cfg config.Configuration, taskID, jobID, stepName string, action *prov.Action, inError bool) error {
	if !inError {
		err := o.resumeWorkflow(cfg, taskID, stepName)
		if err != nil {
			log.Printf("failed to resume job run workflow with actionID:%q, jobID:%q due to error:%+v", action.ID, jobID, err)
		}
	} else {
		return o.setStepInError(cfg, taskID, stepName)
	}
	return nil
}

func (o *actionOperator) resumeWorkflow(cfg config.Configuration, taskID, stepName string) error {
	cc, err := cfg.GetConsulClient()
	if err != nil {
		return err
	}
	// job running step must be set to done and workflow must be resumed
	step := &tasks.TaskStep{Status: tasks.TaskStepStatusDONE.String(), Name: stepName}
	err = tasks.UpdateTaskStepStatus(cc.KV(), taskID, step)
	if err != nil {
		return errors.Wrapf(err, "failed to update step status to DONE for taskID:%q, stepName:%q", taskID, stepName)
	}
	coll := collector.NewCollector(cc)
	err = coll.ResumeTask(taskID)
	if err != nil {
		return errors.Wrapf(err, "failed to resume task with taskID:%q", taskID)
	}
	return nil
}

func (o *actionOperator) setStepInError(cfg config.Configuration, taskID, stepName string) error {
	cc, err := cfg.GetConsulClient()
	if err != nil {
		return err
	}
	// job running step must be set to done and workflow must be resumed
	step := &tasks.TaskStep{Status: tasks.TaskStepStatusERROR.String(), Name: stepName}
	err = tasks.UpdateTaskStepStatus(cc.KV(), taskID, step)
	if err != nil {
		return errors.Wrapf(err, "failed to update step status to ERROR for taskID:%q, stepName:%q", taskID, stepName)
	}
	return nil
}
