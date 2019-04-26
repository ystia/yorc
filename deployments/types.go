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

	"github.com/blang/semver"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/helper/collections"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/tosca"
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

func getLatestBuiltinTypesPaths(kv *api.KV) ([]string, error) {
	keys, _, err := kv.Keys(consulutil.BuiltinTypesKVPrefix+"/", "/", nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	paths := make([]string, 0, len(keys))
	for _, builtinTypesPath := range keys {
		versions, _, err := kv.Keys(builtinTypesPath, "/", nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}

		if len(versions) == 0 {
			continue
		}
		typePath := path.Join(versions[0], "types")
		if len(versions) >= 1 {
			var maxVersion semver.Version
			for _, v := range versions {
				version, err := semver.Make(path.Base(v))
				if err == nil && version.GTE(maxVersion) {
					maxVersion = version
				}
			}
			typePath = path.Join(builtinTypesPath, maxVersion.String(), "types")
		}

		paths = append(paths, typePath)
	}
	return paths, nil
}

func locateTypePath(kv *api.KV, deploymentID, typeName string) (string, error) {
	// First check for type in deployment
	typePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", typeName)
	// Check if node type exist
	exists, err := consulutil.HasChildrenKeys(kv, typePath)
	if err != nil {
		return "", err
	}
	if exists {
		return typePath, nil
	}

	builtinTypesPaths, err := getLatestBuiltinTypesPaths(kv)
	if err != nil {
		return "", err
	}

	for _, builtinTypesPath := range builtinTypesPaths {
		typePath = path.Join(builtinTypesPath, typeName)
		exists, err := consulutil.HasChildrenKeys(kv, typePath)
		if err != nil {
			return "", err
		}
		if exists {
			return typePath, nil
		}
	}

	return "", errors.WithStack(typeMissingError{name: typeName, deploymentID: deploymentID})
}

// GetParentType returns the direct parent type of a given type using the 'derived_from' attributes
//
// An empty string denotes a root type
func GetParentType(kv *api.KV, deploymentID, typeName string) (string, error) {
	if tosca.IsBuiltinType(typeName) {
		return "", nil
	}
	typePath, err := locateTypePath(kv, deploymentID, typeName)
	if err != nil {
		return "", err
	}

	kvp, _, err := kv.Get(path.Join(typePath, "derived_from"), nil)
	if err != nil {
		return "", errors.Wrap(err, "Consul access error: ")
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return "", nil
	}
	return string(kvp.Value), nil
}

// IsTypeDerivedFrom traverses 'derived_from' to check if type derives from another type
func IsTypeDerivedFrom(kv *api.KV, deploymentID, nodeType, derives string) (bool, error) {
	if nodeType == derives {
		return true, nil
	}
	parent, err := GetParentType(kv, deploymentID, nodeType)
	if err != nil || parent == "" {
		return false, err
	}
	return IsTypeDerivedFrom(kv, deploymentID, parent, derives)
}

// GetTypes returns the names of the different types for a given deployment.
func GetTypes(kv *api.KV, deploymentID string) ([]string, error) {
	names := make([]string, 0)
	types, _, err := kv.Keys(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types")+"/", "/", nil)
	if err != nil {
		return names, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, t := range types {
		names = append(names, path.Base(t))
	}

	builtinPaths, err := getLatestBuiltinTypesPaths(kv)
	if err != nil {
		return names, err
	}
	for _, builtinPath := range builtinPaths {
		types, _, err := kv.Keys(builtinPath+"/", "/", nil)
		if err != nil {
			return names, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		for _, t := range types {
			names = append(names, path.Base(t))
		}
	}

	return names, nil
}

// GetTypeProperties returns the list of properties defined in a given type
//
// It lists only properties defined in the given type not in its parent types.
func GetTypeProperties(kv *api.KV, deploymentID, typeName string, exploreParents bool) ([]string, error) {
	return getTypeAttributesOrProperties(kv, deploymentID, typeName, "properties", exploreParents)
}

// GetTypeAttributes returns the list of attributes defined in a given type
//
// It lists only attributes defined in the given type not in its parent types.
func GetTypeAttributes(kv *api.KV, deploymentID, typeName string, exploreParents bool) ([]string, error) {
	return getTypeAttributesOrProperties(kv, deploymentID, typeName, "attributes", exploreParents)
}

func getTypeAttributesOrProperties(kv *api.KV, deploymentID, typeName, paramType string, exploreParents bool) ([]string, error) {
	if tosca.IsBuiltinType(typeName) {
		return nil, nil
	}
	typePath, err := locateTypePath(kv, deploymentID, typeName)
	if err != nil {
		return nil, err
	}
	result, _, err := kv.Keys(path.Join(typePath, paramType)+"/", "/", nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for i := range result {
		result[i] = path.Base(result[i])
	}
	if exploreParents {
		parentType, err := GetParentType(kv, deploymentID, typeName)
		if err != nil {
			return nil, err
		}
		if parentType != "" {
			parentRes, err := getTypeAttributesOrProperties(kv, deploymentID, parentType, paramType, true)
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
func TypeHasProperty(kv *api.KV, deploymentID, typeName, propertyName string, exploreParents bool) (bool, error) {
	props, err := GetTypeProperties(kv, deploymentID, typeName, exploreParents)
	if err != nil {
		return false, err
	}
	return collections.ContainsString(props, propertyName), nil
}

// TypeHasAttribute returns true if the type has a attribute named attributeName defined
//
// exploreParents switch enable attribute check on parent types
func TypeHasAttribute(kv *api.KV, deploymentID, typeName, attributeName string, exploreParents bool) (bool, error) {
	attrs, err := GetTypeAttributes(kv, deploymentID, typeName, exploreParents)
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
func getTypeDefaultProperty(kv *api.KV, deploymentID, typeName, propertyName string, nestedKeys ...string) (*TOSCAValue, bool, error) {
	return getTypeDefaultAttributeOrProperty(kv, deploymentID, typeName, propertyName, true, nestedKeys...)
}

// getTypeDefaultAttribute checks if a type has a default value for a given attribute.
//
// It returns true if a default value is found false otherwise as first return parameter.
// If no default value is found in a given type then the derived_from hierarchy is explored to find the default value.
// The second boolean result indicates if the result is a TOSCA Function that should be evaluated in the caller context.
func getTypeDefaultAttribute(kv *api.KV, deploymentID, typeName, attributeName string, nestedKeys ...string) (*TOSCAValue, bool, error) {
	return getTypeDefaultAttributeOrProperty(kv, deploymentID, typeName, attributeName, false, nestedKeys...)
}

// getTypeDefaultProperty checks if a type has a default value for a given property or attribute.
// It returns true if a default value is found false otherwise as first return parameter.
// If no default value is found in a given type then the derived_from hierarchy is explored to find the default value.
// The second boolean result indicates if the result is a TOSCA Function that should be evaluated in the caller context.
func getTypeDefaultAttributeOrProperty(kv *api.KV, deploymentID, typeName, propertyName string, isProperty bool, nestedKeys ...string) (*TOSCAValue, bool, error) {

	// If this type doesn't contains the property lets continue to explore the type hierarchy
	var hasProp bool
	var err error
	if isProperty {
		hasProp, err = TypeHasProperty(kv, deploymentID, typeName, propertyName, false)
	} else {
		hasProp, err = TypeHasAttribute(kv, deploymentID, typeName, propertyName, false)
	}
	if err != nil {
		return nil, false, err
	}
	if hasProp {
		typePath, err := locateTypePath(kv, deploymentID, typeName)
		if err != nil {
			return nil, false, err
		}
		var t string
		if isProperty {
			t = "properties"
		} else {
			t = "attributes"
		}
		defaultPath := path.Join(typePath, t, propertyName, "default")

		baseDataType, err := getTypePropertyOrAttributeDataType(kv, deploymentID, typeName, propertyName, isProperty)
		if err != nil {
			return nil, false, err
		}

		result, isFunction, err := getValueAssignmentWithoutResolve(kv, deploymentID, defaultPath, baseDataType, nestedKeys...)
		if err != nil || result != nil {
			return result, isFunction, errors.Wrapf(err, "Failed to get default %s %q for type %q", t, propertyName, typeName)
		}
	}
	// No default in this type
	// Lets look at parent type
	parentType, err := GetParentType(kv, deploymentID, typeName)
	if err != nil || parentType == "" {
		return nil, false, err
	}
	return getTypeDefaultAttributeOrProperty(kv, deploymentID, parentType, propertyName, isProperty, nestedKeys...)
}

// IsTypePropertyRequired checks if a property defined in a given type is required.
//
// As per the TOSCA specification a property is considered as required by default.
// An error is returned if the given type doesn't define the given property.
func IsTypePropertyRequired(kv *api.KV, deploymentID, typeName, propertyName string) (bool, error) {
	return isTypePropOrAttrRequired(kv, deploymentID, typeName, typeName, propertyName, "property")
}

// IsTypeAttributeRequired checks if a attribute defined in a given type is required.
//
// As per the TOSCA specification a attribute is considered as required by default.
// An error is returned if the given type doesn't define the given attribute.
func IsTypeAttributeRequired(kv *api.KV, deploymentID, typeName, attributeName string) (bool, error) {
	return isTypePropOrAttrRequired(kv, deploymentID, typeName, typeName, attributeName, "attribute")
}

func isTypePropOrAttrRequired(kv *api.KV, deploymentID, typeName, originalTypeName, elemName, elemType string) (bool, error) {
	var hasElem bool
	var err error
	if elemType == "property" {
		hasElem, err = TypeHasProperty(kv, deploymentID, typeName, elemName, false)
	} else {
		hasElem, err = TypeHasAttribute(kv, deploymentID, typeName, elemName, false)
	}
	if err != nil {
		return false, err
	}
	if !hasElem {
		parentType, err := GetParentType(kv, deploymentID, typeName)
		if err != nil {
			return false, err
		}
		if parentType == "" {
			return false, errors.Errorf("type %q doesn't define %s %q can't check if it is required or not", originalTypeName, elemType, elemName)
		}
		return isTypePropOrAttrRequired(kv, deploymentID, parentType, originalTypeName, elemName, elemType)
	}

	var t string
	if elemType == "property" {
		t = "properties"
	} else {
		t = "attributes"
	}
	typePath, err := locateTypePath(kv, deploymentID, typeName)
	if err != nil {
		return false, err
	}
	reqPath := path.Join(typePath, t, elemName, "required")
	kvp, _, err := kv.Get(reqPath, nil)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		// Required by default
		return true, nil
	}
	// Not required only if explicitly set to false
	return !(strings.ToLower(string(kvp.Value)) == "false"), nil
}

// GetTypeImportPath returns the import path relative to the root of a CSAR of a given TOSCA type.
//
// This is particularly useful for resolving artifacts and implementation
func GetTypeImportPath(kv *api.KV, deploymentID, typeName string) (string, error) {
	typePath, err := locateTypePath(kv, deploymentID, typeName)
	if err != nil {
		return "", err
	}
	kvp, _, err := kv.Get(path.Join(typePath, "importPath"), nil)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	// Can be empty if type is defined into the root topology
	if kvp == nil {
		return "", nil
	}
	return string(kvp.Value), nil
}
