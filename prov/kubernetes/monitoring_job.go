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

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/prov"
)

type actionOperator struct {
}

func (o *actionOperator) ExecAction(ctx context.Context, cfg config.Configuration, taskID, deploymentID string, action *prov.Action) (bool, error) {
	originalTaskID, ok := action.Data["originalTaskID"]
	if !ok {
		return true, errors.New(`missing mandatory parameter "originalTaskID" in monitoring action`)
	}

	jobID, ok := action.Data["jobID"]
	if !ok {
		return true, errors.New(`missing mandatory parameter "jobID" in monitoring action`)
	}
	namespace, ok := action.Data["namespace"]
	if !ok {
		return true, errors.New(`missing mandatory parameter "namespace" in monitoring action`)
	}
	// Check if namespace was provided
	var namespaceProvided bool
	namespaceProvidedStr, ok := action.Data["namespaceProvided"]
	if ok {
		b, err := strconv.ParseBool(namespaceProvidedStr)
		if err != nil {
			return true, errors.New(`unable to transform namespaceProvided value to boolean`)
		}
		namespaceProvided = b
	}

	nodeName, ok := action.Data["nodeName"]
	if !ok {
		return true, errors.New(`missing mandatory parameter "nodeName" in monitoring action`)
	}

	stepName, ok := action.Data["stepName"]
	if !ok {
		return true, errors.New(`missing mandatory parameter "stepName" in monitoring action`)
	}

	var locationProps config.DynamicMap
	locationMgr, err := locations.GetManager(cfg)
	if err == nil {
		locationProps, err = locationMgr.GetLocationPropertiesForNode(ctx, deploymentID, nodeName, infrastructureType)
	}
	if err != nil {
		return true, err
	}

	clientSet, err := getClientSet(locationProps)
	if err != nil {
		return true, err
	}

	return o.monitorJob(ctx, cfg, clientSet, namespaceProvided, deploymentID, originalTaskID, stepName, namespace, jobID, action)
}

// Return true if the job is to be deregisted by yorc, so no more monitoring...
func (o *actionOperator) monitorJob(ctx context.Context, cfg config.Configuration,
	clientSet *kubernetes.Clientset, namespaceProvided bool, deploymentID, originalTaskID, stepName, namespace, jobID string, action *prov.Action) (bool, error) {
	var (
		err        error
		deregister bool
	)
	job, err := clientSet.BatchV1().Jobs(namespace).Get(ctx, jobID, metav1.GetOptions{})
	if err != nil {
		return true, errors.Wrapf(err, "can not retrieve job %q", jobID)
	}

	publishJobLogs(ctx, cfg, clientSet, deploymentID, namespace, job.Name, action)

	// Check for pods status to refine job state
	podsList, err := clientSet.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: "job-name=" + job.Name})
	if err != nil {
		return true, errors.Wrapf(err, "can not retrieve pods for job %q", jobID)
	}
	var nbRunning int
	var nbPending int
	var nbUnknown int
	var nbSucceeded int
	var nbFailed int
	var jobState string
	for _, pod := range podsList.Items {
		switch pod.Status.Phase {
		case "Running":
			// The pod has been bound to a node, and all of the containers have been created.
			// At least one container is still running, or is in the process of starting or restarting.
			nbRunning = nbRunning + 1
		case "Pending":
			// The pod has been accepted by the Kubernetes system, but one or more of the container images has not been created.
			// This includes time before being scheduled as well as time spent downloading images over the network, which could take a while.
			nbPending = nbPending + 1
		case "Succeeded":
			// All containers in the pod have terminated in success, and will not be restarted.
			nbSucceeded = nbSucceeded + 1
		case "Failed":
			// All containers in the pod have terminated, and at least one container has terminated in failure.
			// The container either exited with non-zero status or was terminated by the system.
			nbFailed = nbFailed + 1
		case "Unknown":
			// For some reason the state of the pod could not be obtained, typically due to an error in communicating with the host of the pod.
			nbUnknown = nbUnknown + 1
		}
	}
	// If the job is active, the pods status is used to set more precise value in the state attribute in the job instance.
	if job.Status.Active != 0 {
		// deregister remains false
		if nbUnknown > 0 {
			jobState = "Unknown"
		}
		if nbPending > 0 {
			jobState = "Pending"
		}
		if nbRunning > 0 {
			jobState = "Running"
		}
	}

	if job.Status.Active == 0 && (job.Status.Succeeded != 0 || job.Status.Failed != 0) {
		// job state is either succeeded or failed
		deregister = true
		if job.Status.Succeeded != 0 {
			jobState = "Succeeded"
		} else {
			jobState = "Failed"
		}
		err = deleteJob(ctx, deploymentID, namespace, jobID, namespaceProvided, clientSet)
		if err != nil {
			// error to be returned
			err = errors.Wrapf(err, "failed to delete completed job %q", jobID)
		}
		if job.Status.Succeeded < *job.Spec.Completions {
			jobState = "Failed"
			// error to be returned
			err = errors.Errorf("job failed: succeeded pods: %d, failed pods: %d, requested completions: %d", job.Status.Succeeded, job.Status.Failed, *job.Spec.Completions)
		}
	}

	if job.Status.Active == 0 && job.Status.Succeeded == 0 && job.Status.Failed == 0 {
		jobState = "No pods created"
	}
	// TODO(loicalbertin) for now we consider only instance 0 (https://github.com/ystia/yorc/issues/670)
	// Get previus node status and avoit to set err to nil if no error occurs in get
	previousState, err1 := deployments.GetInstanceStateString(ctx, deploymentID, action.Data["nodeName"], "0")
	if err1 != nil {
		err = errors.Wrapf(err, "failed to get instance state for job %q", jobID)
	}

	if len(jobState) != 0 && previousState != jobState {
		deployments.SetInstanceStateStringWithContextualLogs(ctx, deploymentID, action.Data["nodeName"], "0", jobState)
	}

	return deregister, err
}
