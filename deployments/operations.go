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
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/helper/stringutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/tosca"
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

// GetOperationForNodeOrNodeType traverses the type hierarchy to find an operation matching the given operationName.
//
// First, it checks the node template if operation implementation is present
// Next it gets down to types hierarchy
// Once found it returns the path to the operation and the value of its primary implementation.
// If the operation is not found in the node template or in the type hierarchy then empty strings are returned.
func GetOperationForNodeOrNodeType(ctx context.Context, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName string) (*tosca.OperationDefinition, error) {
	var typeOrNodeTemplate string
	var err error
	if nodeTemplateImpl == "" {
		typeOrNodeTemplate = nodeTypeImpl
	} else {
		typeOrNodeTemplate = nodeTemplateImpl
	}

	operationDef, _, err := getOperationAndInterfaceDefinitions(ctx, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to retrieve primary implementation for operation %q on template/type %q", operationName, typeOrNodeTemplate)
	}
	if isOperationImplemented(operationDef) {
		return operationDef, nil
	}

	// Not found here check the type hierarchy
	parentType, err := GetParentType(ctx, deploymentID, nodeTypeImpl)
	if err != nil || parentType == "" {
		return nil, err
	}

	return getOperationForNodeType(ctx, deploymentID, parentType, operationName)
}

func getOperationForNodeType(ctx context.Context, deploymentID, nodeType, operationName string) (*tosca.OperationDefinition, error) {
	// First check if operation exists in current nodeType
	operationDef, _, err := getOperationAndInterfaceDefinitions(ctx, deploymentID, "", nodeType, operationName)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to retrieve primary implementation for operation %q on type %q", operationName, nodeType)
	}
	if isOperationImplemented(operationDef) {
		return operationDef, nil
	}

	// Not found here check the type hierarchy
	parentType, err := GetParentType(ctx, deploymentID, nodeType)
	if err != nil || parentType == "" {
		return nil, err
	}

	return getOperationForNodeType(ctx, deploymentID, parentType, operationName)
}

// This function return the operation and interface definitions for given deploymentID, operation name
// It returns nil if no operation definition is found
// It returns nil if no interface definition is found
// It handles the ways that implementation is a node template or a node type
func getOperationAndInterfaceDefinitions(ctx context.Context, deploymentID, nodeTemplate, nodeType, operationName string) (*tosca.OperationDefinition, *tosca.InterfaceDefinition, error) {
	interfaceName := stringutil.GetAllExceptLastElement(operationName, ".")
	operationNameShort := stringutil.GetLastElement(operationName, ".")

	var interfaceDef *tosca.InterfaceDefinition
	if nodeTemplate != "" {
		node, err := getNodeTemplateStruct(ctx, deploymentID, nodeTemplate)
		if err != nil {
			return nil, nil, err
		}
		interfaceDef = getInterface(interfaceName, node.Interfaces)
	} else if nodeType != "" {
		nodeT := new(tosca.NodeType)
		err := getTypeStruct(deploymentID, nodeType, nodeT)
		if err != nil {
			return nil, nil, err
		}
		//interfaceDef, exist = nodeT.Interfaces[interfaceName]
		interfaceDef = getInterface(interfaceName, nodeT.Interfaces)
	}

	if interfaceDef != nil {
		opeDef := interfaceDef.Operations[operationNameShort]
		return &opeDef, interfaceDef, nil
	}

	return nil, nil, nil
}

func getInterface(key string, m map[string]tosca.InterfaceDefinition) *tosca.InterfaceDefinition {

	for k, v := range m {
		if strings.ToLower(normalizeInterfaceName(k)) == strings.ToLower(normalizeInterfaceName(key)) {
			return &v
		}
	}

	return nil
}

func normalizeInterfaceName(name string) string {
	if strings.HasPrefix(name, tosca.StandardInterfaceName) {
		return strings.Replace(name, tosca.StandardInterfaceName, tosca.StandardInterfaceShortName, 1)
	} else if strings.HasPrefix(name, tosca.ConfigureInterfaceName) {
		return strings.Replace(name, tosca.ConfigureInterfaceName, tosca.ConfigureInterfaceShortName, 1)
	}
	return name
}

// GetRelationshipTypeImplementingAnOperation  returns the first (bottom-up) type in the type hierarchy of a given relationship that implements a given operation
//
// An error is returned if the operation is not found in the type hierarchy
func GetRelationshipTypeImplementingAnOperation(ctx context.Context, deploymentID, nodeName, operationName, requirementIndex string) (string, error) {
	relTypeInit, err := GetRelationshipForRequirement(ctx, deploymentID, nodeName, requirementIndex)
	if err != nil {
		return "", err
	}
	relType := relTypeInit
	for relType != "" {
		operationDef, _, err := getOperationAndInterfaceDefinitions(ctx, deploymentID, "", relType, operationName)
		if err != nil {
			return "", err
		}
		if isOperationImplemented(operationDef) {
			return relType, nil
		}
		relType, err = GetParentType(ctx, deploymentID, relType)
	}
	return "", operationNotImplemented{msg: fmt.Sprintf("Operation %q not found in the type hierarchy of relationship %q", operationName, relTypeInit)}
}

// GetNodeTypeImplementingAnOperation returns the first (bottom-up) type in the type hierarchy of a given node that implements a given operation
//
// This is a shortcut for retrieving the node type and calling the GetTypeImplementingAnOperation() function
func GetNodeTypeImplementingAnOperation(ctx context.Context, deploymentID, nodeName, operationName string) (string, error) {
	nodeType, err := GetNodeType(ctx, deploymentID, nodeName)
	log.Debugf("[GetNodeTypeImplementingAnOperation] nodeType=%q", nodeType)
	if err != nil {
		return "", err
	}
	t, err := GetTypeImplementingAnOperation(ctx, deploymentID, nodeType, operationName)
	return t, errors.Wrapf(err, "operation not found for node %q", nodeName)
}

// IsNodeTemplateImplementingOperation returns true if the node implements the defined operation
func IsNodeTemplateImplementingOperation(ctx context.Context, deploymentID, nodeName, operationName string) (bool, error) {
	operationDef, _, err := getOperationAndInterfaceDefinitions(ctx, deploymentID, nodeName, "", operationName)
	if err != nil {
		return false, errors.Wrapf(err, "Can't define if operation with name:%q exists for node %q", operationName, nodeName)
	}
	return isOperationImplemented(operationDef), nil
}

// GetTypeImplementingAnOperation returns the first (bottom-up) type in the type hierarchy that implements a given operation
//
// An error is returned if the operation is not found in the type hierarchy
func GetTypeImplementingAnOperation(ctx context.Context, deploymentID, typeName, operationName string) (string, error) {
	log.Debugf("[GetTypeImplementingAnOperation] operationName=%q", operationName)
	implType := typeName
	for implType != "" {
		log.Debugf("[GetTypeImplementingAnOperation] implType=%q", implType)
		operationDef, _, err := getOperationAndInterfaceDefinitions(ctx, deploymentID, "", implType, operationName)
		if err != nil {
			return "", err
		}
		log.Debugf("[GetTypeImplementingAnOperation] operationDef=%+v", operationDef)
		if isOperationImplemented(operationDef) {
			return implType, nil
		}
		implType, err = GetParentType(ctx, deploymentID, implType)
	}
	return "", operationNotImplemented{msg: fmt.Sprintf("operation %q not found in the type hierarchy of type %q", operationName, typeName)}
}

// GetOperationImplementationType allows you when the implementation of an operation is an artifact to retrieve the type of this artifact
func GetOperationImplementationType(ctx context.Context, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName string) (string, error) {
	var nodeOrTypeImpl string
	if nodeTemplateImpl != "" {
		nodeOrTypeImpl = nodeTemplateImpl
	} else {
		nodeOrTypeImpl = nodeTypeImpl
	}
	operationDef, _, err := getOperationAndInterfaceDefinitions(ctx, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName)
	if err != nil {
		return "", errors.Wrapf(err, "Fail to get the type of operation implementation for deploymentID:%q, operation name:%q, node/template:%q", deploymentID, operationName, nodeOrTypeImpl)
	}
	if !isOperationImplemented(operationDef) {
		return "", errors.Errorf("Failed to resolve implementation for operation %q in node template/type %q", operationName, nodeOrTypeImpl)
	}

	if &operationDef.Implementation != nil && &operationDef.Implementation.Artifact != nil && operationDef.Implementation.Artifact.Type != "" {
		return operationDef.Implementation.Artifact.Type, nil
	}

	if &operationDef.Implementation == nil || operationDef.Implementation.Primary == "" {
		return "", errors.Errorf("Failed to resolve implementation for operation %q in node template/type %q", operationName, nodeOrTypeImpl)
	}

	primarySlice := strings.Split(operationDef.Implementation.Primary, ".")
	ext := primarySlice[len(primarySlice)-1]
	artImpl, err := GetImplementationArtifactForExtension(ctx, deploymentID, ext)
	if err != nil {
		return "", err
	}
	if artImpl == "" {
		return "", errors.Errorf("Failed to resolve implementation artifact for node template/type %q, operation %q, implementation %q and extension %q", nodeOrTypeImpl, operationName, operationDef.Implementation.Primary, ext)
	}
	return artImpl, nil

}

func getOperationImplementation(ctx context.Context, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName, implementationType string) (string, error) {
	var res string
	operationDef, _, err := getOperationAndInterfaceDefinitions(ctx, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName)
	if err != nil {
		return "", errors.Wrap(err, "Fail to get operation implementation")
	}

	if !isOperationImplemented(operationDef) {
		return "", nil
	}

	if &operationDef.Implementation == nil || &operationDef.Implementation.Artifact == nil {
		return "", errors.Errorf("Operation implementation %q not found for %q", implementationType, operationName)
	}

	if implementationType == "file" {
		if operationDef.Implementation.Artifact.File == "" {
			return "", errors.Errorf("Operation implementation file not found for %q", operationName)
		}
		res = operationDef.Implementation.Artifact.File
	} else if implementationType == "repository" {
		if operationDef.Implementation.Artifact.Repository == "" {
			return "", errors.Errorf("Operation implementation repository not found for %q", operationName)
		}
		res = operationDef.Implementation.Artifact.Repository
	}
	return res, nil
}

// GetOperationImplementationFile allows you when the implementation of an operation is an artifact to retrieve the file of this artifact
//
// The returned file is the raw value. To have a file with a path relative to the root of the deployment use GetOperationImplementationFileWithRelativePath()
func GetOperationImplementationFile(ctx context.Context, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName string) (string, error) {
	return getOperationImplementation(ctx, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName, "file")
}

// GetOperationImplementationRepository allows you when the implementation of an operation is an artifact to retrieve the repository of this artifact
func GetOperationImplementationRepository(ctx context.Context, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName string) (string, error) {
	return getOperationImplementation(ctx, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName, "repository")
}

// GetOperationImplementationFileWithRelativePath allows you when the implementation of an operation
// is an artifact to retrieve the file of this artifact
//
// The returned file is relative to the root of the deployment. To have the raw value use GetOperationImplementationFile()
func GetOperationImplementationFileWithRelativePath(ctx context.Context, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName string) (string, error) {
	file, err := GetOperationImplementationFile(ctx, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName)
	if err != nil {
		return file, err
	}
	if nodeTemplateImpl != "" {
		return file, err
	}
	// If implementation is a node type, import path must be resolved
	importPath, err := GetTypeImportPath(ctx, deploymentID, nodeTypeImpl)
	return path.Join(importPath, file), err
}

// GetOperationOutputForNode return a map with in index the instance number and in value the result of the output
// The "params" parameter is necessary to pass the path of the output
func GetOperationOutputForNode(ctx context.Context, deploymentID, nodeName, instanceName, interfaceName, operationName, outputName string) (string, error) {
	instancesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName)

	exist, output, err := consulutil.GetStringValue(filepath.Join(instancesPath, instanceName, "outputs", strings.ToLower(interfaceName), strings.ToLower(operationName), outputName))
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if exist && output != "" {
		return output, nil
	}
	// Look at host node
	var host string
	host, err = GetHostedOnNode(ctx, deploymentID, nodeName)
	if err != nil {
		return "", err
	}
	if host != "" {
		// TODO we consider that instance name is the same for the host but we should not
		return GetOperationOutputForNode(ctx, deploymentID, host, instanceName, interfaceName, operationName, outputName)
	}
	return "", nil
}

// GetOperationOutputForRelationship retrieves an operation output for a relationship
// The returned value may be empty if the operation output could not be retrieved
func GetOperationOutputForRelationship(ctx context.Context, deploymentID, nodeName, instanceName, requirementIndex, interfaceName, operationName, outputName string) (string, error) {
	exist, result, err := consulutil.GetStringValue(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances", nodeName, requirementIndex, instanceName, "outputs", strings.ToLower(path.Join(interfaceName, operationName)), outputName))
	if err != nil {
		return "", err
	}

	if !exist || result == "" {
		return "", nil
	}
	return result, nil
}

func getOperationOutputForRequirements(ctx context.Context, deploymentID, nodeName, instanceName, interfaceName, operationName, outputName string) (string, error) {
	reqIndexes, err := GetRequirementsIndexes(ctx, deploymentID, nodeName)
	if err != nil {
		return "", err
	}
	for _, reqIndex := range reqIndexes {
		result, err := GetOperationOutputForRelationship(ctx, deploymentID, nodeName, instanceName, reqIndex, interfaceName, operationName, outputName)
		if err != nil || result != "" {
			return result, err
		}
	}
	return "", nil
}

// GetImplementationArtifactForExtension returns the implementation artifact type for a given extension.
//
// If the extension is unknown then an empty string is returned
func GetImplementationArtifactForExtension(ctx context.Context, deploymentID, extension string) (string, error) {
	extension = strings.ToLower(extension)

	extensionsPtr := new(map[string]string)
	exist, err := storage.GetStore(types.StoreTypeDeployment).Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", implementationArtifactsExtensionsPath), extensionsPtr)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist {
		return "", nil
	}

	extensions := *extensionsPtr
	return extensions[extension], nil
}

// GetImplementationArtifactForOperation returns the implementation artifact type for a given operation.
// operationName, isRelationshipOp and requirementIndex are typically the result of the DecodeOperation function that
// should generally call prior to call this function.
func GetImplementationArtifactForOperation(ctx context.Context, deploymentID, nodeName, operationName string, isNodeTemplateOperation, isRelationshipOp bool, requirementIndex string) (string, error) {
	var typeOrRelType string
	var err error
	if isNodeTemplateOperation {
		implType, err := GetOperationImplementationType(ctx, deploymentID, nodeName, "", operationName)
		if err != nil {
			return "", err
		}
		return implType, nil
	} else if isRelationshipOp {
		typeOrRelType, err = GetRelationshipForRequirement(ctx, deploymentID, nodeName, requirementIndex)
	} else {
		typeOrRelType, err = GetNodeType(ctx, deploymentID, nodeName)
	}
	if err != nil {
		return "", err
	}

	implementedInType, err := GetTypeImplementingAnOperation(ctx, deploymentID, typeOrRelType, operationName)
	if err != nil {
		return "", err
	}

	implType, err := GetOperationImplementationType(ctx, deploymentID, "", implementedInType, operationName)
	if err != nil {
		return "", err
	}
	return implType, nil

}

// GetOperationInputs returns the list of inputs names for a given operation
func GetOperationInputs(ctx context.Context, deploymentID, nodeTemplateImpl, typeNameImpl, operationName string) ([]string, error) {
	inputs := make([]string, 0)
	operationDef, interfaceDef, err := getOperationAndInterfaceDefinitions(ctx, deploymentID, nodeTemplateImpl, typeNameImpl, operationName)
	if err != nil {
		return nil, err
	}
	// First Get operation inputs
	if operationDef != nil {
		for k := range operationDef.Inputs {
			inputs = append(inputs, k)
		}
	}

	if interfaceDef != nil {
		// Then Get global interface inputs
		for k := range interfaceDef.Inputs {
			if !collections.ContainsString(inputs, k) {
				inputs = append(inputs, k)
			}
		}
	}

	return inputs, nil
}

func getParentOperation(ctx context.Context, deploymentID string, operation prov.Operation) (prov.Operation, error) {
	parentType, err := GetParentType(ctx, deploymentID, operation.ImplementedInType)
	if err != nil {
		return prov.Operation{}, err
	}
	if parentType != "" {
		opImplType, err := GetTypeImplementingAnOperation(ctx, deploymentID, parentType, operation.Name)
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
func GetOperationInput(ctx context.Context, deploymentID, nodeName string, operation prov.Operation, inputName string) ([]OperationInputResult, error) {
	operationDef, interfaceDef, err := getOperationAndInterfaceDefinitions(ctx, deploymentID, operation.ImplementedInNodeTemplate, operation.ImplementedInType, operation.Name)
	if err != nil {
		return nil, err
	}

	// Check operation input
	var va *tosca.ValueAssignment
	if operationDef != nil {
		input, is := operationDef.Inputs[inputName]
		if is && &input != nil {
			va = input.ValueAssign
		}
	}

	// Check global interface input
	if va == nil && interfaceDef != nil {
		input, is := interfaceDef.Inputs[inputName]
		if is && &input != nil {
			va = input.ValueAssign
		}
	}

	if va == nil {
		return nil, errors.Errorf("Input %q for operation %v is a property definition we can't resolve it without a task input", inputName, operation)
	}

	res, isFunction, err := getValueAssignmentWithoutResolve(ctx, va, nil, "")
	if err != nil {
		return nil, err
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
			instances, err := GetNodeInstancesIds(ctx, deploymentID, ctxNodeName)
			if err != nil {
				return nil, err
			}
			for _, ins := range instances {
				results = append(results, OperationInputResult{ctxNodeName, ins, res.RawString(), res.IsSecret})
			}
			return results, nil
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
			instances, err = GetNodeInstancesIds(ctx, deploymentID, operation.RelOp.TargetNodeName)
			ctxNodeName = operation.RelOp.TargetNodeName
		} else {
			instances, err = GetNodeInstancesIds(ctx, deploymentID, nodeName)
			ctxNodeName = nodeName
		}
		if err != nil {
			return nil, err
		}

		for _, ins := range instances {
			res, err = resolver(deploymentID).context(withNodeName(nodeName), withInstanceName(ins), withRequirementIndex(operation.RelOp.RequirementIndex)).resolveFunction(ctx, f)
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
	newOp, err := getParentOperation(ctx, deploymentID, operation)
	if err != nil {
		if !IsOperationNotImplemented(err) {
			return nil, err
		}
		return nil, inputNotFound{inputName, operation.Name, operation.ImplementedInType}
	}

	results, err = GetOperationInput(ctx, deploymentID, nodeName, newOp, inputName)
	if err != nil && IsInputNotFound(err) {
		return nil, errors.Wrapf(err, "input not found in type %q", operation.ImplementedInType)
	}
	return results, err
}

// GetOperationInputPropertyDefinitionDefault retrieves the default value of an input of type property definition for a given operation
func GetOperationInputPropertyDefinitionDefault(ctx context.Context, deploymentID, nodeName string, operation prov.Operation, inputName string) ([]OperationInputResult, error) {
	operationDef, interfaceDef, err := getOperationAndInterfaceDefinitions(ctx, deploymentID, operation.ImplementedInNodeTemplate, operation.ImplementedInType, operation.Name)
	if err != nil {
		return nil, err
	}

	// Check operation input
	var va *tosca.ValueAssignment
	if operationDef != nil {
		input, is := operationDef.Inputs[inputName]
		if is && &input != nil {
			va = input.PropDef.Default
		}
	}

	// Check global interface input
	if va == nil && interfaceDef != nil {
		input, is := interfaceDef.Inputs[inputName]
		if is && &input != nil {
			va = input.PropDef.Default
		}
	}

	if va == nil {
		return nil, errors.Errorf("Input %q for operation %v is not a property definition we can't resolve its default value", inputName, operation)
	}

	res, isFunction, err := getValueAssignmentWithoutResolve(ctx, va, nil, "")
	if err != nil {
		return nil, err
	}

	results := make([]OperationInputResult, 0)
	if res != nil {
		if isFunction {
			return nil, errors.Errorf("can't resolve input %q for operation %v on node %q: TOSCA function are not supported for property definition defaults.", inputName, operation, nodeName)
		}
		instances, err := GetNodeInstancesIds(ctx, deploymentID, nodeName)
		if err != nil {
			return nil, err
		}
		for _, ins := range instances {
			results = append(results, OperationInputResult{nodeName, ins, res.RawString(), res.IsSecret})
		}
		return results, nil
	}
	// Check if it is implemented elsewhere
	newOp, err := getParentOperation(ctx, deploymentID, operation)
	if err != nil {
		if !IsOperationNotImplemented(err) {
			return nil, err
		}
		return nil, inputNotFound{inputName, operation.Name, operation.ImplementedInType}
	}

	results, err = GetOperationInputPropertyDefinitionDefault(ctx, deploymentID, nodeName, newOp, inputName)
	if err != nil && IsInputNotFound(err) {
		return nil, errors.Wrapf(err, "input not found in type %q", operation.ImplementedInType)
	}
	return results, err
}

// IsOperationInputAPropertyDefinition checks if a given operation input is a property definition
func IsOperationInputAPropertyDefinition(ctx context.Context, deploymentID, nodeTemplateImpl, typeNameImpl, operationName, inputName string) (bool, error) {
	var typeOrNodeTemplate string
	if nodeTemplateImpl == "" {
		typeOrNodeTemplate = typeNameImpl
	} else {
		typeOrNodeTemplate = nodeTemplateImpl
	}
	operationDef, interfaceDef, err := getOperationAndInterfaceDefinitions(ctx, deploymentID, nodeTemplateImpl, typeNameImpl, operationName)
	if err != nil {
		return false, err
	}
	if operationDef != nil {
		input, is := operationDef.Inputs[inputName]
		if is && &input != nil {
			return input.PropDef != nil, nil
		}
	}

	if interfaceDef != nil {
		input, is := interfaceDef.Inputs[inputName]
		if is && &input != nil {
			return input.PropDef != nil, nil
		}
	}
	return false, errors.Errorf("failed to find input with name:%q for operation:%q and node template/type:%q", inputName, operationName, typeOrNodeTemplate)
}

// GetOperationHostFromTypeOperation return the operation_host declared for this operation if any.
//
// The returned value may be an empty string. This function doesn't explore the type heirarchy to
// find the operation or declared value for operation_host.
func GetOperationHostFromTypeOperation(ctx context.Context, deploymentID, typeName, interfaceName, operationName string) (string, error) {
	return GetOperationHostFromTypeOperationByName(ctx, deploymentID, typeName, interfaceName+"."+operationName)
}

// GetOperationHostFromTypeOperationByName return the operation_host declared for this operation if any.
//
// The given operation name should be in format <interface_name>.<operation_name>
// The returned value may be an empty string. This function doesn't explore the type hierarchy to
// find the operation or declared value for operation_host.
func GetOperationHostFromTypeOperationByName(ctx context.Context, deploymentID, typeName, operationName string) (string, error) {
	operationDef, _, err := getOperationAndInterfaceDefinitions(ctx, deploymentID, "", typeName, operationName)
	if err != nil {
		return "", err
	}
	if !isOperationImplemented(operationDef) {
		return "", nil
	}
	return operationDef.Implementation.OperationHost, nil
}

// IsOperationImplemented checks if a given operation is implemented either in the node template or in the node type hierarchy
//
// An implemented operation means that it has a non empty primary implementation or file for an implementation artifact
func IsOperationImplemented(ctx context.Context, deploymentID, nodeName, operationName string) (bool, error) {
	// First check on node template
	operationDef, _, err := getOperationAndInterfaceDefinitions(ctx, deploymentID, nodeName, "", strings.ToLower(operationName))
	if err != nil {
		return false, err
	}
	if isOperationImplemented(operationDef) {
		return true, nil
	}

	// Then check type hierarchy
	typeName, err := GetNodeType(ctx, deploymentID, nodeName)
	if err != nil {
		return false, err
	}
	for typeName != "" {
		operationDef, _, err := getOperationAndInterfaceDefinitions(ctx, deploymentID, "", typeName, strings.ToLower(operationName))
		if err != nil {
			return false, err
		}
		if isOperationImplemented(operationDef) {
			return true, nil
		}
		typeName, err = GetParentType(ctx, deploymentID, typeName)
		if err != nil {
			return false, err
		}
	}

	return false, nil
}

func isOperationImplemented(operationDefinition *tosca.OperationDefinition) bool {
	if operationDefinition != nil && &operationDefinition.Implementation != nil && (operationDefinition.Implementation.Primary != "" || (&operationDefinition.Implementation.Artifact != nil && operationDefinition.Implementation.Artifact.File != "")) {
		return true
	}
	return false
}
