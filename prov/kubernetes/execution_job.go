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
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
)

func (e *execution) executeAsync(ctx context.Context, stepName string, clientset kubernetes.Interface) (*prov.Action, time.Duration, error) {
	if strings.ToLower(e.operation.Name) != "tosca.interfaces.node.lifecycle.runnable.run" {
		return nil, 0, errors.Errorf("%q operation is not supported by the Kubernetes executor only \"tosca.interfaces.node.lifecycle.Runnable.run\" is.", e.operation.Name)
	}

	if e.nodeType != "yorc.nodes.kubernetes.api.types.JobResource" {
		return nil, 0, errors.Errorf("%q node type is not supported by the Kubernetes executor only \"yorc.nodes.kubernetes.api.types.JobResource\" is.", e.nodeType)
	}

	rSpec, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.nodeName, "resource_spec")
	if err != nil {
		return nil, 0, err
	}

	if rSpec == nil {
		return nil, 0, errors.Errorf("no resource_spec defined for node %q", e.nodeName)
	}

	jobRepr := &batchv1.Job{}
	log.Debugf("jobspec: %v", rSpec.RawString())
	// Unmarshal JSON to k8s data structs
	if err = json.Unmarshal([]byte(rSpec.RawString()), jobRepr); err != nil {
		return nil, 0, errors.Wrap(err, "The resource-spec JSON unmarshaling failed")
	}

	// Get the namespace if provided. Otherwise, the namespace is generated using the default yorc policy
	objectMeta := jobRepr.ObjectMeta
	var namespaceName string
	var namespaceProvided bool
	namespaceName, namespaceProvided = getNamespace(e.deploymentID, objectMeta)
	if !namespaceProvided {
		err = createNamespaceIfMissing(e.deploymentID, namespaceName, clientset)
		if err != nil {
			return nil, 0, err
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf("k8s Namespace %s created", namespaceName)
	}

	if jobRepr.Spec.Template.Spec.RestartPolicy != "Never" && jobRepr.Spec.Template.Spec.RestartPolicy != "OnFailure" {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf(`job RestartPolicy %q is invalid for a job, settings to "Never" by default`, jobRepr.Spec.Template.Spec.RestartPolicy)
		jobRepr.Spec.Template.Spec.RestartPolicy = "Never"
	}

	jobRepr, err = clientset.BatchV1().Jobs(namespaceName).Create(jobRepr)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to create job for node %q", e.nodeName)
	}
	// Fill all used data for job monitoring
	data := make(map[string]string)
	data["originalTaskID"] = e.taskID
	data["jobID"] = jobRepr.Name
	data["namespace"] = namespaceName
	if namespaceProvided {
		data["providedNamespace"] = namespaceName
	}
	data["stepName"] = stepName
	// TODO deal with outputs?
	// data["outputs"] = strings.Join(e.jobInfo.outputs, ",")
	return &prov.Action{ActionType: "k8s-job-monitoring", Data: data}, 5 * time.Second, nil
}
