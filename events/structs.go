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
	infoDeploymentID infoType = iota

	infoStatus

	infoTimestamp

	infoEventType

	infoWorkFlowID

	infoAlienExecutionID

	infoNodeID

	infoInstanceID

	infoOperationName

	infoTargetNodeID

	infoTargetInstanceID

	infoAlienTaskExecutionID

	infoWorkflowStepID
)

func (i infoType) String() string {
	switch i {
	case infoDeploymentID:
		return "deploymentId"
	case infoStatus:
		return "status"
	case infoTimestamp:
		return "timestamp"
	case infoEventType:
		return "type"
	case infoWorkFlowID:
		return "workFlowID"
	case infoAlienExecutionID:
		return "alienExecutionID"
	case infoNodeID:
		return "nodeId"
	case infoInstanceID:
		return "instanceId"
	case infoOperationName:
		return "operationName"
	case infoTargetNodeID:
		return "targetNodeId"
	case infoTargetInstanceID:
		return "targetInstanceId"
	case infoAlienTaskExecutionID:
		return "alienTaskId"
	case infoWorkflowStepID:
		return "stepId"
	}
	return ""
}

// statusChange represents status change event
type statusChange struct {
	timestamp    string
	eventType    StatusChangeType
	deploymentID string
	status       string
	info         Info
}

// WorkflowStepInfo represents specific workflow step event information
type WorkflowStepInfo struct {
	WorkflowName     string
	InstanceName     string
	NodeName         string
	StepName         string
	OperationName    string
	TargetNodeID     string
	TargetInstanceID string
}

// Create a KVPair corresponding to an event and put it to Consul under the event prefix,
// in a sub-tree corresponding to its deployment
// The eventType goes to the KVPair's Flags field
// The content is JSON format
func (e *statusChange) register() (string, error) {
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

func (e *statusChange) flat() map[string]interface{} {
	flat := make(map[string]interface{})

	flat[infoDeploymentID.String()] = e.deploymentID
	flat[infoStatus.String()] = e.status
	flat[infoTimestamp.String()] = e.timestamp
	flat[infoEventType.String()] = e.eventType.String()
	for k, v := range e.info {
		flat[k.String()] = v
	}
	return flat
}

// newStatusChange allows to create a new statusChange if mandatory information is ok
func newStatusChange(eventType StatusChangeType, info Info, deploymentID, status string) (*statusChange, error) {
	e := &statusChange{info: info, eventType: eventType, status: status, deploymentID: deploymentID}
	if err := e.check(); err != nil {
		// Just print Warning log if any mandatory field is not set
		// But let's us the possibility to return an error
		log.Print("[WARNING] " + err.Error())
	}
	return e, nil
}

func (e *statusChange) check() error {
	if e.deploymentID == "" || e.status == "" {
		return errors.New("DeploymentID and status are mandatory parameters for EventStatusVChange")
	}

	mandatoryMap := map[StatusChangeType][]infoType{
		StatusChangeTypeInstance:      {infoNodeID, infoInstanceID},
		StatusChangeTypeCustomCommand: {infoAlienExecutionID},
		StatusChangeTypeScaling:       {infoAlienExecutionID},
		StatusChangeTypeWorkflow:      {infoAlienExecutionID},
		StatusChangeTypeWorkflowStep:  {infoAlienExecutionID, infoWorkFlowID, infoNodeID, infoWorkflowStepID, infoInstanceID},
		StatusChangeTypeAlienTask:     {infoAlienExecutionID, infoWorkFlowID, infoNodeID, infoWorkflowStepID, infoInstanceID, infoAlienTaskExecutionID},
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
