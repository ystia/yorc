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

package deployments

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/ystia/yorc/v4/helper/collections"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
)

const purgedDeploymentsLock = consulutil.PurgedDeploymentKVPrefix + ".lock"

type deploymentNotFound struct {
	deploymentID string
}

func (d deploymentNotFound) Error() string {
	return fmt.Sprintf("Deployment with id %q not found", d.deploymentID)
}

// IsDeploymentNotFoundError checks if an error is a deployment not found error
func IsDeploymentNotFoundError(err error) bool {
	cause := errors.Cause(err)
	_, ok := cause.(deploymentNotFound)
	return ok
}

type inconsistentDeploymentError struct {
	deploymentID string
}

func (t inconsistentDeploymentError) Error() string {
	return fmt.Sprintf("Inconsistent deployment with ID %q", t.deploymentID)
}

// IsInconsistentDeploymentError checks if an error is an inconsistent deployment error
func IsInconsistentDeploymentError(err error) bool {
	cause := errors.Cause(err)
	_, ok := cause.(inconsistentDeploymentError)
	return ok
}

// NewInconsistentDeploymentError allows to create a new inconsistentDeploymentError error
func NewInconsistentDeploymentError(deploymentID string) error {
	return inconsistentDeploymentError{deploymentID: deploymentID}
}

// DeploymentStatusFromString returns a DeploymentStatus from its textual representation.
//
// If ignoreCase is 'true' the given status is upper cased to match the generated status strings.
// If the given status does not match any known status an error is returned
func DeploymentStatusFromString(status string, ignoreCase bool) (DeploymentStatus, error) {
	if ignoreCase {
		status = strings.ToUpper(status)
	}
	return ParseDeploymentStatus(status)
}

// GetDeploymentStatus returns a DeploymentStatus for a given deploymentId
//
// If the given deploymentId doesn't refer to an existing deployment an error is returned. This error could be checked with
//  IsDeploymentNotFoundError(err error) bool
//
// For example:
//  if status, err := GetDeploymentStatus(kv, deploymentId); err != nil {
//  	if IsDeploymentNotFoundError(err) {
//  		// Do something in case of deployment not found
//  	}
//  }
func GetDeploymentStatus(kv *api.KV, deploymentID string) (DeploymentStatus, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "status"))
	if err != nil {
		return INITIAL, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist || value == "" {
		return INITIAL, deploymentNotFound{deploymentID: deploymentID}
	}
	return DeploymentStatusFromString(value, true)
}

//GetDeploymentTemplateName only return the name of the template used during the deployment
func GetDeploymentTemplateName(kv *api.KV, deploymentID string) (string, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "name"))
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist || value == "" {
		return "", deploymentNotFound{deploymentID: deploymentID}
	}
	return value, nil
}

// DoesDeploymentExists checks if a given deploymentId refer to an existing deployment
func DoesDeploymentExists(kv *api.KV, deploymentID string) (bool, error) {
	if _, err := GetDeploymentStatus(kv, deploymentID); err != nil {
		if IsDeploymentNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// SetDeploymentStatus sets the deployment status to the given status.
//
// This function will first check for an update of the current status and do it only if necessary.
// It will also emit a proper event to notify of status change.
// It is safe for concurrent use by using a CAS mechanism.
func SetDeploymentStatus(ctx context.Context, kv *api.KV, deploymentID string, status DeploymentStatus) error {
RETRY:
	// As we loop check if context is not cancelled
	select {
	case <-ctx.Done():
		return errors.Wrapf(ctx.Err(), "failed to update deployment %q status to %q", deploymentID, status.String())
	default:
	}

	kvp, meta, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "status"), nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	if status != INITIAL {
		if kvp == nil || len(kvp.Value) == 0 || meta == nil {
			return errors.WithStack(&deploymentNotFound{deploymentID})
		}

		currentStatus, err := DeploymentStatusFromString(string(kvp.Value), true)
		if err != nil {
			return err
		}

		if status != currentStatus {
			kvp.Value = []byte(status.String())
			kvp.ModifyIndex = meta.LastIndex
			ok, _, err := kv.CAS(kvp, nil)
			if err != nil {
				return errors.Wrapf(err, "Failed to set deployment status to %q for deploymentID:%q", status.String(), deploymentID)
			}
			if !ok {
				goto RETRY
			}
			log.Debugf("Deployment status change for %s from %s to %s",
				deploymentID, currentStatus.String(), status.String())
			events.PublishAndLogDeploymentStatusChange(ctx, kv, deploymentID, strings.ToLower(status.String()))
		}
		return nil
	}

	// Set the status to INITIAL: no need to handle concurrency and to check previous status
	if err = consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "status"), status.String()); err != nil {
		return errors.Wrapf(err, "Failed to set deployment status to %q for deploymentID:%q", status.String(), deploymentID)
	}
	events.PublishAndLogDeploymentStatusChange(ctx, kv, deploymentID, strings.ToLower(status.String()))
	return nil
}

// TagDeploymentAsPurged registers current purge time and emit a deployment status change event and a log for the given deployment
//
// The timestamp will be used to evict purged deployments after an given delay.
func TagDeploymentAsPurged(ctx context.Context, cc *api.Client, deploymentID string) error {
	lock, _, err := acquirePurgedDeploymentsLock(ctx, cc)
	if err != nil {
		return err
	}
	defer lock.Unlock()
	consulutil.StoreConsulKeyAsString(path.Join(consulutil.PurgedDeploymentKVPrefix, deploymentID), time.Now().Format(time.RFC3339Nano))

	// Just Publish an event that the deployment is successfully
	// This event will stay into consul even if the deployment is actually purged...
	// To prevent unexpected errors
	_, err = events.PublishAndLogDeploymentStatusChange(ctx, cc.KV(), deploymentID, strings.ToLower(PURGED.String()))
	return err
}

// CleanupPurgedDeployments definitively removes purged deployments
//
// Deployment are cleaned-up if they have been purged for at least evictionTimeout or
// if the are in the extraDeployments list
func CleanupPurgedDeployments(ctx context.Context, cc *api.Client, evictionTimeout time.Duration, extraDeployments ...string) error {
	lock, _, err := acquirePurgedDeploymentsLock(ctx, cc)
	if err != nil {
		return err
	}
	defer lock.Unlock()
	// Appending a final "/" here is not necessary as there is no other keys starting with consulutil.PurgedDeploymentKVPrefix prefix
	kvpList, _, err := cc.KV().List(consulutil.PurgedDeploymentKVPrefix, nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, kvp := range kvpList {
		if kvp.Key == purgedDeploymentsLock {
			// ignore lock
			continue
		}
		deploymentID := path.Base(kvp.Key)
		if collections.ContainsString(extraDeployments, deploymentID) {
			err = cleanupPurgedDeployment(ctx, cc.KV(), deploymentID)
			if err != nil {
				return err
			}
			continue
		}
		purgeDate, err := time.Parse(time.RFC3339Nano, string(kvp.Value))
		if err != nil {
			log.Printf("WARN failed to parse %q for purged timestamp of deployment %q", string(kvp.Value), deploymentID)
			continue
		}
		if purgeDate.Add(evictionTimeout).Before(time.Now()) {
			err = cleanupPurgedDeployment(ctx, cc.KV(), deploymentID)
			if err != nil {
				return err
			}
			continue
		}
	}
	return nil
}

func cleanupPurgedDeployment(ctx context.Context, kv *api.KV, deploymentID string) error {
	// Delete events & logs tree corresponding to the deployment
	// This is useful when redeploying an application that has been previously purged
	// as it may still have the purged event and log.
	err := events.PurgeDeploymentEvents(kv, deploymentID)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	err = events.PurgeDeploymentLogs(kv, deploymentID)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	_, err = kv.Delete(path.Join(consulutil.PurgedDeploymentKVPrefix, deploymentID), nil)
	return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
}

func acquirePurgedDeploymentsLock(ctx context.Context, cc *api.Client) (*api.Lock, <-chan struct{}, error) {
	lock, err := cc.LockKey(purgedDeploymentsLock)
	if err != nil {
		return nil, nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	var leaderCh <-chan struct{}
	for leaderCh == nil {
		leaderCh, err = lock.Lock(ctx.Done())
		if err != nil {
			return nil, nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		select {
		case <-ctx.Done():
			return nil, nil, errors.Wrap(ctx.Err(), "failed to acquire lock on purged deployments")
		default:
		}
	}
	return lock, leaderCh, nil
}

// DeleteDeployment deletes a given deploymentID from the deployments path
func DeleteDeployment(kv *api.KV, deploymentID string) error {
	// Remove from KV this purge tasks
	_, err := kv.DeleteTree(path.Join(consulutil.DeploymentKVPrefix, deploymentID)+"/", nil)
	return errors.Wrap(err, consulutil.ConsulGenericErrMsg)

}
