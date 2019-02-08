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
	"path"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/helper/consulutil"
)

// GetTopologyOutputValue returns the value of a given topology output
func GetTopologyOutputValue(kv *api.KV, deploymentID, outputName string, nestedKeys ...string) (*TOSCAValue, error) {
	dataType, err := GetTopologyOutputType(kv, deploymentID, outputName)
	if err != nil {
		return nil, err
	}
	valuePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/outputs", outputName, "value")
	// TODO this is not clear in the specification but why do we return a single value in this context as in case of attributes and multi-instances
	// we can have different results.
	// We have to improve this.
	res, err := getValueAssignmentWithDataType(kv, deploymentID, valuePath, "", "0", "", dataType, nestedKeys...)
	if err != nil || res != nil {
		return res, err
	}
	// check the default
	defaultPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/outputs", outputName, "default")
	return getValueAssignmentWithDataType(kv, deploymentID, defaultPath, "", "0", "", dataType, nestedKeys...)
}

// GetTopologyOutputsNames returns the list of outputs for the deployment
func GetTopologyOutputsNames(kv *api.KV, deploymentID string) ([]string, error) {
	optPaths, _, err := kv.Keys(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "/topology/outputs")+"/", "/", nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for i := range optPaths {
		optPaths[i] = path.Base(optPaths[i])
	}
	return optPaths, nil
}
