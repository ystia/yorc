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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/ystia/yorc/v3/events"
	"github.com/ystia/yorc/v3/helper/collections"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/helper/stringutil"
	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/prov"
	"github.com/ystia/yorc/v3/tosca"
)

// IsOperationNotImplemented checks if a given error is an error indicating that an operation is not implemented
func IsOperationNotImplemented(err error) bool {
	err = errors.Cause(err)
	_, ok := err.(operationNotImplemented)
	return ok
}

type operationNotImplemented struct {
	msg string
}

func (oni operationNotImplemented) Error() string {
	return oni.msg
}

// IsInputNotFound checks if a given error is an error indicating that an input was not found in an operation
func IsInputNotFound(err error) bool {
	err = errors.Cause(err)
	_, ok := err.(inputNotFound)
	return ok
}

type inputNotFound struct {
	inputName          string
	operationName      string
	implementationType string
}

func (inf inputNotFound) Error() string {
	return fmt.Sprintf("input %q not found for operation %q implemented in type %q", inf.inputName, inf.operationName, inf.implementationType)
}

const implementationArtifactsExtensionsPath = "implementation_artifacts_extensions"

// GetOperationPathAndPrimaryImplementation traverses the type hierarchy to find an operation matching the given operationName.
//
// First, it checks the node template if operation implementation is present
// Next it gets down to types hierarchy
// Once found it returns the path to the operation and the value of its primary implementation.
// If the operation is not found in the node template or in the type hierarchy then empty strings are returned.
func GetOperationPathAndPrimaryImplementation(kv *api.KV, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName string) (string, string, error) {
	var importPath, typeOrNodeTemplate string
	var err error
	if nodeTemplateImpl == "" {
		importPath, err = GetTypeImportPath(kv, deploymentID, nodeTypeImpl)
		if err != nil {
			return "", "", err
		}
		typeOrNodeTemplate = nodeTypeImpl
	} else {
		typeOrNodeTemplate = nodeTemplateImpl
	}

	operationPath, err := getOperationPath(kv, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName)
	if err != nil {
		return "", "", errors.Wrapf(err, "Failed to retrieve primary implementation for operation %q on template/type %q", operationName, typeOrNodeTemplate)
	}
	kvp, _, err := kv.Get(path.Join(operationPath, "implementation/primary"), nil)
	if err != nil {
		return "", "", errors.Wrapf(err, "Failed to retrieve primary implementation for operation %q on template/type %q", operationName, typeOrNodeTemplate)
	}
	if kvp != nil && len(kvp.Value) > 0 {
		if importPath != "" {
			return operationPath, path.Join(importPath, string(kvp.Value)), nil
		}
		return operationPath, string(kvp.Value), nil
	}

	// then check operation file
	kvp, _, err = kv.Get(path.Join(operationPath, "implementation/file"), nil)
	if err != nil {
		return "", "", errors.Wrapf(err, "Failed to retrieve primary implementation for operation %q on template/type %q", operationName, typeOrNodeTemplate)
	}
	if kvp != nil && len(kvp.Value) > 0 {
		if importPath != "" {
			return operationPath, path.Join(importPath, string(kvp.Value)), nil
		}
		return operationPath, string(kvp.Value), nil
	}

	// Not found here check the type hierarchy
	parentType, err := GetParentType(kv, deploymentID, nodeTypeImpl)
	if err != nil || parentType == "" {
		return "", "", err
	}

	return getOperationPathAndPrimaryImplementationForNodeType(kv, deploymentID, parentType, operationName)
}

func getOperationPathAndPrimaryImplementationForNodeType(kv *api.KV, deploymentID, nodeType, operationName string) (string, string, error) {
	// First check if operation exists in current nodeType
	operationPath, err := getOperationPath(kv, deploymentID, "", nodeType, operationName)
	if err != nil {
		return "", "", err
	}
	importPath, err := GetTypeImportPath(kv, deploymentID, nodeType)
	if err != nil {
		return "", "", err
	}
	kvp, _, err := kv.Get(path.Join(operationPath, "implementation/primary"), nil)
	if err != nil {
		return "", "", errors.Wrapf(err, "Failed to retrieve primary implementation for operation %q on type %q", operationName, nodeType)
	}
	if kvp != nil && len(kvp.Value) > 0 {
		return operationPath, path.Join(importPath, string(kvp.Value)), nil
	}

	// then check operation file
	kvp, _, err = kv.Get(path.Join(operationPath, "implementation/file"), nil)
	if err != nil {
		return "", "", errors.Wrapf(err, "Failed to retrieve primary implementation for operation %q on type %q", operationName, nodeType)
	}
	if kvp != nil && len(kvp.Value) > 0 {
		return operationPath, path.Join(importPath, string(kvp.Value)), nil
	}

	// Not found here check the type hierarchy
	parentType, err := GetParentType(kv, deploymentID, nodeType)
	if err != nil || parentType == "" {
		return "", "", err
	}

	return getOperationPathAndPrimaryImplementationForNodeType(kv, deploymentID, parentType, operationName)
}

// This function return the path for a given operation
func getOperationPath(kv *api.KV, deploymentID, nodeTemplate, nodeType, operationName string) (string, error) {
	opPath, _, err := getOperationAndInterfacePath(kv, deploymentID, nodeTemplate, nodeType, operationName)
	return opPath, err
}

// This function return the path for a given operation and the path of its interface
// It handles the ways that implementation is a node template or a node type
func getOperationAndInterfacePath(kv *api.KV, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName string) (string, string, error) {
	var operationPath, interfacePath string
	opShortName := stringutil.GetLastElement(operationName, ".")

	interfaceName := stringutil.GetAllExceptLastElement(operationName, ".")
	if strings.HasPrefix(interfaceName, tosca.StandardInterfaceName) {
		interfaceName = strings.Replace(interfaceName, tosca.StandardInterfaceName, tosca.StandardInterfaceShortName, 1)
	} else if strings.HasPrefix(interfaceName, tosca.ConfigureInterfaceName) {
		interfaceName = strings.Replace(interfaceName, tosca.ConfigureInterfaceName, tosca.ConfigureInterfaceShortName, 1)
	}
	if nodeTemplateImpl != "" {
		operationPath = path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeTemplateImpl, "interfaces", interfaceName, opShortName)
		interfacePath = path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeTemplateImpl, "interfaces", interfaceName)
	} else {
		typePath, err := locateTypePath(kv, deploymentID, nodeTypeImpl)
		if err != nil {
			return "", "", err
		}
		operationPath = path.Join(typePath, "interfaces", interfaceName, opShortName)
		interfacePath = path.Join(typePath, "interfaces", interfaceName)
	}

	return operationPath, interfacePath, nil
}

// GetRelationshipTypeImplementingAnOperation  returns the first (bottom-up) type in the type hierarchy of a given relationship that implements a given operation
//
// An error is returned if the operation is not found in the type hierarchy
func GetRelationshipTypeImplementingAnOperation(kv *api.KV, deploymentID, nodeName, operationName, requirementIndex string) (string, error) {
	relTypeInit, err := GetRelationshipForRequirement(kv, deploymentID, nodeName, requirementIndex)
	if err != nil {
		return "", err
	}
	relType := relTypeInit
	for relType != "" {
		operationPath, err := getOperationPath(kv, deploymentID, "", relType, operationName)
		if err != nil {
			return "", err
		}
		keys, _, err := kv.Keys(operationPath+"/", "/", nil)
		if err != nil {
			return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if len(keys) > 0 {
			return relType, nil
		}
		relType, err = GetParentType(kv, deploymentID, relType)
	}
	return "", operationNotImplemented{msg: fmt.Sprintf("Operation %q not found in the type hierarchy of relationship %q", operationName, relTypeInit)}
}

// GetNodeTypeImplementingAnOperation returns the first (bottom-up) type in the type hierarchy of a given node that implements a given operation
//
// This is a shortcut for retrieving the node type and calling the GetTypeImplementingAnOperation() function
func GetNodeTypeImplementingAnOperation(kv *api.KV, deploymentID, nodeName, operationName string) (string, error) {
	nodeType, err := GetNodeType(kv, deploymentID, nodeName)
	log.Debugf("[GetNodeTypeImplementingAnOperation] nodeType=%q", nodeType)
	if err != nil {
		return "", err
	}
	t, err := GetTypeImplementingAnOperation(kv, deploymentID, nodeType, operationName)
	return t, errors.Wrapf(err, "operation not found for node %q", nodeName)
}

// IsNodeTemplateImplementingOperation returns true if the node implements the defined operation
func IsNodeTemplateImplementingOperation(kv *api.KV, deploymentID, nodeName, operationName string) (bool, error) {
	operationPath, err := getOperationPath(kv, deploymentID, nodeName, "", operationName)
	if err != nil {
		return false, errors.Wrapf(err, "Can't define if operation with name:%q exists for node %q", operationName, nodeName)
	}
	keys, _, err := kv.Keys(operationPath+"/", "/", nil)
	if err != nil {
		return false, errors.Wrapf(err, "Can't define if operation with name:%q exists for node %q", operationName, nodeName)
	}
	if keys == nil || len(keys) == 0 {
		return false, nil
	}
	return true, nil
}

// GetTypeImplementingAnOperation returns the first (bottom-up) type in the type hierarchy that implements a given operation
//
// An error is returned if the operation is not found in the type hierarchy
func GetTypeImplementingAnOperation(kv *api.KV, deploymentID, typeName, operationName string) (string, error) {
	log.Debugf("[GetTypeImplementingAnOperation] operationName=%q", operationName)
	implType := typeName
	for implType != "" {
		log.Debugf("[GetTypeImplementingAnOperation] implType=%q", implType)
		operationPath, err := getOperationPath(kv, deploymentID, "", implType, operationName)
		if err != nil {
			return "", err
		}
		log.Debugf("[GetTypeImplementingAnOperation] operationPath=%q", operationPath)

		keys, _, err := kv.Keys(operationPath+"/", "/", nil)
		if err != nil {
			return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if len(keys) > 0 {
			return implType, nil
		}
		implType, err = GetParentType(kv, deploymentID, implType)
	}
	return "", operationNotImplemented{msg: fmt.Sprintf("operation %q not found in the type hierarchy of type %q", operationName, typeName)}
}

// GetOperationImplementationType allows you when the implementation of an operation is an artifact to retrieve the type of this artifact
func GetOperationImplementationType(kv *api.KV, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName string) (string, error) {
	operationPath, err := getOperationPath(kv, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName)
	if err != nil {
		return "", errors.Wrap(err, "Fail to get the type of operation implementation")
	}
	kvp, _, err := kv.Get(path.Join(operationPath, "implementation/type"), nil)
	if err != nil {
		return "", errors.Wrap(err, "Fail to get the type of operation implementation")
	}

	if kvp != nil && len(kvp.Value) > 0 {
		return string(kvp.Value), nil
	}

	var nodeOrTypeImpl string
	if nodeTemplateImpl != "" {
		nodeOrTypeImpl = nodeTemplateImpl
	} else {
		nodeOrTypeImpl = nodeTypeImpl
	}

	kvp, _, err = kv.Get(path.Join(operationPath, "implementation/primary"), nil)
	if err != nil {
		return "", errors.Wrap(err, "Fail to get the type of operation primary implementation")
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return "", errors.Errorf("Failed to resolve implementation for operation %q in node template/type %q", operationName, nodeOrTypeImpl)
	}

	primary := string(kvp.Value)
	primarySlice := strings.Split(primary, ".")
	ext := primarySlice[len(primarySlice)-1]
	artImpl, err := GetImplementationArtifactForExtension(kv, deploymentID, ext)
	if err != nil {
		return "", err
	}
	if artImpl == "" {
		return "", errors.Errorf("Failed to resolve implementation artifact for node template/type %q, operation %q, implementation %q and extension %q", nodeOrTypeImpl, operationName, primary, ext)
	}
	return artImpl, nil

}

func getOperationImplementation(kv *api.KV, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName, implementationType string) (string, error) {
	operationPath, err := getOperationPath(kv, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName)
	if err != nil {
		return "", errors.Wrap(err, "Fail to get the file of operation implementation")
	}
	kvp, _, err := kv.Get(path.Join(operationPath, "implementation", implementationType), nil)
	if err != nil {
		return "", errors.Wrap(err, "Fail to get the file of operation implementation")
	}

	if kvp == nil {
		return "", errors.Errorf("Operation type not found for %q", operationName)
	}

	return string(kvp.Value), nil
}

// GetOperationImplementationFile allows you when the implementation of an operation is an artifact to retrieve the file of this artifact
//
// The returned file is the raw value. To have a file with a path relative to the root of the deployment use GetOperationImplementationFileWithRelativePath()
func GetOperationImplementationFile(kv *api.KV, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName string) (string, error) {
	return getOperationImplementation(kv, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName, "file")
}

// GetOperationImplementationRepository allows you when the implementation of an operation is an artifact to retrieve the repository of this artifact
func GetOperationImplementationRepository(kv *api.KV, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName string) (string, error) {
	return getOperationImplementation(kv, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName, "repository")
}

// GetOperationImplementationFileWithRelativePath allows you when the implementation of an operation
// is an artifact to retrieve the file of this artifact
//
// The returned file is relative to the root of the deployment. To have the raw value use GetOperationImplementationFile()
func GetOperationImplementationFileWithRelativePath(kv *api.KV, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName string) (string, error) {
	file, err := GetOperationImplementationFile(kv, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName)
	if err != nil {
		return file, err
	}
	if nodeTemplateImpl != "" {
		return file, err
	}
	// If implementation is a node type, import path must be resolved
	importPath, err := GetTypeImportPath(kv, deploymentID, nodeTypeImpl)
	return path.Join(importPath, file), err
}

// GetOperationOutputForNode return a map with in index the instance number and in value the result of the output
// The "params" parameter is necessary to pass the path of the output
func GetOperationOutputForNode(kv *api.KV, deploymentID, nodeName, instanceName, interfaceName, operationName, outputName string) (string, error) {
	instancesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName)

	output, _, err := kv.Get(filepath.Join(instancesPath, instanceName, "outputs", strings.ToLower(interfaceName), strings.ToLower(operationName), outputName), nil)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if output != nil && len(output.Value) > 0 {
		return string(output.Value), nil
	}
	// Look at host node
	var host string
	host, err = GetHostedOnNode(kv, deploymentID, nodeName)
	if err != nil {
		return "", err
	}
	if host != "" {
		// TODO we consider that instance name is the same for the host but we should not
		return GetOperationOutputForNode(kv, deploymentID, host, instanceName, interfaceName, operationName, outputName)
	}
	return "", nil
}

// GetOperationOutputForRelationship retrieves an operation output for a relationship
// The returned value may be empty if the operation output could not be retrieved
func GetOperationOutputForRelationship(kv *api.KV, deploymentID, nodeName, instanceName, requirementIndex, interfaceName, operationName, outputName string) (string, error) {
	result, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances", nodeName, requirementIndex, instanceName, "outputs", strings.ToLower(path.Join(interfaceName, operationName)), outputName), nil)
	if err != nil {
		return "", err
	}

	if result == nil || len(result.Value) == 0 {
		return "", nil
	}
	return string(result.Value), nil
}

func getOperationOutputForRequirements(kv *api.KV, deploymentID, nodeName, instanceName, interfaceName, operationName, outputName string) (string, error) {
	reqIndexes, err := GetRequirementsIndexes(kv, deploymentID, nodeName)
	if err != nil {
		return "", err
	}
	for _, reqIndex := range reqIndexes {
		result, err := GetOperationOutputForRelationship(kv, deploymentID, nodeName, instanceName, reqIndex, interfaceName, operationName, outputName)
		if err != nil || result != "" {
			return result, err
		}
	}
	return "", nil
}

// GetImplementationArtifactForExtension returns the implementation artifact type for a given extension.
//
// If the extension is unknown then an empty string is returned
func GetImplementationArtifactForExtension(kv *api.KV, deploymentID, extension string) (string, error) {
	extension = strings.ToLower(extension)
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", implementationArtifactsExtensionsPath, extension), nil)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil {
		return "", nil
	}
	return string(kvp.Value), nil
}

// GetImplementationArtifactForOperation returns the implementation artifact type for a given operation.
// operationName, isRelationshipOp and requirementIndex are typically the result of the DecodeOperation function that
// should generally call prior to call this function.
func GetImplementationArtifactForOperation(kv *api.KV, deploymentID, nodeName, operationName string, isNodeTemplateOperation, isRelationshipOp bool, requirementIndex string) (string, error) {
	var typeOrRelType string
	var err error
	if isNodeTemplateOperation {
		implType, err := GetOperationImplementationType(kv, deploymentID, nodeName, "", operationName)
		if err != nil {
			return "", err
		}
		return implType, nil
	} else if isRelationshipOp {
		typeOrRelType, err = GetRelationshipForRequirement(kv, deploymentID, nodeName, requirementIndex)
	} else {
		typeOrRelType, err = GetNodeType(kv, deploymentID, nodeName)
	}
	if err != nil {
		return "", err
	}

	implementedInType, err := GetTypeImplementingAnOperation(kv, deploymentID, typeOrRelType, operationName)
	if err != nil {
		return "", err
	}

	implType, err := GetOperationImplementationType(kv, deploymentID, "", implementedInType, operationName)
	if err != nil {
		return "", err
	}
	return implType, nil

}

// GetOperationInputs returns the list of inputs names for a given operation
func GetOperationInputs(kv *api.KV, deploymentID, nodeTemplateImpl, typeNameImpl, operationName string) ([]string, error) {
	operationPath, interfacePath, err := getOperationAndInterfacePath(kv, deploymentID, nodeTemplateImpl, typeNameImpl, operationName)
	if err != nil {
		return nil, err
	}
	// First Get operation inputs
	inputKeys, _, err := kv.Keys(operationPath+"/inputs/", "/", nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	inputs := make([]string, len(inputKeys))
	for i, input := range inputKeys {
		inputs[i] = path.Base(input)
	}

	// Then Get global interface inputs
	inputKeys, _, err = kv.Keys(interfacePath+"/inputs/", "/", nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, input := range inputKeys {
		inputName := path.Base(input)
		if !collections.ContainsString(inputs, inputName) {
			inputs = append(inputs, inputName)
		}
	}
	return inputs, nil
}

func getParentOperation(kv *api.KV, deploymentID string, operation prov.Operation) (prov.Operation, error) {
	parentType, err := GetParentType(kv, deploymentID, operation.ImplementedInType)
	if err != nil {
		return prov.Operation{}, err
	}
	if parentType != "" {
		opImplType, err := GetTypeImplementingAnOperation(kv, deploymentID, parentType, operation.Name)
		if err != nil {
			return prov.Operation{}, err
		}
		return prov.Operation{
			Name: operation.Name, ImplementationArtifact: operation.ImplementationArtifact,
			ImplementedInType: opImplType,
			RelOp: prov.RelationshipOperation{
				IsRelationshipOperation: operation.RelOp.IsRelationshipOperation,
				RequirementIndex:        operation.RelOp.RequirementIndex,
				TargetNodeName:          operation.RelOp.TargetNodeName,
			},
		}, nil
	}
	return prov.Operation{}, operationNotImplemented{msg: fmt.Sprintf("operation %q not found in the type hierarchy of type %q", operation.Name, operation.ImplementedInType)}

}

// An OperationInputResult represents a result of retrieving an operation input
//
// As in case of attributes it may have different values based on the instance name this struct contains the necessary information to identify the result context
type OperationInputResult struct {
	NodeName     string
	InstanceName string
	Value        string
	IsSecret     bool
}

// GetOperationInput retrieves the value of an input for a given operation
func GetOperationInput(kv *api.KV, deploymentID, nodeName string, operation prov.Operation, inputName string) ([]OperationInputResult, error) {
	isPropDef, err := IsOperationInputAPropertyDefinition(kv, deploymentID, operation.ImplementedInNodeTemplate, operation.ImplementedInType, operation.Name, inputName)
	if err != nil {
		return nil, err
	} else if isPropDef {
		return nil, errors.Errorf("Input %q for operation %v is a property definition we can't resolve it without a task input", inputName, operation)
	}

	operationPath, interfacePath, err := getOperationAndInterfacePath(kv, deploymentID, operation.ImplementedInNodeTemplate, operation.ImplementedInType, operation.Name)
	if err != nil {
		return nil, err
	}
	inputPath := path.Join(operationPath, "inputs", inputName, "data")
	res, isFunction, err := getValueAssignmentWithoutResolve(kv, deploymentID, inputPath, "")
	if err != nil {
		return nil, err
	}
	if res == nil {
		// Check global interface input
		inputPath = path.Join(interfacePath, "inputs", inputName, "data")
		res, isFunction, err = getValueAssignmentWithoutResolve(kv, deploymentID, inputPath, "")
		if err != nil {
			return nil, err
		}
	}
	results := make([]OperationInputResult, 0)
	if res != nil {
		if !isFunction {
			var ctxNodeName string
			if operation.RelOp.IsRelationshipOperation && operation.OperationHost == "TARGET" {
				ctxNodeName = operation.RelOp.TargetNodeName
			} else {
				ctxNodeName = nodeName
			}
			instances, err := GetNodeInstancesIds(kv, deploymentID, ctxNodeName)
			if err != nil {
				return nil, err
			}
			for _, ins := range instances {
				results = append(results, OperationInputResult{ctxNodeName, ins, res.RawString(), res.IsSecret})
			}
			return results, nil
		}
		va := &tosca.ValueAssignment{}
		err = yaml.Unmarshal([]byte(res.RawString()), va)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal TOSCA Function definition %q", res)
		}
		f := va.GetFunction()
		var hasAttrOnTarget bool
		var hasAttrOnSrcOrSelf bool
		for _, ga := range f.GetFunctionsByOperator(tosca.GetAttributeOperator) {
			switch ga.Operands[0].String() {
			case funcKeywordTARGET, funcKeywordRTARGET:
				hasAttrOnTarget = true
			case funcKeywordSELF, funcKeywordSOURCE, funcKeywordHOST:
				hasAttrOnSrcOrSelf = true
			}
		}

		var hasSecret bool
		if res.IsSecret || len(f.GetFunctionsByOperator(tosca.GetSecretOperator)) > 0 {
			hasSecret = true
		}

		if hasAttrOnSrcOrSelf && hasAttrOnTarget {
			return nil, errors.Errorf("can't resolve input %q for operation %v on node %q: get_attribute functions on TARGET and SELF/SOURCE/HOST at the same time is not supported.", inputName, operation, nodeName)
		}
		var instances []string
		var ctxNodeName string
		if hasAttrOnTarget && operation.RelOp.IsRelationshipOperation {
			instances, err = GetNodeInstancesIds(kv, deploymentID, operation.RelOp.TargetNodeName)
			ctxNodeName = operation.RelOp.TargetNodeName
		} else {
			instances, err = GetNodeInstancesIds(kv, deploymentID, nodeName)
			ctxNodeName = nodeName
		}
		if err != nil {
			return nil, err
		}

		for _, ins := range instances {
			res, err = resolver(kv, deploymentID).context(withNodeName(nodeName), withInstanceName(ins), withRequirementIndex(operation.RelOp.RequirementIndex)).resolveFunction(f)
			if err != nil {
				return nil, err
			}
			if res != nil {
				results = append(results, OperationInputResult{ctxNodeName, ins, res.RawString(), hasSecret || res.IsSecret})
			} else {
				ctx := events.NewContext(context.Background(), events.LogOptionalFields{events.NodeID: nodeName, events.InstanceID: ins})
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).Registerf("[WARNING] The function %q hasn't be resolved for deployment: %q, node: %q, instance: %q. No operation input variable will be set", f, deploymentID, nodeName, ins)
			}
		}
		return results, nil

	}
	// Check if it is implemented elsewhere
	newOp, err := getParentOperation(kv, deploymentID, operation)
	if err != nil {
		if !IsOperationNotImplemented(err) {
			return nil, err
		}
		return nil, inputNotFound{inputName, operation.Name, operation.ImplementedInType}
	}

	results, err = GetOperationInput(kv, deploymentID, nodeName, newOp, inputName)
	if err != nil && IsInputNotFound(err) {
		return nil, errors.Wrapf(err, "input not found in type %q", operation.ImplementedInType)
	}
	return results, err
}

// GetOperationInputPropertyDefinitionDefault retrieves the default value of an input of type property definition for a given operation
func GetOperationInputPropertyDefinitionDefault(kv *api.KV, deploymentID, nodeName string, operation prov.Operation, inputName string) ([]OperationInputResult, error) {
	isPropDef, err := IsOperationInputAPropertyDefinition(kv, deploymentID, operation.ImplementedInNodeTemplate, operation.ImplementedInType, operation.Name, inputName)
	if err != nil {
		return nil, err
	} else if !isPropDef {
		return nil, errors.Errorf("Input %q for operation %v is not a property definition we can't resolve its default value", inputName, operation)
	}
	operationPath, interfacePath, err := getOperationAndInterfacePath(kv, deploymentID, operation.ImplementedInNodeTemplate, operation.ImplementedInType, operation.Name)
	if err != nil {
		return nil, err
	}
	inputPath := path.Join(operationPath, "inputs", inputName, "default")
	// TODO base datatype should be retrieved
	res, isFunction, err := getValueAssignmentWithoutResolve(kv, deploymentID, inputPath, "")
	if err != nil {
		return nil, err
	}

	if res == nil {
		// Check global interface input
		inputPath = path.Join(interfacePath, "inputs", inputName, "default")
		// TODO base datatype should be retrieved
		res, isFunction, err = getValueAssignmentWithoutResolve(kv, deploymentID, inputPath, "")
		if err != nil {
			return nil, err
		}
	}
	results := make([]OperationInputResult, 0)
	if res != nil {
		if isFunction {
			return nil, errors.Errorf("can't resolve input %q for operation %v on node %q: TOSCA function are not supported for property definition defaults.", inputName, operation, nodeName)
		}
		instances, err := GetNodeInstancesIds(kv, deploymentID, nodeName)
		if err != nil {
			return nil, err
		}
		for _, ins := range instances {
			results = append(results, OperationInputResult{nodeName, ins, res.RawString(), res.IsSecret})
		}
		return results, nil
	}
	// Check if it is implemented elsewhere
	newOp, err := getParentOperation(kv, deploymentID, operation)
	if err != nil {
		if !IsOperationNotImplemented(err) {
			return nil, err
		}
		return nil, inputNotFound{inputName, operation.Name, operation.ImplementedInType}
	}

	results, err = GetOperationInputPropertyDefinitionDefault(kv, deploymentID, nodeName, newOp, inputName)
	if err != nil && IsInputNotFound(err) {
		return nil, errors.Wrapf(err, "input not found in type %q", operation.ImplementedInType)
	}
	return results, err
}

// IsOperationInputAPropertyDefinition checks if a given operation input is a property definition
func IsOperationInputAPropertyDefinition(kv *api.KV, deploymentID, nodeTemplateImpl, typeNameImpl, operationName, inputName string) (bool, error) {
	operationPath, interfacePath, err := getOperationAndInterfacePath(kv, deploymentID, nodeTemplateImpl, typeNameImpl, operationName)
	if err != nil {
		return false, err
	}
	kvp, _, err := kv.Get(path.Join(operationPath, "inputs", inputName, "is_property_definition"), nil)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	var nodeOrTypeImpl string
	if nodeTemplateImpl != "" {
		nodeOrTypeImpl = nodeTemplateImpl
	} else {
		nodeOrTypeImpl = typeNameImpl
	}

	if kvp == nil || len(kvp.Value) == 0 {
		kvp, _, err = kv.Get(path.Join(interfacePath, "inputs", inputName, "is_property_definition"), nil)
		if err != nil {
			return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp == nil || len(kvp.Value) == 0 {
			return false, errors.Errorf("Operation input %q not found for operation %q in node template/type %q", inputName, operationName, nodeOrTypeImpl)
		}
	}

	isPropDef, err := strconv.ParseBool(string(kvp.Value))
	return isPropDef, errors.Wrapf(err, "Failed to parse boolean for operation %q of node template/type %q", operationName, nodeOrTypeImpl)
}

// GetOperationHostFromTypeOperation return the operation_host declared for this operation if any.
//
// The returned value may be an empty string. This function doesn't explore the type heirarchy to
// find the operation or declared value for operation_host.
func GetOperationHostFromTypeOperation(kv *api.KV, deploymentID, typeName, interfaceName, operationName string) (string, error) {
	return GetOperationHostFromTypeOperationByName(kv, deploymentID, typeName, interfaceName+"."+operationName)
}

// GetOperationHostFromTypeOperationByName return the operation_host declared for this operation if any.
//
// The given operation name should be in format <interface_name>.<operation_name>
// The returned value may be an empty string. This function doesn't explore the type heirarchy to
// find the operation or declared value for operation_host.
func GetOperationHostFromTypeOperationByName(kv *api.KV, deploymentID, typeName, operationName string) (string, error) {
	opPath, _, err := getOperationAndInterfacePath(kv, deploymentID, "", typeName, operationName)
	if err != nil {
		return "", err
	}
	hop := path.Join(opPath, "implementation/operation_host")
	kvp, _, err := kv.Get(hop, nil)
	if err != nil || kvp == nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return string(kvp.Value), nil
}

// IsOperationImplemented checks if a given operation is implemented either in the node template or in the node type hierarchy
//
// An implemented operation means that it has a non empty primary implementation or file for an implementation artifact
func IsOperationImplemented(kv *api.KV, deploymentID, nodeName, operationName string) (bool, error) {
	// First check on node template
	nodeTemplateOpPath, err := getOperationPath(kv, deploymentID, nodeName, "", strings.ToLower(operationName))
	if err != nil {
		return false, err
	}
	result, err := isOperationPathImplemented(kv, nodeTemplateOpPath)
	if err != nil {
		return false, err
	}
	if result {
		return true, nil
	}

	// Then check type hierarchy
	typeName, err := GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return false, err
	}
	for typeName != "" {
		typeNameOpPath, err := getOperationPath(kv, deploymentID, "", typeName, strings.ToLower(operationName))
		if err != nil {
			return false, err
		}
		result, err := isOperationPathImplemented(kv, typeNameOpPath)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil
		}
		typeName, err = GetParentType(kv, deploymentID, typeName)
		if err != nil {
			return false, err
		}
	}

	return false, nil
}

func isOperationPathImplemented(kv *api.KV, operationPath string) (bool, error) {
	kvp, _, err := kv.Get(path.Join(operationPath, "implementation", "primary"), nil)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil && len(kvp.Value) != 0 {
		return true, nil
	}

	kvp, _, err = kv.Get(path.Join(operationPath, "implementation", "file"), nil)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return kvp != nil && len(kvp.Value) != 0, nil
}
