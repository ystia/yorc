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

	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/log"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/helper/consulutil"
)

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
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "status"), nil)
	if err != nil {
		return INITIAL, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return INITIAL, deploymentNotFound{deploymentID: deploymentID}
	}
	return DeploymentStatusFromString(string(kvp.Value), true)
}

//GetDeploymentTemplateName only return the name of the template used during the deployment
func GetDeploymentTemplateName(kv *api.KV, deploymentID string) (string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "name"), nil)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return "", deploymentNotFound{deploymentID: deploymentID}
	}
	return string(kvp.Value), nil
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
