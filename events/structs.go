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

package events

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"path"
	"strings"
	"time"
)

//go:generate go-enum -f=structs.go --lower

// StatusChangeType x ENUM(
// Instance,
// Deployment,
// CustomCommand,
// Scaling,
// Workflow,
// WorkflowStep
// AlienTask
// )
type StatusChangeType int

// Info allows to provide custom/specific additional information for event
type Info map[infoType]interface{}

// infoType represents Event status change information type
type infoType int

const (
	infoWorkFlowID infoType = iota

	infoExecutionID

	infoNodeID

	infoInstanceID

	infoStepID

	infoOperationName

	infoTargetNodeID

	infoTargetInstanceID

	infoAlienTaskID

	infoWorkflowStepID
)

func (i infoType) String() string {
	switch i {
	case infoWorkFlowID:
		return "workFlowID"
	case infoExecutionID:
		return "executionID"
	case infoNodeID:
		return "nodeId"
	case infoInstanceID:
		return "instanceId"
	case infoStepID:
		return "stepId"
	case infoOperationName:
		return "operationName"
	case infoTargetNodeID:
		return "targetNodeId"
	case infoTargetInstanceID:
		return "targetInstanceId"
	case infoAlienTaskID:
		return "taskId"
	case infoWorkflowStepID:
		return "workflowStepId"
	}
	return ""
}

// StatusChange represents status change event
type StatusChange struct {
	timestamp    string
	eventType    StatusChangeType
	deploymentID string
	status       string
	info         Info
}

// Create a KVPair corresponding to an event and put it to Consul under the event prefix,
// in a sub-tree corresponding to its deployment
// The eventType goes to the KVPair's Flags field
// The content is JSON format
func (e *StatusChange) register() (string, error) {
	e.timestamp = time.Now().Format(time.RFC3339Nano)
	eventsPrefix := path.Join(consulutil.EventsPrefix, e.deploymentID)

	// For presentation purpose, each field is in flat json object
	flat := e.flat()
	b, err := json.Marshal(flat)
	if err != nil {
		log.Printf("Failed to marshal event [%+v]: due to error:%+v", e, err)
	}
	err = consulutil.StoreConsulKey(path.Join(eventsPrefix, e.timestamp), b)
	if err != nil {
		return "", err
	}
	return e.timestamp, nil
}

func (e *StatusChange) flat() map[string]interface{} {
	flat := make(map[string]interface{})

	flat["deploymentId"] = e.deploymentID
	flat["status"] = e.status
	flat["timestamp"] = e.timestamp
	flat["eventType"] = e.eventType.String()
	for k, v := range e.info {
		flat[k.String()] = v
	}
	return flat
}

// newStatusChange allows to create a new StatusChange if mandatory information is ok
func newStatusChange(eventType StatusChangeType, info Info, deploymentID, status string) (*StatusChange, error) {
	e := &StatusChange{info: info, eventType: eventType, status: status, deploymentID: deploymentID}
	if err := e.check(); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *StatusChange) check() error {
	if e.deploymentID == "" || e.status == "" {
		return errors.New("DeploymentID and status are mandatory parameters for EventStatusVChange")
	}

	mandatoryMap := map[StatusChangeType][]infoType{
		StatusChangeTypeInstance:      {infoNodeID, infoInstanceID},
		StatusChangeTypeCustomCommand: {infoExecutionID},
		StatusChangeTypeScaling:       {infoExecutionID},
		StatusChangeTypeWorkflow:      {infoExecutionID},
		StatusChangeTypeWorkflowStep:  {infoExecutionID, infoWorkFlowID, infoNodeID, infoStepID, infoOperationName},
		StatusChangeTypeAlienTask:     {infoExecutionID, infoWorkFlowID, infoNodeID, infoWorkflowStepID, infoOperationName, infoAlienTaskID},
	}
	// Check mandatory info in function of status change type
	if mandatoryInfos, is := mandatoryMap[e.eventType]; is {
		for _, mandatoryInfo := range mandatoryInfos {
			missing := make([]string, 0)
			if e.info == nil {
				for _, m := range mandatoryInfos {
					missing = append(missing, m.String())
				}
			}
			if _, is := e.info[mandatoryInfo]; !is {
				missing = append(missing, mandatoryInfo.String())
			}

			if len(missing) > 0 {
				return errors.Errorf("Missing mandatory parameters:%q for event:%q", strings.Join(missing, ","), e.eventType.String())
			}
		}
	}
	return nil
}
