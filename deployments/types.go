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
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/deployments/store"
	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"
	"path"
)

type typeMissingError struct {
	name         string
	deploymentID string
}

func (e typeMissingError) Error() string {
	return fmt.Sprintf("Looking for a type %q that do not exists in deployment %q.", e.name, e.deploymentID)
}

// IsTypeMissingError checks if the given error is a TypeMissing error
func IsTypeMissingError(err error) bool {
	cause := errors.Cause(err)
	_, ok := cause.(typeMissingError)
	return ok
}

func getTypeBase(typePath string) (bool, *tosca.Type, error) {
	typ := new(tosca.Type)
	exist, err := storage.GetStore(types.StoreTypeDeployment).Get(typePath, typ)
	if err != nil {
		return false, nil, err
	}
	return exist, typ, nil
}

func getTypeStruct(deploymentID, typeName string, typ interface{}) error {
	typePath, err := locateTypePath(deploymentID, typeName)
	if err != nil {
		return err
	}

	exist, err := storage.GetStore(types.StoreTypeDeployment).Get(typePath, typ)
	if err != nil {
		return err
	}
	if !exist {
		return typeMissingError{deploymentID: deploymentID, name: typeName}
	}
	return nil
}

func getTypePropertyDefinitions(deploymentID, typeName string) (map[string]tosca.PropertyDefinition, error) {
	typePath, err := locateTypePath(deploymentID, typeName)
	if err != nil {
		return nil, err
	}

	// Retrieve type of the type (i.e "node", "relationship", "policy"...) to get the related struct
	exist, typBase, err := getTypeBase(typePath)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.Errorf("No type found with name:%q", typeName)
	}

	var typ interface{}
	switch typBase.Base {
	case "node":
		typ = new(tosca.NodeType)
	case "capability":
		typ = new(tosca.CapabilityType)
	case "relationship":
		typ = new(tosca.RelationshipType)
	case "artifact":
		typ = new(tosca.ArtifactType)
	case "policy":
		typ = new(tosca.PolicyType)
	case "data":
		typ = new(tosca.DataType)
	default:
		return nil, errors.Errorf("Unknown type:%q", typBase.Base)

	}

	exist, err = storage.GetStore(types.StoreTypeDeployment).Get(typePath, typ)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.Errorf("No type found with name:%q", typeName)
	}

	var mapProps map[string]tosca.PropertyDefinition
	switch t := typ.(type) {
	case *tosca.NodeType:
		mapProps = t.Properties
	case *tosca.RelationshipType:
		mapProps = t.Properties
	case *tosca.CapabilityType:
		mapProps = t.Properties
	case *tosca.DataType:
		mapProps = t.Properties
	case *tosca.ArtifactType:
		mapProps = t.Properties
	case *tosca.PolicyType:
		mapProps = t.Properties
	}

	return mapProps, nil
}

func getTypeAttributeDefinitions(deploymentID, typeName string) (map[string]tosca.AttributeDefinition, error) {
	typePath, err := locateTypePath(deploymentID, typeName)
	if err != nil {
		return nil, err
	}

	// Retrieve type of the type (i.e "node", "relationship", "policy"...) to get the related struct
	exist, typBase, err := getTypeBase(typePath)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.Errorf("No type found with name:%q", typeName)
	}

	var typ interface{}
	switch typBase.Base {
	case "node":
		typ = new(tosca.NodeType)
	case "capability":
		typ = new(tosca.CapabilityType)
	case "relationship":
		typ = new(tosca.RelationshipType)
	default:
		return nil, errors.Errorf("Unknown type:%q", typBase.Base)

	}

	exist, err = storage.GetStore(types.StoreTypeDeployment).Get(typePath, typ)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.Errorf("No type found with name:%q", typeName)
	}

	var mapAttrs map[string]tosca.AttributeDefinition
	switch t := typ.(type) {
	case *tosca.NodeType:
		mapAttrs = t.Attributes
	case *tosca.RelationshipType:
		mapAttrs = t.Attributes
	case *tosca.CapabilityType:
		mapAttrs = t.Attributes
	}

	return mapAttrs, nil
}

func checkIfTypeExists(typePath string) (bool, error) {
	return storage.GetStore(types.StoreTypeDeployment).Exist(typePath)
}

func locateTypePath(deploymentID, typeName string) (string, error) {
	// First check for type in deployment
	typePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", typeName)
	// Check if node type exist
	exist, err := checkIfTypeExists(typePath)
	if err != nil {
		return "", err
	}
	if exist {
		return typePath, nil
	}

	builtinTypesPaths := store.GetCommonsTypesPaths()
	for i := range builtinTypesPaths {
		builtinTypesPaths[i] = path.Join(builtinTypesPaths[i], "types", typeName)
		exist, err := checkIfTypeExists(builtinTypesPaths[i])
		if err != nil {
			return "", err
		}
		if exist {
			return builtinTypesPaths[i], nil
		}
	}

	return "", errors.WithStack(typeMissingError{name: typeName, deploymentID: deploymentID})
}

// GetParentType returns the direct parent type of a given type using the 'derived_from' attributes
//
// An empty string denotes a root type
func GetParentType(ctx context.Context, deploymentID, typeName string) (string, error) {
	if tosca.IsBuiltinType(typeName) {
		return "", nil
	}
	typePath, err := locateTypePath(deploymentID, typeName)
	if err != nil {
		return "", err
	}

	_, typ, err := getTypeBase(typePath)
	if err != nil {
		return "", err
	}
	return typ.DerivedFrom, nil
}

// IsTypeDerivedFrom traverses 'derived_from' to check if type derives from another type
func IsTypeDerivedFrom(ctx context.Context, deploymentID, nodeType, derives string) (bool, error) {
	if nodeType == derives {
		return true, nil
	}
	parent, err := GetParentType(ctx, deploymentID, nodeType)
	if err != nil || parent == "" {
		return false, err
	}
	return IsTypeDerivedFrom(ctx, deploymentID, parent, derives)
}

// GetTypes returns the names of the different types for a given deployment.
func GetTypes(ctx context.Context, deploymentID string) ([]string, error) {
	names := make([]string, 0)
	typs, err := storage.GetStore(types.StoreTypeDeployment).Keys(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types"))
	if err != nil {
		return names, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, t := range typs {
		names = append(names, path.Base(t))
	}

	builtinTypesPaths := store.GetCommonsTypesPaths()
	for i := range builtinTypesPaths {
		builtinTypesPaths[i] = path.Join(builtinTypesPaths[i], "types")
	}
	for _, builtinTypesPath := range builtinTypesPaths {
		typs, err := storage.GetStore(types.StoreTypeDeployment).Keys(builtinTypesPath)
		if err != nil {
			return names, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		for _, t := range typs {
			names = append(names, path.Base(t))
		}
	}

	return names, nil
}

// GetTypeProperties returns the list of properties defined for a given type nam of the specified type tType
// tType can be "node", "relationship", "capability", "artifact", "data", policy"
// It lists only properties defined in the given type not in its parent types.
func GetTypeProperties(ctx context.Context, deploymentID, typeName string, exploreParents bool) ([]string, error) {
	return getTypeAttributesOrProperties(ctx, deploymentID, typeName, "properties", exploreParents)
}

// GetTypeAttributes returns the list of attributes defined for a given type name of the specified type tType
// tType can be "node", "relationship", "capability"
// It lists only attributes defined in the given type not in its parent types.
func GetTypeAttributes(ctx context.Context, deploymentID, typeName string, exploreParents bool) ([]string, error) {
	return getTypeAttributesOrProperties(ctx, deploymentID, typeName, "attributes", exploreParents)
}

func getTypeAttributesOrProperties(ctx context.Context, deploymentID, typeName, paramType string, exploreParents bool) ([]string, error) {
	if tosca.IsBuiltinType(typeName) {
		return nil, nil
	}

	results := make([]string, 0)

	if paramType == "properties" {
		mapProps, err := getTypePropertyDefinitions(deploymentID, typeName)
		if err != nil {
			return nil, err
		}
		for k := range mapProps {
			results = append(results, k)
		}
	} else {
		mapAttrs, err := getTypeAttributeDefinitions(deploymentID, typeName)
		if err != nil {
			return nil, err
		}
		for k := range mapAttrs {
			results = append(results, k)
		}
	}

	if exploreParents {
		parent, err := GetParentType(ctx, deploymentID, typeName)
		if err != nil {
			return nil, err
		}
		// Check parent
		if parent != "" {
			pResults, err := getTypeAttributesOrProperties(ctx, deploymentID, parent, paramType, exploreParents)
			if err != nil {
				return nil, err
			}
			results = append(results, pResults...)
		}
	}
	return results, nil
}

// TypeHasProperty returns true if the type has a property named propertyName defined
// exploreParents switch enable property check on parent types
func TypeHasProperty(ctx context.Context, deploymentID, typeName, propertyName string, exploreParents bool) (bool, error) {
	props, err := GetTypeProperties(ctx, deploymentID, typeName, exploreParents)
	if err != nil {
		return false, err
	}
	return collections.ContainsString(props, propertyName), nil
}

// TypeHasAttribute returns true if the type has a attribute named attributeName defined
// exploreParents switch enable attribute check on parent types
func TypeHasAttribute(ctx context.Context, deploymentID, typeName, attributeName string, exploreParents bool) (bool, error) {
	attrs, err := GetTypeAttributes(ctx, deploymentID, typeName, exploreParents)
	if err != nil {
		return false, err
	}
	return collections.ContainsString(attrs, attributeName), nil
}

// getTypeDefaultProperty checks if a type has a default value for a given property.
// It returns true if a default value is found false otherwise as first return parameter.
// If no default value is found in a given type then the derived_from hierarchy is explored to find the default value.
// The second boolean result indicates if the result is a TOSCA Function that should be evaluated in the caller context.
func getTypeDefaultProperty(ctx context.Context, deploymentID, typeName, propertyName string, nestedKeys ...string) (*TOSCAValue, bool, error) {
	return getTypeDefaultAttributeOrProperty(ctx, deploymentID, typeName, propertyName, true, nestedKeys...)
}

// getTypeDefaultAttribute checks if a node type has a default value for a given attribute.
// It returns true if a default value is found false otherwise as first return parameter.
// If no default value is found in a given type then the derived_from hierarchy is explored to find the default value.
// The second boolean result indicates if the result is a TOSCA Function that should be evaluated in the caller context.
func getTypeDefaultAttribute(ctx context.Context, deploymentID, typeName, attributeName string, nestedKeys ...string) (*TOSCAValue, bool, error) {
	return getTypeDefaultAttributeOrProperty(ctx, deploymentID, typeName, attributeName, false, nestedKeys...)
}

// getTypeDefaultProperty checks if a type has a default value for a given property or attribute.
// It returns true if a default value is found false otherwise as first return parameter.
// If no default value is found in a given type then the derived_from hierarchy is explored to find the default value.
// The second boolean result indicates if the result is a TOSCA Function that should be evaluated in the caller context.
func getTypeDefaultAttributeOrProperty(ctx context.Context, deploymentID, typeName, propertyName string, isProperty bool, nestedKeys ...string) (*TOSCAValue, bool, error) {
	var vaDef *tosca.ValueAssignment
	if isProperty {
		def, err := getTypePropertyDefinition(ctx, deploymentID, typeName, propertyName)
		if err != nil {
			return nil, false, err
		}
		if def != nil {
			vaDef = def.Default
		}
	} else {
		def, err := getTypeAttributeDefinition(ctx, deploymentID, typeName, propertyName)
		if err != nil {
			return nil, false, err
		}
		if def != nil {
			vaDef = def.Default
		}
	}

	baseDataType, err := getTypePropertyOrAttributeDataType(ctx, deploymentID, typeName, propertyName, isProperty)
	if err != nil {
		return nil, false, err
	}

	return getValueAssignmentWithoutResolve(ctx, deploymentID, vaDef, baseDataType, nestedKeys...)
}

// IsTypePropertyRequired checks if a property defined in a given type is required.
//
// As per the TOSCA specification a property is considered as required by default.
// An error is returned if the given type doesn't define the given property.
func IsTypePropertyRequired(ctx context.Context, deploymentID, typeName, propertyName string) (bool, error) {
	return isTypePropertyRequired(ctx, deploymentID, typeName, propertyName)
}

func isTypePropertyRequired(ctx context.Context, deploymentID, typeName, propertyName string) (bool, error) {
	// Required is true by default
	required := true
	def, err := getTypePropertyDefinition(ctx, deploymentID, typeName, propertyName)
	if err != nil {
		return false, err
	}
	if def != nil && def.Required != nil {
		required = *def.Required
	}

	return required, nil
}

func getTypePropertyDefinition(ctx context.Context, deploymentID, typeName, propertyName string) (*tosca.PropertyDefinition, error) {
	mapProps, err := getTypePropertyDefinitions(deploymentID, typeName)
	if err != nil {
		return nil, err
	}

	propDef, is := mapProps[propertyName]
	if is {
		return &propDef, nil
	}

	// Check parent
	parent, err := GetParentType(ctx, deploymentID, typeName)
	if err != nil {
		return nil, err
	}
	if parent != "" {
		return getTypePropertyDefinition(ctx, deploymentID, parent, propertyName)
	}

	// Not found
	return nil, nil
}

func getTypeAttributeDefinition(ctx context.Context, deploymentID, typeName, attributeName string) (*tosca.AttributeDefinition, error) {
	typ := new(tosca.NodeType)
	err := getTypeStruct(deploymentID, typeName, typ)
	if err != nil {
		return nil, err
	}

	attrDef, is := typ.Attributes[attributeName]
	if is && &attrDef != nil {
		return &attrDef, nil
	}

	// Check parent
	if typ.DerivedFrom != "" {
		return getTypeAttributeDefinition(ctx, deploymentID, typ.DerivedFrom, attributeName)
	}

	// Not found
	return nil, nil
}

// GetTypeImportPath returns the import path relative to the root of a CSAR of a given TOSCA type.
//
// This is particularly useful for resolving artifacts and implementation
func GetTypeImportPath(ctx context.Context, deploymentID, typeName string) (string, error) {
	typePath, err := locateTypePath(deploymentID, typeName)
	if err != nil {
		return "", err
	}

	_, typ, err := getTypeBase(typePath)
	if err != nil {
		return "", err
	}
	// Can be empty if type is defined into the root topology
	return typ.ImportPath, nil
}
