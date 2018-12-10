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

package ansible

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/prov"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/tosca"
)

type actionOperator struct {
	executor *defaultExecutor
}

func (o *actionOperator) ExecAction(ctx context.Context, cfg config.Configuration, taskID, deploymentID string, action *prov.Action) (bool, error) {

	originalTaskID, ok := action.Data["originalTaskID"]
	if !ok {
		return true, errors.New(`missing mandatory parameter "originalTaskID" in monitoring action`)
	}

	nodeName, ok := action.Data["nodeName"]
	if !ok {
		return true, errors.New(`missing mandatory parameter "nodeName" in monitoring action`)
	}

	operationString, ok := action.Data["operation"]
	if !ok {
		return true, errors.New(`missing mandatory parameter "operation" in monitoring action`)
	}

	var operation prov.Operation
	err := json.Unmarshal([]byte(operationString), &operation)
	if !ok {
		return true, errors.Wrap(err, "failed to unmarshal given operation")
	}

	opErr := o.executor.ExecOperation(ctx, cfg, originalTaskID, deploymentID, nodeName, operation)

	if strings.ToLower(operation.Name) == tosca.RunnableRunOperationName {
		cc, err := cfg.GetConsulClient()
		if err != nil {
			return false, err
		}
		// for now we consider only instance 0
		dataName := nodeName + "-0-TOSCA_JOB_STATUS"
		status, err := tasks.GetTaskData(cc.KV(), originalTaskID, dataName)
		if err != nil {
			return false, err
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).Registerf("got job status %q, error: %v", status, err)
		switch status {
		case "RUNNING", "QUEUED":
			return false, opErr
		case "COMPLETED":
			return true, opErr
		case "FAILED":
			if opErr == nil {
				opErr = errors.Errorf("job implementation of node %q was detected as failed", nodeName)
			}
			return true, opErr
		}

	}
	return false, opErr
}
