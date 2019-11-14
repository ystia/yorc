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
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/deployments/store"
	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/tosca"
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

func getType(deploymentID, typeName string, typ interface{}) (bool, error) {
	typePath, err := locateTypePath(deploymentID, typeName)
	if err != nil {
		return false, err
	}

	return storage.GetStore(types.StoreTypeDeployment).Get(typePath, typ)
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

// GetTypeProperties returns the list of properties defined in a given type
//
// It lists only properties defined in the given type not in its parent types.
func GetTypeProperties(ctx context.Context, deploymentID, typeName string, exploreParents bool) ([]string, error) {
	return getTypeAttributesOrProperties(ctx, deploymentID, typeName, "properties", exploreParents)
}

// GetTypeAttributes returns the list of attributes defined in a given type
//
// It lists only attributes defined in the given type not in its parent types.
func GetTypeAttributes(ctx context.Context, deploymentID, typeName string, exploreParents bool) ([]string, error) {
	return getTypeAttributesOrProperties(ctx, deploymentID, typeName, "attributes", exploreParents)
}

func getTypeAttributesOrProperties(ctx context.Context, deploymentID, typeName, paramType string, exploreParents bool) ([]string, error) {
	if tosca.IsBuiltinType(typeName) {
		return nil, nil
	}
	typePath, err := locateTypePath(deploymentID, typeName)
	if err != nil {
		return nil, err
	}

	result, err := consulutil.GetKeys(path.Join(typePath, paramType))
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for i := range result {
		result[i] = path.Base(result[i])
	}
	if exploreParents {
		parentType, err := GetParentType(ctx, deploymentID, typeName)
		if err != nil {
			return nil, err
		}
		if parentType != "" {
			parentRes, err := getTypeAttributesOrProperties(ctx, deploymentID, parentType, paramType, true)
			if err != nil {
				return nil, err
			}
			result = append(result, parentRes...)
		}
	}
	return result, nil
}

// TypeHasProperty returns true if the type has a property named propertyName defined
//
// exploreParents switch enable property check on parent types
func TypeHasProperty(ctx context.Context, deploymentID, typeName, propertyName string, exploreParents bool) (bool, error) {
	props, err := GetTypeProperties(ctx, deploymentID, typeName, exploreParents)
	if err != nil {
		return false, err
	}
	return collections.ContainsString(props, propertyName), nil
}

// TypeHasAttribute returns true if the type has a attribute named attributeName defined
//
// exploreParents switch enable attribute check on parent types
func TypeHasAttribute(ctx context.Context, deploymentID, typeName, attributeName string, exploreParents bool) (bool, error) {
	attrs, err := GetTypeAttributes(ctx, deploymentID, typeName, exploreParents)
	if err != nil {
		return false, err
	}
	return collections.ContainsString(attrs, attributeName), nil
}

// getTypeDefaultProperty checks if a type has a default value for a given property.
//
// It returns true if a default value is found false otherwise as first return parameter.
// If no default value is found in a given type then the derived_from hierarchy is explored to find the default value.
// The second boolean result indicates if the result is a TOSCA Function that should be evaluated in the caller context.
func getTypeDefaultProperty(ctx context.Context, deploymentID, typeName, propertyName string, nestedKeys ...string) (*TOSCAValue, bool, error) {
	return getTypeDefaultAttributeOrProperty(ctx, deploymentID, typeName, propertyName, true, nestedKeys...)
}

// getTypeDefaultAttribute checks if a type has a default value for a given attribute.
//
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
	hasProp, prop, err := getPropertyDefinition(ctx, deploymentID, typeName, propertyName, isProperty)
	// If this type doesn't contains the property lets continue to explore the type hierarchy
	if err != nil {
		return nil, false, err
	}
	if hasProp && prop.Default != nil {
		// Just need to resolve the va
		va := prop.Default
		switch va.Type {
		case tosca.ValueAssignmentLiteral:
			return &TOSCAValue{Value: va.String()}, va.Type == tosca.ValueAssignmentFunction, nil
		case tosca.ValueAssignmentFunction, tosca.ValueAssignmentList, tosca.ValueAssignmentMap:
			return &TOSCAValue{Value: va.String()}, false, nil
		}
	}
	// No default in this type
	// Lets look at parent type
	parentType, err := GetParentType(ctx, deploymentID, typeName)
	if err != nil || parentType == "" {
		return nil, false, err
	}
	return getTypeDefaultAttributeOrProperty(ctx, deploymentID, parentType, propertyName, isProperty, nestedKeys...)
}

// IsTypePropertyRequired checks if a property defined in a given type is required.
//
// As per the TOSCA specification a property is considered as required by default.
// An error is returned if the given type doesn't define the given property.
func IsTypePropertyRequired(ctx context.Context, deploymentID, typeName, propertyName string) (bool, error) {
	return isTypePropOrAttrRequired(ctx, deploymentID, typeName, typeName, propertyName, "property")
}

// IsTypeAttributeRequired checks if a attribute defined in a given type is required.
//
// As per the TOSCA specification a attribute is considered as required by default.
// An error is returned if the given type doesn't define the given attribute.
func IsTypeAttributeRequired(ctx context.Context, deploymentID, typeName, attributeName string) (bool, error) {
	return isTypePropOrAttrRequired(ctx, deploymentID, typeName, typeName, attributeName, "attribute")
}

func isTypePropOrAttrRequired(ctx context.Context, deploymentID, typeName, originalTypeName, elemName, elemType string) (bool, error) {
	var hasElem bool
	var err error
	if elemType == "property" {
		hasElem, err = TypeHasProperty(ctx, deploymentID, typeName, elemName, false)
	} else {
		hasElem, err = TypeHasAttribute(ctx, deploymentID, typeName, elemName, false)
	}
	if err != nil {
		return false, err
	}
	if !hasElem {
		parentType, err := GetParentType(ctx, deploymentID, typeName)
		if err != nil {
			return false, err
		}
		if parentType == "" {
			return false, errors.Errorf("type %q doesn't define %s %q can't check if it is required or not", originalTypeName, elemType, elemName)
		}
		return isTypePropOrAttrRequired(ctx, deploymentID, parentType, originalTypeName, elemName, elemType)
	}

	var t string
	if elemType == "property" {
		t = "properties"
	} else {
		t = "attributes"
	}
	typePath, err := locateTypePath(deploymentID, typeName)
	if err != nil {
		return false, err
	}
	reqPath := path.Join(typePath, t, elemName, "required")
	exist, value, err := consulutil.GetStringValue(reqPath)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist || value == "" {
		// Required by default
		return true, nil
	}
	// Not required only if explicitly set to false
	return !(strings.ToLower(value) == "false"), nil
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
