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
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"
	"path"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/helper/consulutil"
)

func getParameterDefinitionStruct(ctx context.Context, deploymentID, parameterName, parameterType string) (bool, *tosca.ParameterDefinition, error) {
	valuePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", parameterType, parameterName)
	parameterDef := new(tosca.ParameterDefinition)
	exist, err := storage.GetStore(types.StoreTypeDeployment).Get(valuePath, parameterDef)
	if err != nil {
		return false, nil, err
	}
	return exist, parameterDef, nil
}

// GetTopologyOutputValue returns the value of a given topology output
func GetTopologyOutputValue(ctx context.Context, deploymentID, outputName string, nestedKeys ...string) (*TOSCAValue, error) {
	exist, paramDef, err := getParameterDefinitionStruct(ctx, deploymentID, outputName, "outputs")
	if err != nil || !exist {
		return nil, err
	}
	// TODO this is not clear in the specification but why do we return a single value in this context as in case of attributes and multi-instances
	// we can have different results.
	// We have to improve this.
	return getValueAssignment(ctx, deploymentID, "", "0", "", paramDef.Value, paramDef.Default, nestedKeys...)
}

// GetTopologyOutputsNames returns the list of outputs for the deployment
func GetTopologyOutputsNames(ctx context.Context, deploymentID string) ([]string, error) {
	optPaths, err := storage.GetStore(types.StoreTypeDeployment).Keys(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "/topology/outputs"))
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for i := range optPaths {
		optPaths[i] = path.Base(optPaths[i])
	}
	return optPaths, nil
}
