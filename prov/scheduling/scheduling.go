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

package scheduling

import (
	"encoding/json"
	"path"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov"
)

// RegisterAction allows to register a scheduled action and to start scheduling it
func RegisterAction(client *api.Client, deploymentID string, timeInterval time.Duration, action *prov.Action) (string, error) {
	log.Debugf("Action with ID:%q has been requested to be registered for scheduling with [deploymentID:%q, timeInterval:%q]", action.ID, deploymentID, timeInterval.String())
	u, err := uuid.NewV4()
	if err != nil {
		return "", errors.Wrapf(err, "Failed to generate a UUID for action %q", action.ID)
	}

	id := u.String()

	// Check mandatory parameters
	if deploymentID == "" {
		return "", errors.New("deploymentID is mandatory parameter to register scheduled action")
	}
	if action == nil || action.ActionType == "" {
		return "", errors.New("actionType is mandatory parameter to register scheduled action")
	}
	asyncOp, err := json.Marshal(action.AsyncOperation)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to generate async operation representation for action %q", action.ID)
	}
	scaPath := path.Join(consulutil.SchedulingKVPrefix, "actions", id)
	scaOps := api.KVTxnOps{
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(scaPath, "deploymentID"),
			Value: []byte(deploymentID),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(scaPath, "type"),
			Value: []byte(action.ActionType),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(scaPath, "interval"),
			Value: []byte(timeInterval.String()),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(scaPath, "async_op"),
			Value: asyncOp,
		},
	}

	if action.Data != nil {
		for k, v := range action.Data {
			scaOps = append(scaOps, &api.KVTxnOp{
				Verb:  api.KVSet,
				Key:   path.Join(scaPath, "data", k),
				Value: []byte(v),
			})
		}
	}

	ok, response, _, err := client.KV().Txn(scaOps, nil)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to register scheduled action for deploymentID:%q, type:%q, id:%q", deploymentID, action.ActionType, id)
	}
	if !ok {
		// Check the response
		errs := make([]string, 0)
		for _, e := range response.Errors {
			errs = append(errs, e.What)
		}
		return "", errors.Errorf("Failed to register scheduled action for deploymentID:%q, type:%q, id:%q due to:%s", deploymentID, action.ActionType, id, strings.Join(errs, ", "))
	}
	return id, nil
}

// UnregisterAction allows to unregister a scheduled action and to stop scheduling it
func UnregisterAction(client *api.Client, id string) error {
	log.Debugf("Unregister scheduled action with id:%q", id)
	scaPath := path.Join(consulutil.SchedulingKVPrefix, "actions", id)
	return errors.Wrap(consulutil.StoreConsulKeyAsString(path.Join(scaPath, ".unregisterFlag"), "true"), "Failed to flag scheduled action for removal")
}

// UpdateActionData updates the value of a given data within an action
func UpdateActionData(client *api.Client, id, key, value string) error {

	// check if action still exists
	actionIdPrefix := path.Join(consulutil.SchedulingKVPrefix, "actions", id)
	kvp, _, err := client.KV().Get(path.Join(actionIdPrefix, "deploymentID"), nil)
	if err != nil {
		return err
	}
	if kvp == nil {
		return errors.Errorf("Action with ID %s is unregistered", id)
	}

	scaKeyPath := path.Join(actionIdPrefix, "data", key)
	return errors.Wrapf(consulutil.StoreConsulKeyAsString(scaKeyPath, value), "Failed to update data %q for action %q", key, id)
}
