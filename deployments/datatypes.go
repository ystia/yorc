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
	"strings"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/helper/consulutil"
)

// GetTypePropertyDataType returns the type of a property as defined in its property definition
//
// Default value is "string" if not specified.
// Lists and Maps types have their entry_schema value append separated by a semicolon (ex "map:string")
// again if there is specified entry_schema "string" is assumed.
func GetTypePropertyDataType(deploymentID, typeName, propertyName string) (string, error) {
	return getTypePropertyOrAttributeDataType(deploymentID, typeName, propertyName, true)
}

// GetTypeAttributeDataType returns the type of a attribute as defined in its attribute definition
//
// Default value is "string" if not specified.
// Lists and Maps types have their entry_schema value append separated by a semicolon (ex "map:string")
// again if there is specified entry_schema "string" is assumed.
func GetTypeAttributeDataType(deploymentID, typeName, propertyName string) (string, error) {
	return getTypePropertyOrAttributeDataType(deploymentID, typeName, propertyName, false)
}

func getTypePropertyOrAttributeDataType(deploymentID, typeName, propertyName string, isProp bool) (string, error) {
	tType := "properties"
	if !isProp {
		tType = "attributes"
	}
	typePath, err := locateTypePath(deploymentID, typeName)
	if err != nil {
		return "", err
	}
	propertyDefinitionPath := path.Join(typePath, tType, propertyName)
	exist, value, err := consulutil.GetStringValue(path.Join(propertyDefinitionPath, "type"))
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist || value == "" {
		// Check parent
		parentType, err := GetParentType(deploymentID, typeName)
		if parentType == "" {
			return "", nil
			// return "", errors.Errorf("property %q not found in type %q", propertyName, typeName)
		}
		result, err := GetTypePropertyDataType(deploymentID, parentType, propertyName)
		return result, errors.Wrapf(err, "property %q not found in type %q", propertyName, typeName)
	}
	dataType := value
	if dataType == "map" || dataType == "list" {
		exist, value, err := consulutil.GetStringValue(path.Join(propertyDefinitionPath, "entry_schema"))
		if err != nil {
			return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if !exist || value == "" {
			dataType += ":string"
		} else {
			dataType += ":" + value
		}
	} else if dataType == "" {
		dataType = "string"
	}
	return dataType, nil
}

// GetNestedDataType return the type of a nested datatype
func GetNestedDataType(deploymentID, baseType string, nestedKeys ...string) (string, error) {
	currentType := baseType
	var err error
	for i := 0; i < len(nestedKeys); i++ {
		if strings.HasPrefix(currentType, "list:") {
			currentType = currentType[5:]
			continue
		} else if strings.HasPrefix(currentType, "map:") {
			currentType = currentType[4:]
			continue
		}
		currentType, err = GetTypePropertyDataType(deploymentID, currentType, nestedKeys[i])
		if err != nil {
			return "", errors.Wrapf(err, "failed to get type of nested datatype %q.%q", baseType, strings.Join(nestedKeys, "."))
		}

	}
	return currentType, nil
}

// GetTopologyInputType retrieves the optional data type of the parameter.
//
// As this keyname is required for a TOSCA Property definition, but is not for a TOSCA Parameter definition it may be empty.
// If the input type is list or map and an entry_schema is provided a semicolon and the entry_schema value are appended to
// the type (ie list:integer) otherwise string is assumed for then entry_schema.
func GetTopologyInputType(deploymentID, inputName string) (string, error) {
	return getTopologyInputOrOutputType(deploymentID, inputName, "inputs")
}

// GetTopologyOutputType retrieves the optional data type of the parameter.
//
// As this keyname is required for a TOSCA Property definition, but is not for a TOSCA Parameter definition it may be empty.
// If the input type is list or map and an entry_schema is provided a semicolon and the entry_schema value are appended to
// the type (ie list:integer) otherwise string is assumed for then entry_schema.
func GetTopologyOutputType(deploymentID, outputName string) (string, error) {
	return getTopologyInputOrOutputType(deploymentID, outputName, "outputs")
}

func getTopologyInputOrOutputType(deploymentID, parameterName, parameterType string) (string, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", parameterType, parameterName, "type"))
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist {
		return "", nil
	}
	iType := value
	if iType == "list" || iType == "map" {
		exist, value, err = consulutil.GetStringValue(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", parameterType, parameterName, "entry_schema"))
		if err != nil {
			return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if exist && value != "" {
			iType += ":" + value
		} else {
			iType += ":string"
		}
	}
	return iType, nil
}
