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
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/tasks"
	"github.com/ystia/yorc/v4/tosca"
)

func (e *execution) executeAsync(ctx context.Context, checksPeriod time.Duration, stepName string, clientset kubernetes.Interface) (*prov.Action, time.Duration, error) {
	if strings.ToLower(e.operation.Name) != strings.ToLower(tosca.RunnableRunOperationName) {
		return nil, 0, errors.Errorf("%q operation is not supported by the Kubernetes executor only %q is.", e.operation.Name, tosca.RunnableRunOperationName)
	}

	if e.nodeType != "yorc.nodes.kubernetes.api.types.JobResource" {
		return nil, 0, errors.Errorf("%q node type is not supported by the Kubernetes executor only \"yorc.nodes.kubernetes.api.types.JobResource\" is.", e.nodeType)
	}

	jobID, err := tasks.GetTaskData(e.taskID, e.nodeName+"-jobId")
	if err != nil {
		return nil, 0, err
	}

	job, err := getJob(ctx, clientset, e.deploymentID, e.nodeName)
	if err != nil {
		return nil, 0, err
	}

	// Fill all used data for job monitoring
	data := make(map[string]string)
	data["originalTaskID"] = e.taskID
	data["jobID"] = jobID
	data["namespace"] = job.namespace
	data["nodeName"] = e.nodeName
	data["namespaceProvided"] = strconv.FormatBool(job.namespaceProvided)
	data["stepName"] = stepName
	// TODO deal with outputs?
	// data["outputs"] = strings.Join(e.jobInfo.outputs, ",")

	return &prov.Action{ActionType: "k8s-job-monitoring", Data: data}, checksPeriod, nil
}

func (e *execution) submitJob(ctx context.Context, clientset kubernetes.Interface) error {
	job, err := getJob(ctx, clientset, e.deploymentID, e.nodeName)
	if err != nil {
		return err
	}

	if !job.namespaceProvided {
		err = createNamespaceIfMissing(job.namespace, clientset)
		if err != nil {
			return err
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf("k8s Namespace %s created", job.namespace)
	}

	if job.jobRepr.Spec.Template.Spec.RestartPolicy != "Never" && job.jobRepr.Spec.Template.Spec.RestartPolicy != "OnFailure" {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf(`job RestartPolicy %q is invalid for a job, settings to "Never" by default`, job.jobRepr.Spec.Template.Spec.RestartPolicy)
		job.jobRepr.Spec.Template.Spec.RestartPolicy = "Never"
	}

	job.jobRepr, err = clientset.BatchV1().Jobs(job.namespace).Create(job.jobRepr)
	if err != nil {
		return errors.Wrapf(err, "failed to create job for node %q", e.nodeName)
	}

	// Set job id to the instance attribute
	err = deployments.SetAttributeForAllInstances(ctx, e.deploymentID, e.nodeName, "job_id", job.jobRepr.Name)
	if err != nil {
		return errors.Wrapf(err, "failed to create job for node %q", e.nodeName)
	}

	return tasks.SetTaskData(e.taskID, e.nodeName+"-jobId", job.jobRepr.Name)
}

func (e *execution) cancelJob(ctx context.Context, clientset kubernetes.Interface) error {
	jobID, err := tasks.GetTaskData(e.taskID, e.nodeName+"-jobId")
	if err != nil {
		if !tasks.IsTaskDataNotFoundError(err) {
			return err
		}
		// Not cancelling within the same task try to get jobID from attribute
		// TODO(loicalbertin) for now we consider only instance 0 (https://github.com/ystia/yorc/issues/670)
		jobIDValue, err := deployments.GetInstanceAttributeValue(ctx, e.deploymentID, e.nodeName, "0", "job_id")
		if err != nil {
			return errors.Wrap(err, "failed to retrieve job id to cancel, found neither in task context neither as instance attribute")
		}
		if jobIDValue == nil {
			return errors.New("failed to retrieve job id to cancel, found neither in task context neither as instance attribute")
		}
		jobID = jobIDValue.RawString()
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf(
			"k8s job cancellation called from a dedicated \"cancel\" workflow. JobID retrieved from node %q attribute. This may cause issues if multiple workflows are running in parallel. Prefer using a workflow cancellation.", e.nodeName)
	}
	job, err := getJob(ctx, clientset, e.deploymentID, e.nodeName)
	if err != nil {
		return errors.Wrapf(err, "failed to delete job for node %q", e.nodeName)
	}
	return deleteJob(ctx, e.deploymentID, job.namespace, jobID, job.namespaceProvided, clientset)
}

func (e *execution) executeJobOperation(ctx context.Context, clientset kubernetes.Interface) (err error) {
	switch strings.ToLower(e.operation.Name) {
	case strings.ToLower(tosca.RunnableSubmitOperationName):
		return e.submitJob(ctx, clientset)
	case strings.ToLower(tosca.RunnableCancelOperationName):
		return e.cancelJob(ctx, clientset)
	default:
		return errors.Errorf("unsupported operation %q for node %q", e.operation.Name, e.nodeName)
	}

}
