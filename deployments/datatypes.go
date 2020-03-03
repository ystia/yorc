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
	"github.com/ystia/yorc/v4/tosca"
	"strings"

	"github.com/pkg/errors"
)

// GetTypePropertyDataType returns the type of a property as defined in its property definition
//
// Default value is "string" if not specified.
// Lists and Maps types have their entry_schema value append separated by a semicolon (ex "map:string")
// again if there is specified entry_schema "string" is assumed.
func GetTypePropertyDataType(ctx context.Context, deploymentID, typeName, propertyName string) (string, error) {
	return getTypePropertyOrAttributeDataType(ctx, deploymentID, typeName, propertyName, true)
}

// GetTypeAttributeDataType returns the type of a attribute as defined in its attribute definition
//
// Default value is "string" if not specified.
// Lists and Maps types have their entry_schema value append separated by a semicolon (ex "map:string")
// again if there is specified entry_schema "string" is assumed.
func GetTypeAttributeDataType(ctx context.Context, deploymentID, typeName, propertyName string) (string, error) {
	return getTypePropertyOrAttributeDataType(ctx, deploymentID, typeName, propertyName, false)
}

func getTypePropertyOrAttributeDataType(ctx context.Context, deploymentID, typeName, propertyName string, isProp bool) (string, error) {
	var dataType string
	var entrySchemaType string
	if isProp {
		def, err := getTypePropertyDefinition(ctx, deploymentID, typeName, propertyName)
		if err != nil {
			return "", err
		}
		if def != nil {
			dataType = def.Type
			if &def.EntrySchema != nil {
				entrySchemaType = def.EntrySchema.Type
			}
		}
	} else {
		def, err := getTypeAttributeDefinition(ctx, deploymentID, typeName, propertyName)
		if err != nil {
			return "", err
		}
		if def != nil {
			dataType = def.Type
			entrySchemaType = def.EntrySchema.Type
		}
	}

	if dataType == "" {
		return "", nil
	}

	if dataType == "map" || dataType == "list" {
		if entrySchemaType == "" {
			dataType += ":string"
		} else {
			dataType += ":" + entrySchemaType
		}
	} else if dataType == "" {
		dataType = "string"
	}
	return dataType, nil
}

// GetNestedDataType return the type of a nested datatype
func GetNestedDataType(ctx context.Context, deploymentID, baseType string, nestedKeys ...string) (string, error) {
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
		currentType, err = GetTypePropertyDataType(ctx, deploymentID, currentType, nestedKeys[i])
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
func GetTopologyInputType(ctx context.Context, deploymentID, inputName string) (string, error) {
	return getTopologyInputOrOutputType(ctx, deploymentID, inputName, "inputs")
}

// GetTopologyOutputType retrieves the optional data type of the parameter.
//
// As this keyname is required for a TOSCA Property definition, but is not for a TOSCA Parameter definition it may be empty.
// If the input type is list or map and an entry_schema is provided a semicolon and the entry_schema value are appended to
// the type (ie list:integer) otherwise string is assumed for then entry_schema.
func GetTopologyOutputType(ctx context.Context, deploymentID, outputName string) (string, error) {
	return getTopologyInputOrOutputType(ctx, deploymentID, outputName, "outputs")
}

func getTopologyInputOrOutputType(ctx context.Context, deploymentID, parameterName, parameterType string) (string, error) {
	exist, paramDef, err := getParameterDefinition(ctx, deploymentID, parameterName, parameterType)
	if err != nil || !exist {
		return "", err
	}
	return getTypeFromParamDefinition(ctx, paramDef), nil
}

func getTypeFromParamDefinition(ctx context.Context, parameterDefinition *tosca.ParameterDefinition) string {
	if parameterDefinition == nil {
		return ""
	}

	iType := parameterDefinition.Type
	if iType == "list" || iType == "map" {
		if parameterDefinition.EntrySchema.Type != "" {
			iType += ":" + parameterDefinition.EntrySchema.Type
		} else {
			iType += ":string"
		}
	}
	return iType
}

func getDataTypeComplexEntrySchema(dataType string) string {
	if strings.HasPrefix(dataType, "list:") {
		return dataType[5:]
	} else if strings.HasPrefix(dataType, "map:") {
		return dataType[4:]
	}
	return dataType
}

func getDataTypeComplexType(dataType string) string {
	var tType string
	if strings.HasPrefix(dataType, "list:") {
		tType = "list"
	} else if strings.HasPrefix(dataType, "map:") {
		tType = "map"
	}
	return tType
}

func isPrimitiveDataType(dataType string) bool {
	switch dataType {
	case "string", "integer", "float", "boolean", "timestamp":
		return true
	default:
		return false
	}
}
