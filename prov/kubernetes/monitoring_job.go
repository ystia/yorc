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

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/prov"
)

type actionOperator struct {
	clientset kubernetes.Interface
}

func (o *actionOperator) ExecAction(ctx context.Context, cfg config.Configuration, taskID, deploymentID string, action *prov.Action) (bool, error) {
	originalTaskID, ok := action.Data["originalTaskID"]
	if !ok {
		return true, errors.New(`missing mandatory parameter "originalTaskID" in monitoring action`)
	}
	var err error
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

	stepName, ok := action.Data["stepName"]
	if !ok {
		return true, errors.New(`missing mandatory parameter "stepName" in monitoring action`)
	}

	return o.monitorJob(ctx, cfg, namespaceProvided, deploymentID, originalTaskID, stepName, namespace, jobID, action)
}

func (o *actionOperator) monitorJob(ctx context.Context, cfg config.Configuration, namespaceProvided bool, deploymentID, originalTaskID, stepName, namespace, jobID string, action *prov.Action) (bool, error) {
	job, err := o.clientset.BatchV1().Jobs(namespace).Get(jobID, metav1.GetOptions{})
	if err != nil {
		return true, errors.Wrapf(err, "can not retrieve job %q", jobID)
	}

	publishJobLogs(ctx, cfg, o.clientset, deploymentID, namespace, job.Name, action)

	if job.Status.Active == 0 && (job.Status.Succeeded != 0 || job.Status.Failed != 0) {
		err = deleteJob(ctx, deploymentID, namespace, jobID, namespaceProvided, o.clientset)
		if err != nil {
			return true, errors.Wrapf(err, "failed to delete completed job %q", jobID)
		}
		if job.Status.Succeeded < *job.Spec.Completions {
			return true, errors.Errorf("job failed: succeeded pods: %d, failed pods: %d, requested completions: %d", job.Status.Succeeded, job.Status.Failed, *job.Spec.Completions)
		}
		return true, nil
	}
	return false, nil
}
