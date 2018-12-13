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

package upgradeschema

import (
	"encoding/json"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"path"
	"strings"
)

// UpgradeFromPre31 allows to upgrade Consul schema from schema version before 1.0.0 (pre 3.1 yorc version)
func UpgradeFromPre31(kv *api.KV, leaderch <-chan struct{}) error {
	log.Print("Preparing upgrade database schema to 1.0.0 schema version")
	return eventsChange(kv, leaderch)
}

func eventsChange(kv *api.KV, leaderch <-chan struct{}) error {
	// Events format has changed
	// Need to retrieve the previous StatusUpdateType and related event information
	// To store with the new format (JSON)
	log.Print("Update events format change")
	eventsPrefix := path.Join(consulutil.EventsPrefix)
	type StatusUpdateType uint64
	const (
		// InstanceStatusChangeType is the StatusUpdate type for an instance state change event
		InstanceStatusChangeType StatusUpdateType = iota
		// DeploymentStatusChangeType is the StatusUpdate type for an deployment status change event
		DeploymentStatusChangeType
		// CustomCommandStatusChangeType is the StatusUpdate type for an custom command status change event
		CustomCommandStatusChangeType
		// ScalingStatusChangeType is the StatusUpdate type for an scaling status change event
		ScalingStatusChangeType
		// WorkflowStatusChangeType is the StatusUpdate type for an workflow status change event
		WorkflowStatusChangeType
	)

	stringStatusUpdate := func(s StatusUpdateType) string {
		switch s {
		case InstanceStatusChangeType:
			return events.StatusChangeTypeInstance.String()
		case DeploymentStatusChangeType:
			return events.StatusChangeTypeDeployment.String()
		case CustomCommandStatusChangeType:
			return events.StatusChangeTypeCustomCommand.String()
		case ScalingStatusChangeType:
			return events.StatusChangeTypeScaling.String()
		case WorkflowStatusChangeType:
			return events.StatusChangeTypeWorkflow.String()
		}
		return ""
	}

	kvps, qm, err := kv.List(eventsPrefix, nil)
	if err != nil || qm == nil {
		return errors.Wrapf(err, "failed to upgrade consul database schema from 100")
	}
	for _, kvp := range kvps {
		event := make(map[string]string)
		depIDAndTimestamp := strings.Split(strings.TrimPrefix(kvp.Key, eventsPrefix+"/"), "/")
		deploymentID := depIDAndTimestamp[0]
		eventTimestamp := depIDAndTimestamp[1]

		values := strings.Split(string(kvp.Value), "\n")
		eventType := StatusUpdateType(kvp.Flags)
		switch eventType {
		case InstanceStatusChangeType:
			if len(values) != 3 {
				return errors.Errorf("failed to upgrade consul database schema from 100: unexpected event value %q for event %q", string(kvp.Value), kvp.Key)
			}
			event["deploymentId"] = deploymentID
			event["timestamp"] = eventTimestamp
			event["type"] = stringStatusUpdate(eventType)
			event["nodeId"] = values[0]
			event["status"] = values[1]
			event["instanceId"] = values[2]
		case DeploymentStatusChangeType:
			if len(values) != 1 {
				return errors.Errorf("failed to upgrade consul database schema from 100: unexpected event value %q for event %q", string(kvp.Value), kvp.Key)
			}
			event["deploymentId"] = deploymentID
			event["timestamp"] = eventTimestamp
			event["type"] = stringStatusUpdate(eventType)
			event["status"] = values[0]
		case CustomCommandStatusChangeType, ScalingStatusChangeType, WorkflowStatusChangeType:
			if len(values) != 2 {
				return errors.Errorf("failed to upgrade consul database schema from 100: unexpected event value %q for event %q", string(kvp.Value), kvp.Key)
			}
			event["deploymentId"] = deploymentID
			event["timestamp"] = eventTimestamp
			event["type"] = stringStatusUpdate(eventType)
			event["status"] = values[1]
			event["alienExecutionId"] = values[0]
		default:
			return errors.Errorf("failed to upgrade consul database schema from 100: unsupported event type %d for event %q", kvp.Flags, kvp.Key)
		}

		// Save new format value
		log.Debugf("Convert event format with event:%+v", event)
		b, err := json.Marshal(event)
		if err != nil {
			log.Printf("failed to upgrade consul database schema from 100:  failed to marshal event [%+v]: due to error:%+v", event, err)
		}
		err = consulutil.StoreConsulKey(path.Join(eventsPrefix, deploymentID, event["timestamp"]), b)
		if err != nil {
			return errors.Wrapf(err, "failed to upgrade consul database schema from 100")
		}
	}
	return nil
}
