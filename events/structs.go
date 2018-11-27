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
	"context"
	"encoding/json"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"path"
	"time"
)

//go:generate go-enum -f=structs.go --lower

// LogLevel x ENUM(
// Instance,
// Deployment,
// CustomCommand,
// Scaling,
// Workflow,
// WorkflowStep
// )
type StatusChangeType int

// EventAdditionalInfo allows to provide custom/specific additional information for event
type EventAdditionalInfo map[string]interface{}

// EventStatusChange represents status change event
type EventStatusChange struct {
	timestamp      string
	eventType      StatusChangeType
	deploymentID   string
	status         string
	additionalInfo EventAdditionalInfo
}

// Create a KVPair corresponding to an event and put it to Consul under the event prefix,
// in a sub-tree corresponding to its deployment
// The eventType goes to the KVPair's Flags field
// The content is JSON format
func (e EventStatusChange) register() (string, error) {
	e.timestamp = time.Now().Format(time.RFC3339Nano)
	eventsPrefix := path.Join(consulutil.EventsPrefix, e.deploymentID)

	// For presentation purpose, each field is in flat json object
	flat := e.flat()
	b, err := json.Marshal(flat)
	if err != nil {
		log.Printf("Failed to marshal event [%+v]: due to error:%+v", e, err)
	}

	//FIXME Do we still need to store event type as a flag ?
	err = consulutil.StoreConsulKeyWithFlags(path.Join(eventsPrefix, e.timestamp), b, uint64(e.eventType))
	if err != nil {
		return "", err
	}
	return e.timestamp, nil
}

func (e EventStatusChange) flat() map[string]interface{} {
	flat := make(map[string]interface{})

	flat["deploymentId"] = e.deploymentID
	flat["status"] = e.status
	flat["timestamp"] = e.timestamp
	for k, v := range e.additionalInfo {
		flat[k] = v
	}
	return flat
}

// newEventStatusChange allows to create a new EventStatusChange pointer by retrieving context log information
func newEventStatusChange(ctx context.Context, eventType StatusChangeType, info EventAdditionalInfo, deploymentID, status string) *EventStatusChange {
	event := &EventStatusChange{additionalInfo: info, eventType: eventType, status: status, deploymentID: deploymentID}
	if event.additionalInfo == nil {
		event.additionalInfo = make(EventAdditionalInfo)
	}
	logInfo, ok := FromContext(ctx)
	if ok {
		for k, v := range logInfo {
			event.additionalInfo[k.String()] = v
		}
	}
	return event
}
