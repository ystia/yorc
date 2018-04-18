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
	"fmt"
	"path"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/tosca"
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
	for i := startOfDepStatusConst + 1; i < endOfDepStatusConst; i++ {
		if DeploymentStatus(i).String() == status {
			return i, nil
		}
	}
	return INITIAL, errors.Errorf("Invalid deployment status %q", status)
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

// GetDeploymentSubstitutionMapping returns the Substitution Mappings defined
// for a given nodeType (ie. a service provided by another deployment)
func GetDeploymentSubstitutionMapping(kv *api.KV, deploymentID, nodeType string) (tosca.SubstitutionMapping, error) {
	// TODO call util getSubstitutionMappingFromStore
	var substitutionMapping tosca.SubstitutionMapping
	return substitutionMapping, nil
}
