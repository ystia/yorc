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
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/collections"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/registry"
	"github.com/ystia/yorc/tosca"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"
)

// Internal type used to uniquely identify the errorgroup in a context
type ctxErrGrpKey struct{}

// Internal variable used to uniquely identify the errorgroup in a context
var errGrpKey ctxErrGrpKey

// Internal type used to uniquely identify the ConsulStore in a context
type ctxConsulStoreKey struct{}

// Internal variable used to uniquely identify the ConsulStore in a context
var consulStoreKey ctxConsulStoreKey

var reg = registry.GetRegistry()

// StoreDeploymentDefinition takes a defPath and parse it as a tosca.Topology then it store it in consul under
// consulutil.DeploymentKVPrefix/deploymentID
func StoreDeploymentDefinition(ctx context.Context, kv *api.KV, deploymentID string, defPath string) error {
	topology := tosca.Topology{}
	definition, err := os.Open(defPath)
	if err != nil {
		return handleDeploymentStatus(deploymentID, errors.Wrapf(err, "Failed to open definition file %q", defPath))
	}
	defBytes, err := ioutil.ReadAll(definition)
	if err != nil {
		return handleDeploymentStatus(deploymentID, errors.Wrapf(err, "Failed to open definition file %q", defPath))
	}

	err = yaml.Unmarshal(defBytes, &topology)
	if err != nil {
		return handleDeploymentStatus(deploymentID, errors.Wrapf(err, "Failed to unmarshal yaml definition for file %q", defPath))
	}

	err = storeDeployment(ctx, topology, deploymentID, filepath.Dir(defPath))
	if err != nil {
		return handleDeploymentStatus(deploymentID, errors.Wrapf(err, "Failed to store TOSCA Definition for deployment with id %q, (file path %q)", deploymentID, defPath))
	}
	err = registerImplementationTypes(ctx, kv, deploymentID)
	if err != nil {
		return handleDeploymentStatus(deploymentID, err)
	}

	return handleDeploymentStatus(deploymentID, enhanceNodes(ctx, kv, deploymentID))
}

func handleDeploymentStatus(deploymentID string, err error) error {
	if err != nil {
		consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "status"), fmt.Sprint(DEPLOYMENT_FAILED))
	}
	return err
}

// storeDeployment stores a whole deployment.
func storeDeployment(ctx context.Context, topology tosca.Topology, deploymentID, rootDefPath string) error {
	errCtx, errGroup, consulStore := consulutil.WithContext(ctx)
	errCtx = context.WithValue(errCtx, errGrpKey, errGroup)
	errCtx = context.WithValue(errCtx, consulStoreKey, consulStore)
	consulStore.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "status"), fmt.Sprint(INITIAL))

	errGroup.Go(func() error {
		return storeTopology(errCtx, topology, deploymentID, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology"), "", "", rootDefPath)
	})

	return errGroup.Wait()
}

// storeTopology stores a given topology.
//
// The given topology may be an import in this case importPrefix and importPath should be specified
func storeTopology(ctx context.Context, topology tosca.Topology, deploymentID, topologyPrefix, importPrefix, importPath, rootDefPath string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	log.Debugf("Storing topology with name %q (Import prefix %q)", topology.Metadata[tosca.TemplateName], importPrefix)
	storeTopologyTopLevelKeyNames(ctx, topology, path.Join(topologyPrefix, importPrefix))
	if err := storeImports(ctx, topology, deploymentID, topologyPrefix, importPath, rootDefPath); err != nil {
		return err
	}
	storeRepositories(ctx, topology, topologyPrefix)
	storeDataTypes(ctx, topology, topologyPrefix, importPath)

	// There is no need to parse a topology template if this topology template
	// is declared in an import.
	// Parsing only the topology template declared in the root topology file
	isRootTopologyTemplate := (importPrefix == "")
	if isRootTopologyTemplate {
		storeInputs(ctx, topology, topologyPrefix)
		storeOutputs(ctx, topology, topologyPrefix)
		storeSubstitutionMappings(ctx, topology, topologyPrefix)
		storeNodes(ctx, topology, topologyPrefix, importPath, rootDefPath)
	} else {
		// For imported templates, storing substitution mappings if any
		// as they contain details on service to application/node type mapping
		storeSubstitutionMappings(ctx, topology,
			path.Join(topologyPrefix, importPrefix))
	}

	if err := storeNodeTypes(ctx, topology, topologyPrefix, importPath); err != nil {
		return err
	}
	if err := storeRelationshipTypes(ctx, topology, topologyPrefix, importPath); err != nil {
		return err
	}
	storeCapabilityTypes(ctx, topology, topologyPrefix, importPath)
	storeArtifactTypes(ctx, topology, topologyPrefix, importPath)

	// Detect potential cycles in inline workflows
	if err := checkNestedWorkflows(topology); err != nil {
		return err
	}

	if isRootTopologyTemplate {
		storeWorkflows(ctx, topology, deploymentID)
	}
	return nil
}

// storeTopologyTopLevelKeyNames stores top level keynames for a topology.
//
// This may be done under the import path in case of imports.
func storeTopologyTopLevelKeyNames(ctx context.Context, topology tosca.Topology, topologyPrefix string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	consulStore.StoreConsulKeyAsString(topologyPrefix+"/tosca_version", topology.TOSCAVersion)
	consulStore.StoreConsulKeyAsString(topologyPrefix+"/description", topology.Description)
	storeStringMap(consulStore, topologyPrefix+"/metadata", topology.Metadata)
}

//storeRepositories store repositories
func storeRepositories(ctx context.Context, topology tosca.Topology, topologyPrefix string) error {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	repositoriesPrefix := path.Join(topologyPrefix, "repositories")
	for repositoryName, repo := range topology.Repositories {
		repoPrefix := path.Join(repositoriesPrefix, repositoryName)
		consulStore.StoreConsulKeyAsString(path.Join(repoPrefix, "url"), repo.URL)
		consulStore.StoreConsulKeyAsString(path.Join(repoPrefix, "type"), repo.Type)
		consulStore.StoreConsulKeyAsString(path.Join(repoPrefix, "description"), repo.Description)
		consulStore.StoreConsulKeyAsString(path.Join(repoPrefix, "credentials", "user"), repo.Credit.User)
		consulStore.StoreConsulKeyAsString(path.Join(repoPrefix, "credentials", "token"), repo.Credit.Token)
		if repo.Credit.TokenType == "" {
			repo.Credit.TokenType = "password"
		}
		consulStore.StoreConsulKeyAsString(path.Join(repoPrefix, "credentials", "token_type"), repo.Credit.TokenType)
	}

	return nil
}

func storeCommonType(consulStore consulutil.ConsulStore, commonType tosca.Type, typePrefix, importPath string) {
	consulStore.StoreConsulKeyAsString(path.Join(typePrefix, "derived_from"), commonType.DerivedFrom)
	consulStore.StoreConsulKeyAsString(path.Join(typePrefix, "version"), commonType.Version)
	consulStore.StoreConsulKeyAsString(path.Join(typePrefix, "description"), commonType.Description)
	consulStore.StoreConsulKeyAsString(path.Join(typePrefix, "importPath"), importPath)
	for metaName, metaValue := range commonType.Metadata {
		consulStore.StoreConsulKeyAsString(path.Join(typePrefix, "metadata", metaName), metaValue)
	}
}

//storeDataTypes store data types
func storeDataTypes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath string) error {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	dataTypesPrefix := path.Join(topologyPrefix, "types")
	for dataTypeName, dataType := range topology.DataTypes {
		dtPrefix := path.Join(dataTypesPrefix, dataTypeName)
		storeCommonType(consulStore, dataType.Type, dtPrefix, importPath)
		consulStore.StoreConsulKeyAsString(path.Join(dtPrefix, "name"), dataTypeName)
		for propName, propDefinition := range dataType.Properties {
			storePropertyDefinition(ctx, path.Join(dtPrefix, "properties", propName), propName, propDefinition)
		}
	}

	return nil
}

// storeImports parses and store imports.
func storeImports(ctx context.Context, topology tosca.Topology, deploymentID, topologyPrefix, importPath, rootDefPath string) error {
	errGroup := ctx.Value(errGrpKey).(*errgroup.Group)
	for _, element := range topology.Imports {

		importURI := strings.Trim(element.File, " \t")
		importedTopology := tosca.Topology{}

		if strings.HasPrefix(importURI, "<") && strings.HasSuffix(importURI, ">") {
			// Internal import
			importName := strings.Trim(importURI, "<>")
			var defBytes []byte
			var err error
			if defBytes, err = reg.GetToscaDefinition(importName); err != nil {
				return errors.Errorf("Failed to import internal definition %s: %v", importURI, err)
			}
			if err = yaml.Unmarshal(defBytes, &importedTopology); err != nil {
				return errors.Errorf("Failed to parse internal definition %s: %v", importURI, err)
			}
			errGroup.Go(func() error {
				return storeTopology(ctx, importedTopology, deploymentID, topologyPrefix, path.Join("imports", importURI), "", rootDefPath)
			})
		} else {
			uploadFile := filepath.Join(rootDefPath, filepath.FromSlash(importPath), filepath.FromSlash(importURI))

			definition, err := os.Open(uploadFile)
			if err != nil {
				return errors.Errorf("Failed to parse internal definition %s: %v", importURI, err)
			}

			defBytes, err := ioutil.ReadAll(definition)
			if err != nil {
				return errors.Errorf("Failed to parse internal definition %s: %v", importURI, err)
			}

			if err = yaml.Unmarshal(defBytes, &importedTopology); err != nil {
				return errors.Errorf("Failed to parse internal definition %s: %v", importURI, err)
			}

			errGroup.Go(func() error {
				// Using flat keys under imports, for convenience when searching
				// for an import containing a given metadata template name
				// (function getSubstitutableNodeType in this package)
				importPrefix := path.Join("imports", strings.Replace(path.Join(importPath, importURI), "/", "_", -1))
				return storeTopology(ctx, importedTopology, deploymentID, topologyPrefix, importPrefix, path.Dir(path.Join(importPath, importURI)), rootDefPath)
			})
		}
	}
	return nil
}

// storeOutputs stores topology outputs
func storeOutputs(ctx context.Context, topology tosca.Topology, topologyPrefix string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	outputsPrefix := path.Join(topologyPrefix, "outputs")
	for outputName, output := range topology.TopologyTemplate.Outputs {
		outputPrefix := path.Join(outputsPrefix, outputName)
		consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "name"), outputName)
		consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "description"), output.Description)
		storeValueAssignment(consulStore, path.Join(outputPrefix, "default"), output.Default)
		if output.Required == nil {
			// Required by default
			consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "required"), "true")
		} else {
			consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "required"), strconv.FormatBool(*output.Required))
		}
		consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "status"), output.Status)
		consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "type"), output.Type)
		consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "entry_schema"), output.EntrySchema.Type)
		storeValueAssignment(consulStore, path.Join(outputPrefix, "value"), output.Value)
	}
}

// storeInputs stores topology outputs
func storeInputs(ctx context.Context, topology tosca.Topology, topologyPrefix string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	inputsPrefix := path.Join(topologyPrefix, "inputs")
	for inputName, input := range topology.TopologyTemplate.Inputs {
		inputPrefix := path.Join(inputsPrefix, inputName)
		consulStore.StoreConsulKeyAsString(path.Join(inputPrefix, "name"), inputName)
		consulStore.StoreConsulKeyAsString(path.Join(inputPrefix, "description"), input.Description)
		storeValueAssignment(consulStore, path.Join(inputPrefix, "default"), input.Default)
		if input.Required == nil {
			// Required by default
			consulStore.StoreConsulKeyAsString(path.Join(inputPrefix, "required"), "true")
		} else {
			consulStore.StoreConsulKeyAsString(path.Join(inputPrefix, "required"), strconv.FormatBool(*input.Required))
		}
		consulStore.StoreConsulKeyAsString(path.Join(inputPrefix, "status"), input.Status)
		consulStore.StoreConsulKeyAsString(path.Join(inputPrefix, "type"), input.Type)
		consulStore.StoreConsulKeyAsString(path.Join(inputPrefix, "entry_schema"), input.EntrySchema.Type)
		storeValueAssignment(consulStore, path.Join(inputPrefix, "value"), input.Value)
	}
}

func storeRequirementAssignment(ctx context.Context, requirement tosca.RequirementAssignment, requirementPrefix, requirementName string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	consulStore.StoreConsulKeyAsString(requirementPrefix+"/name", requirementName)
	consulStore.StoreConsulKeyAsString(requirementPrefix+"/node", requirement.Node)
	consulStore.StoreConsulKeyAsString(requirementPrefix+"/relationship", requirement.Relationship)
	consulStore.StoreConsulKeyAsString(requirementPrefix+"/capability", requirement.Capability)
	consulStore.StoreConsulKeyAsString(requirementPrefix+"/type_requirement", requirement.TypeRequirement)
	for propName, propValue := range requirement.RelationshipProps {
		storeValueAssignment(consulStore, requirementPrefix+"/properties/"+url.QueryEscape(propName), propValue)
	}
}

// storeNodes stores topology nodes
func storeNodes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath, rootDefPath string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	nodesPrefix := path.Join(topologyPrefix, "nodes")
	for nodeName, node := range topology.TopologyTemplate.NodeTemplates {
		nodePrefix := nodesPrefix + "/" + nodeName
		consulStore.StoreConsulKeyAsString(nodePrefix+"/name", nodeName)
		consulStore.StoreConsulKeyAsString(nodePrefix+"/type", node.Type)
		if node.Directives != nil {
			consulStore.StoreConsulKeyAsString(
				path.Join(nodePrefix, "directives"),
				strings.Join(node.Directives, ","))
		}
		propertiesPrefix := nodePrefix + "/properties"
		for propName, propValue := range node.Properties {
			storeValueAssignment(consulStore, propertiesPrefix+"/"+url.QueryEscape(propName), propValue)
		}
		attributesPrefix := nodePrefix + "/attributes"
		for attrName, attrValue := range node.Attributes {
			storeValueAssignment(consulStore, attributesPrefix+"/"+url.QueryEscape(attrName), attrValue)
		}
		capabilitiesPrefix := nodePrefix + "/capabilities"
		for capName, capability := range node.Capabilities {
			capabilityPrefix := capabilitiesPrefix + "/" + capName
			capabilityPropsPrefix := capabilityPrefix + "/properties"
			for propName, propValue := range capability.Properties {
				storeValueAssignment(consulStore, capabilityPropsPrefix+"/"+url.QueryEscape(propName), propValue)
			}
			capabilityAttrPrefix := capabilityPrefix + "/attributes"
			for attrName, attrValue := range capability.Attributes {
				storeValueAssignment(consulStore, capabilityAttrPrefix+"/"+url.QueryEscape(attrName), attrValue)
			}
		}
		requirementsPrefix := nodePrefix + "/requirements"
		for reqIndex, reqValueMap := range node.Requirements {
			for reqName, reqValue := range reqValueMap {
				reqPrefix := requirementsPrefix + "/" + strconv.Itoa(reqIndex)
				storeRequirementAssignment(ctx, reqValue, reqPrefix, reqName)
			}
		}
		artifactsPrefix := nodePrefix + "/artifacts"
		for artName, artDef := range node.Artifacts {
			artFile := filepath.Join(rootDefPath, filepath.FromSlash(path.Join(importPath, artDef.File)))
			log.Debugf("Looking if artifact %q exists on filesystem", artFile)
			if _, err := os.Stat(artFile); os.IsNotExist(err) {
				log.Printf("Warning: Artifact %q for node %q with computed path %q doesn't exists on filesystem, ignoring it.", artName, nodeName, artFile)
				continue
			}
			artPrefix := artifactsPrefix + "/" + artName
			consulStore.StoreConsulKeyAsString(artPrefix+"/name", artName)
			consulStore.StoreConsulKeyAsString(artPrefix+"/description", artDef.Description)
			consulStore.StoreConsulKeyAsString(artPrefix+"/file", artDef.File)
			consulStore.StoreConsulKeyAsString(artPrefix+"/type", artDef.Type)
			consulStore.StoreConsulKeyAsString(artPrefix+"/repository", artDef.Repository)
			consulStore.StoreConsulKeyAsString(artPrefix+"/deploy_path", artDef.DeployPath)
		}

		metadataPrefix := nodePrefix + "/metadata/"
		for metaName, metaValue := range node.Metadata {
			consulStore.StoreConsulKeyAsString(metadataPrefix+metaName, metaValue)
		}

		storeInterfaces(consulStore, node.Interfaces, nodePrefix, false)
	}

}

// storePropertyDefinition stores a property definition
func storePropertyDefinition(ctx context.Context, propPrefix, propName string, propDefinition tosca.PropertyDefinition) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	consulStore.StoreConsulKeyAsString(propPrefix+"/name", propName)
	consulStore.StoreConsulKeyAsString(propPrefix+"/description", propDefinition.Description)
	consulStore.StoreConsulKeyAsString(propPrefix+"/type", propDefinition.Type)
	consulStore.StoreConsulKeyAsString(propPrefix+"/entry_schema", propDefinition.EntrySchema.Type)
	storeValueAssignment(consulStore, propPrefix+"/default", propDefinition.Default)
	if propDefinition.Required == nil {
		// Required by default
		consulStore.StoreConsulKeyAsString(propPrefix+"/required", "true")
	} else {
		consulStore.StoreConsulKeyAsString(propPrefix+"/required", fmt.Sprint(*propDefinition.Required))
	}
}

// storeAttributeDefinition stores an attribute definition
func storeAttributeDefinition(ctx context.Context, attrPrefix, attrName string, attrDefinition tosca.AttributeDefinition) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	consulStore.StoreConsulKeyAsString(attrPrefix+"/name", attrName)
	consulStore.StoreConsulKeyAsString(attrPrefix+"/description", attrDefinition.Description)
	consulStore.StoreConsulKeyAsString(attrPrefix+"/type", attrDefinition.Type)
	consulStore.StoreConsulKeyAsString(attrPrefix+"/entry_schema", attrDefinition.EntrySchema.Type)
	storeValueAssignment(consulStore, attrPrefix+"/default", attrDefinition.Default)
	consulStore.StoreConsulKeyAsString(attrPrefix+"/status", attrDefinition.Status)
}

func storeComplexType(consulStore consulutil.ConsulStore, valuePath string, value interface{}) {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Slice:
		// Store an empty string all we want is to have a the value type here
		consulStore.StoreConsulKeyAsStringWithFlags(valuePath, "", uint64(tosca.ValueAssignmentList))
		for i := 0; i < v.Len(); i++ {
			storeComplexType(consulStore, path.Join(valuePath, strconv.Itoa(i)), v.Index(i).Interface())
		}
	case reflect.Map:
		// Store an empty string all we want is to have a the value type here
		consulStore.StoreConsulKeyAsStringWithFlags(valuePath, "", uint64(tosca.ValueAssignmentMap))
		for _, key := range v.MapKeys() {
			storeComplexType(consulStore, path.Join(valuePath, url.QueryEscape(fmt.Sprint(key.Interface()))), v.MapIndex(key).Interface())
		}
	default:
		// Default flag is for literal
		consulStore.StoreConsulKeyAsString(valuePath, fmt.Sprint(v))
	}
}

func storeValueAssignment(consulStore consulutil.ConsulStore, vaPrefix string, va *tosca.ValueAssignment) {
	if va == nil {
		return
	}
	switch va.Type {
	case tosca.ValueAssignmentLiteral:
		// Default flag is for literal
		consulStore.StoreConsulKeyAsString(vaPrefix, va.GetLiteral())
	case tosca.ValueAssignmentFunction:
		consulStore.StoreConsulKeyAsStringWithFlags(vaPrefix, va.GetFunction().String(), uint64(tosca.ValueAssignmentFunction))
	case tosca.ValueAssignmentList:
		storeComplexType(consulStore, vaPrefix, va.GetList())
	case tosca.ValueAssignmentMap:
		storeComplexType(consulStore, vaPrefix, va.GetMap())
	}
}

func storeMapValueAssignment(consulStore consulutil.ConsulStore, prefix string,
	mapValueAssignment map[string]*tosca.ValueAssignment) {

	for name, value := range mapValueAssignment {
		storeValueAssignment(consulStore, path.Join(prefix, name), value)
	}
}

func storeStringMap(consulStore consulutil.ConsulStore, prefix string,
	stringMap map[string]string) {

	for name, value := range stringMap {
		consulStore.StoreConsulKeyAsString(path.Join(prefix, name), value)
	}
}

func storeInputDefinition(consulStore consulutil.ConsulStore, inputPrefix, operationOutputPrefix string, inputDef tosca.Input) error {
	isValueAssignment := false
	isPropertyDefinition := false
	if inputDef.ValueAssign != nil {
		storeValueAssignment(consulStore, inputPrefix+"/data", inputDef.ValueAssign)
		isValueAssignment = true
		if inputDef.ValueAssign.Type == tosca.ValueAssignmentFunction {
			f := inputDef.ValueAssign.GetFunction()
			opOutputFuncs := f.GetFunctionsByOperator(tosca.GetOperationOutputOperator)
			for _, oof := range opOutputFuncs {
				if len(oof.Operands) != 4 {
					return errors.Errorf("Invalid %q TOSCA function: %v", tosca.GetOperationOutputOperator, oof)
				}
				entityName := url.QueryEscape(oof.Operands[0].String())
				if entityName == "TARGET" || entityName == "SOURCE" {
					return errors.Errorf("Can't use SOURCE or TARGET keyword in a %q in node type context: %v", tosca.GetOperationOutputOperator, oof)
				}
				interfaceName := strings.ToLower(url.QueryEscape(oof.Operands[1].String()))
				operationName := strings.ToLower(url.QueryEscape(oof.Operands[2].String()))
				outputVariableName := url.QueryEscape(oof.Operands[3].String())
				consulStore.StoreConsulKeyAsString(operationOutputPrefix+"/"+interfaceName+"/"+operationName+"/outputs/"+entityName+"/"+outputVariableName+"/expression", oof.String())
			}
		}
	}
	if inputDef.PropDef != nil {
		consulStore.StoreConsulKeyAsString(inputPrefix+"/type", inputDef.PropDef.Type)
		storeValueAssignment(consulStore, inputPrefix+"/default", inputDef.PropDef.Default)
		consulStore.StoreConsulKeyAsString(inputPrefix+"/description", inputDef.PropDef.Description)
		consulStore.StoreConsulKeyAsString(inputPrefix+"/status", inputDef.PropDef.Status)
		if inputDef.PropDef.Required == nil {
			// Required by default
			consulStore.StoreConsulKeyAsString(inputPrefix+"/required", "true")
		} else {
			consulStore.StoreConsulKeyAsString(inputPrefix+"/required", strconv.FormatBool(*inputDef.PropDef.Required))
		}
		isPropertyDefinition = true
	}

	consulStore.StoreConsulKeyAsString(inputPrefix+"/is_value_assignment", strconv.FormatBool(isValueAssignment))
	consulStore.StoreConsulKeyAsString(inputPrefix+"/is_property_definition", strconv.FormatBool(isPropertyDefinition))
	return nil
}

func storeInterfaces(consulStore consulutil.ConsulStore, interfaces map[string]tosca.InterfaceDefinition, prefix string, isRelationshipType bool) error {
	for intTypeName, intMap := range interfaces {
		intTypeName = strings.ToLower(intTypeName)
		interfacePrefix := path.Join(prefix, "interfaces", intTypeName)
		// Store Global inputs
		for inputName, inputDef := range intMap.Inputs {
			inputPrefix := path.Join(interfacePrefix, "inputs", inputName)
			consulStore.StoreConsulKeyAsString(inputPrefix+"/name", inputName)
			err := storeInputDefinition(consulStore, inputPrefix, interfacePrefix, inputDef)
			if err != nil {
				return err
			}
		}

		for opName, operationDef := range intMap.Operations {
			opName = strings.ToLower(opName)
			operationPrefix := path.Join(interfacePrefix, opName)
			consulStore.StoreConsulKeyAsString(operationPrefix+"/name", opName)
			consulStore.StoreConsulKeyAsString(operationPrefix+"/description", operationDef.Description)

			for inputName, inputDef := range operationDef.Inputs {
				inputPrefix := path.Join(operationPrefix, "inputs", inputName)
				consulStore.StoreConsulKeyAsString(inputPrefix+"/name", inputName)
				err := storeInputDefinition(consulStore, inputPrefix, interfacePrefix, inputDef)
				if err != nil {
					return err
				}
			}
			if operationDef.Implementation.Artifact != (tosca.ArtifactDefinition{}) {
				consulStore.StoreConsulKeyAsString(operationPrefix+"/implementation/file", operationDef.Implementation.Artifact.File)
				consulStore.StoreConsulKeyAsString(operationPrefix+"/implementation/type", operationDef.Implementation.Artifact.Type)
				consulStore.StoreConsulKeyAsString(operationPrefix+"/implementation/repository", operationDef.Implementation.Artifact.Repository)
				consulStore.StoreConsulKeyAsString(operationPrefix+"/implementation/description", operationDef.Implementation.Artifact.Description)
				consulStore.StoreConsulKeyAsString(operationPrefix+"/implementation/deploy_path", operationDef.Implementation.Artifact.DeployPath)

			} else {
				consulStore.StoreConsulKeyAsString(operationPrefix+"/implementation/primary", operationDef.Implementation.Primary)
				consulStore.StoreConsulKeyAsString(operationPrefix+"/implementation/dependencies", strings.Join(operationDef.Implementation.Dependencies, ","))
			}
			if operationDef.Implementation.OperationHost != "" {
				if err := checkOperationHost(operationDef.Implementation.OperationHost, isRelationshipType); err != nil {
					return err
				}
				consulStore.StoreConsulKeyAsString(operationPrefix+"/implementation/operation_host", strings.ToUpper(operationDef.Implementation.OperationHost))
			}
		}
	}
	return nil
}

func storeArtifacts(consulStore consulutil.ConsulStore, artifacts tosca.ArtifactDefMap, prefix string) {
	for artName, artDef := range artifacts {
		artPrefix := prefix + "/" + artName
		consulStore.StoreConsulKeyAsString(artPrefix+"/name", artName)
		consulStore.StoreConsulKeyAsString(artPrefix+"/description", artDef.Description)
		consulStore.StoreConsulKeyAsString(artPrefix+"/file", artDef.File)
		consulStore.StoreConsulKeyAsString(artPrefix+"/type", artDef.Type)
		consulStore.StoreConsulKeyAsString(artPrefix+"/repository", artDef.Repository)
		consulStore.StoreConsulKeyAsString(artPrefix+"/deploy_path", artDef.DeployPath)
	}
}

// storeNodeTypes stores topology types
func storeNodeTypes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath string) error {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	typesPrefix := path.Join(topologyPrefix, "types")
	for nodeTypeName, nodeType := range topology.NodeTypes {
		nodeTypePrefix := typesPrefix + "/" + nodeTypeName
		storeCommonType(consulStore, nodeType.Type, nodeTypePrefix, importPath)
		consulStore.StoreConsulKeyAsString(nodeTypePrefix+"/name", nodeTypeName)
		propertiesPrefix := nodeTypePrefix + "/properties"
		for propName, propDefinition := range nodeType.Properties {
			propPrefix := propertiesPrefix + "/" + propName
			storePropertyDefinition(ctx, propPrefix, propName, propDefinition)
		}

		requirementsPrefix := nodeTypePrefix + "/requirements"
		for reqIndex, reqMap := range nodeType.Requirements {
			for reqName, reqDefinition := range reqMap {
				reqPrefix := requirementsPrefix + "/" + strconv.Itoa(reqIndex)
				consulStore.StoreConsulKeyAsString(reqPrefix+"/name", reqName)
				consulStore.StoreConsulKeyAsString(reqPrefix+"/node", reqDefinition.Node)
				consulStore.StoreConsulKeyAsString(reqPrefix+"/occurrences/lower_bound", strconv.FormatUint(reqDefinition.Occurrences.LowerBound, 10))
				consulStore.StoreConsulKeyAsString(reqPrefix+"/occurrences/upper_bound", strconv.FormatUint(reqDefinition.Occurrences.UpperBound, 10))
				consulStore.StoreConsulKeyAsString(reqPrefix+"/relationship", reqDefinition.Relationship)
				consulStore.StoreConsulKeyAsString(reqPrefix+"/capability", reqDefinition.Capability)
				consulStore.StoreConsulKeyAsString(reqPrefix+"/capability_name", reqDefinition.CapabilityName)
			}
		}
		capabilitiesPrefix := nodeTypePrefix + "/capabilities"
		for capName, capability := range nodeType.Capabilities {
			capabilityPrefix := capabilitiesPrefix + "/" + capName

			consulStore.StoreConsulKeyAsString(capabilityPrefix+"/name", capName)
			consulStore.StoreConsulKeyAsString(capabilityPrefix+"/type", capability.Type)
			consulStore.StoreConsulKeyAsString(capabilityPrefix+"/description", capability.Description)
			consulStore.StoreConsulKeyAsString(capabilityPrefix+"/occurrences/lower_bound", strconv.FormatUint(capability.Occurrences.LowerBound, 10))
			consulStore.StoreConsulKeyAsString(capabilityPrefix+"/occurrences/upper_bound", strconv.FormatUint(capability.Occurrences.UpperBound, 10))
			consulStore.StoreConsulKeyAsString(capabilityPrefix+"/valid_sources", strings.Join(capability.ValidSourceTypes, ","))
			capabilityPropsPrefix := capabilityPrefix + "/properties"
			for propName, propValue := range capability.Properties {
				storeValueAssignment(consulStore, capabilityPropsPrefix+"/"+url.QueryEscape(propName), propValue)
			}
			capabilityAttrPrefix := capabilityPrefix + "/attributes"
			for attrName, attrValue := range capability.Attributes {
				storeValueAssignment(consulStore, capabilityAttrPrefix+"/"+url.QueryEscape(attrName), attrValue)
			}
		}

		err := storeInterfaces(consulStore, nodeType.Interfaces, nodeTypePrefix, false)
		if err != nil {
			return err
		}
		attributesPrefix := nodeTypePrefix + "/attributes"
		for attrName, attrDefinition := range nodeType.Attributes {
			attrPrefix := attributesPrefix + "/" + attrName
			storeAttributeDefinition(ctx, attrPrefix, attrName, attrDefinition)
			if attrDefinition.Default != nil && attrDefinition.Default.Type == tosca.ValueAssignmentFunction {
				f := attrDefinition.Default.GetFunction()
				opOutputFuncs := f.GetFunctionsByOperator(tosca.GetOperationOutputOperator)
				for _, oof := range opOutputFuncs {
					if len(oof.Operands) != 4 {
						return errors.Errorf("Invalid %q TOSCA function: %v", tosca.GetOperationOutputOperator, oof)
					}
					entityName := url.QueryEscape(oof.Operands[0].String())
					if entityName == "TARGET" || entityName == "SOURCE" {
						return errors.Errorf("Can't use SOURCE or TARGET keyword in a %q in node type context: %v", tosca.GetOperationOutputOperator, oof)
					}
					interfaceName := strings.ToLower(url.QueryEscape(oof.Operands[1].String()))
					operationName := strings.ToLower(url.QueryEscape(oof.Operands[2].String()))
					outputVariableName := url.QueryEscape(oof.Operands[3].String())
					consulStore.StoreConsulKeyAsString(nodeTypePrefix+"/interfaces/"+interfaceName+"/"+operationName+"/outputs/"+entityName+"/"+outputVariableName+"/expression", oof.String())
				}
			}
		}

		storeArtifacts(consulStore, nodeType.Artifacts, nodeTypePrefix+"/artifacts")

	}
	return nil
}

// storeRelationshipTypes stores topology relationships types
func storeRelationshipTypes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath string) error {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	for relationName, relationType := range topology.RelationshipTypes {
		relationTypePrefix := path.Join(topologyPrefix, "types", relationName)
		storeCommonType(consulStore, relationType.Type, relationTypePrefix, importPath)
		consulStore.StoreConsulKeyAsString(relationTypePrefix+"/name", relationName)
		propertiesPrefix := relationTypePrefix + "/properties"
		for propName, propDefinition := range relationType.Properties {
			propPrefix := propertiesPrefix + "/" + propName
			storePropertyDefinition(ctx, propPrefix, propName, propDefinition)
		}
		attributesPrefix := relationTypePrefix + "/attributes"
		for attrName, attrDefinition := range relationType.Attributes {
			attrPrefix := attributesPrefix + "/" + attrName
			storeAttributeDefinition(ctx, attrPrefix, attrName, attrDefinition)
			if attrDefinition.Default != nil && attrDefinition.Default.Type == tosca.ValueAssignmentFunction {
				f := attrDefinition.Default.GetFunction()
				opOutputFuncs := f.GetFunctionsByOperator(tosca.GetOperationOutputOperator)
				for _, oof := range opOutputFuncs {
					if len(oof.Operands) != 4 {
						return errors.Errorf("Invalid %q TOSCA function: %v", tosca.GetOperationOutputOperator, oof)
					}
					entityName := url.QueryEscape(oof.Operands[0].String())

					interfaceName := strings.ToLower(url.QueryEscape(oof.Operands[1].String()))
					operationName := strings.ToLower(url.QueryEscape(oof.Operands[2].String()))
					outputVariableName := url.QueryEscape(oof.Operands[3].String())
					consulStore.StoreConsulKeyAsString(relationTypePrefix+"/interfaces/"+interfaceName+"/"+operationName+"/outputs/"+entityName+"/"+outputVariableName+"/expression", oof.String())
				}
			}
		}

		err := storeInterfaces(consulStore, relationType.Interfaces, relationTypePrefix, true)
		if err != nil {
			return err
		}

		storeArtifacts(consulStore, relationType.Artifacts, relationTypePrefix+"/artifacts")

		consulStore.StoreConsulKeyAsString(relationTypePrefix+"/valid_target_type", strings.Join(relationType.ValidTargetTypes, ", "))

	}
	return nil
}

// storeCapabilityTypes stores topology capabilities types
func storeCapabilityTypes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	for capabilityTypeName, capabilityType := range topology.CapabilityTypes {
		capabilityTypePrefix := path.Join(topologyPrefix, "types", capabilityTypeName)
		storeCommonType(consulStore, capabilityType.Type, capabilityTypePrefix, importPath)
		consulStore.StoreConsulKeyAsString(capabilityTypePrefix+"/name", capabilityTypeName)
		propertiesPrefix := capabilityTypePrefix + "/properties"
		for propName, propDefinition := range capabilityType.Properties {
			propPrefix := propertiesPrefix + "/" + propName
			storePropertyDefinition(ctx, propPrefix, propName, propDefinition)
		}
		attributesPrefix := capabilityTypePrefix + "/attributes"
		for attrName, attrDefinition := range capabilityType.Attributes {
			attrPrefix := attributesPrefix + "/" + attrName
			storeAttributeDefinition(ctx, attrPrefix, attrName, attrDefinition)
		}
		consulStore.StoreConsulKeyAsString(capabilityTypePrefix+"/valid_source_types", strings.Join(capabilityType.ValidSourceTypes, ","))
	}
}

// storeTypes stores topology types
func storeArtifactTypes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	typesPrefix := path.Join(topologyPrefix, "types")
	for artTypeName, artType := range topology.ArtifactTypes {
		artTypePrefix := path.Join(typesPrefix, artTypeName)
		storeCommonType(consulStore, artType.Type, artTypePrefix, importPath)
		consulStore.StoreConsulKeyAsString(artTypePrefix+"/name", artTypeName)
		consulStore.StoreConsulKeyAsString(artTypePrefix+"/mime_type", artType.MimeType)
		consulStore.StoreConsulKeyAsString(artTypePrefix+"/file_ext", strings.Join(artType.FileExt, ","))
		propertiesPrefix := artTypePrefix + "/properties"
		for propName, propDefinition := range artType.Properties {
			propPrefix := propertiesPrefix + "/" + propName
			storePropertyDefinition(ctx, propPrefix, propName, propDefinition)
		}
	}
}

func storeWorkflowStep(consulStore consulutil.ConsulStore, deploymentID, workflowName, stepName string, step *tosca.Step) {
	stepPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "workflows", url.QueryEscape(workflowName), "steps", url.QueryEscape(stepName))
	if step.Target != "" {
		consulStore.StoreConsulKeyAsString(stepPrefix+"/target", step.Target)
	}
	if step.TargetRelationShip != "" {
		consulStore.StoreConsulKeyAsString(stepPrefix+"/target_relationship", step.TargetRelationShip)
	}
	if step.OperationHost != "" {
		consulStore.StoreConsulKeyAsString(stepPrefix+"/operation_host", strings.ToUpper(step.OperationHost))
	}
	activitiesPrefix := stepPrefix + "/activities"
	for actIndex, activity := range step.Activities {
		activityPrefix := activitiesPrefix + "/" + strconv.Itoa(actIndex)
		if activity.CallOperation != "" {
			// Preserve case for requirement and target node name in case of relationship operation
			opSlice := strings.SplitN(activity.CallOperation, "/", 2)
			opSlice[0] = strings.ToLower(opSlice[0])
			consulStore.StoreConsulKeyAsString(activityPrefix+"/call-operation", strings.Join(opSlice, "/"))
		}
		if activity.Delegate != "" {
			consulStore.StoreConsulKeyAsString(activityPrefix+"/delegate", strings.ToLower(activity.Delegate))
		}
		if activity.SetState != "" {
			consulStore.StoreConsulKeyAsString(activityPrefix+"/set-state", strings.ToLower(activity.SetState))
		}
		if activity.Inline != "" {
			consulStore.StoreConsulKeyAsString(activityPrefix+"/inline", strings.ToLower(activity.Inline))
		}
	}
	for _, next := range step.OnSuccess {
		// store in consul a prefix for the next step to be executed ; this prefix is stepPrefix/next/onSuccess_value
		consulStore.StoreConsulKeyAsString(fmt.Sprintf("%s/next/%s", stepPrefix, url.QueryEscape(next)), "")
	}
	for _, of := range step.OnFailure {
		// store in consul a prefix for the next step to be executed on failure ; this prefix is stepPrefix/on-failure/onFailure_value
		consulStore.StoreConsulKeyAsString(fmt.Sprintf("%s/on-failure/%s", stepPrefix, url.QueryEscape(of)), "")
	}
	for _, oc := range step.OnCancel {
		// store in consul a prefix for the next step to be executed on cancel ; this prefix is stepPrefix/on-cancel/onCancel_value
		consulStore.StoreConsulKeyAsString(fmt.Sprintf("%s/on-cancel/%s", stepPrefix, url.QueryEscape(oc)), "")
	}
}

// storeWorkflow stores a workflow
func storeWorkflow(consulStore consulutil.ConsulStore, deploymentID, workflowName string, workflow tosca.Workflow) {
	for stepName, step := range workflow.Steps {
		storeWorkflowStep(consulStore, deploymentID, workflowName, stepName, step)
	}
}

// storeWorkflows stores topology workflows
func storeWorkflows(ctx context.Context, topology tosca.Topology, deploymentID string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	for wfName, workflow := range topology.TopologyTemplate.Workflows {
		storeWorkflow(consulStore, deploymentID, wfName, workflow)
	}
}

// checkNestedWorkflows detect potential cycle in all nested workflows
func checkNestedWorkflows(topology tosca.Topology) error {
	for wfName, workflow := range topology.TopologyTemplate.Workflows {
		nestedWfs := make([]string, 0)
		if err := checkNestedWorkflow(topology, workflow, nestedWfs, wfName); err != nil {
			return err
		}
	}
	return nil
}

// checkNestedWorkflows detect potential cycle in a nested workflow
func checkNestedWorkflow(topology tosca.Topology, workflow tosca.Workflow, nestedWfs []string, wfName string) error {
	nestedWfs = append(nestedWfs, wfName)
	for _, step := range workflow.Steps {
		for _, activity := range step.Activities {
			if activity.Inline != "" {
				if collections.ContainsString(nestedWfs, activity.Inline) {
					return errors.Errorf("A cycle has been detected in inline workflows [initial: %q, repeated: %q]", nestedWfs[0], activity.Inline)
				}
				if err := checkNestedWorkflow(topology, topology.TopologyTemplate.Workflows[activity.Inline], nestedWfs, activity.Inline); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// createInstancesForNode checks if the given node is hosted on a Scalable node, stores the number of required instances and sets the instance's status to INITIAL
func createInstancesForNode(ctx context.Context, kv *api.KV, deploymentID, nodeName string) error {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	nbInstances, err := GetDefaultNbInstancesForNode(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	createNodeInstances(consulStore, kv, nbInstances, deploymentID, nodeName)

	// Check for FIPConnectivity capabilities
	is, capabilityNodeName, err := HasAnyRequirementCapability(kv, deploymentID, nodeName, "network", "yorc.capabilities.openstack.FIPConnectivity")
	if err != nil {
		return err
	}
	if is {
		createNodeInstances(consulStore, kv, nbInstances, deploymentID, capabilityNodeName)
	}

	// Check for Assignable capabilities
	is, capabilityNodeName, err = HasAnyRequirementCapability(kv, deploymentID, nodeName, "assignment", "yorc.capabilities.Assignable")
	if err != nil {
		return err
	}
	if is {
		createNodeInstances(consulStore, kv, nbInstances, deploymentID, capabilityNodeName)
	}

	bs, bsNames, err := checkBlockStorage(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}

	if bs {
		for _, name := range bsNames {
			createNodeInstances(consulStore, kv, nbInstances, deploymentID, name)
		}

	}
	return nil
}

func registerImplementationTypes(ctx context.Context, kv *api.KV, deploymentID string) error {
	// We use synchronous communication with consul here to allow to check for duplicates
	types, err := GetTypes(kv, deploymentID)
	if err != nil {
		return err
	}
	for _, t := range types {
		isImpl, err := IsTypeDerivedFrom(kv, deploymentID, t, "tosca.artifacts.Implementation")
		if err != nil {
			if IsTypeMissingError(err) {
				// Bypassing this error it may happen in case of an used type let's trust Alien
				events.SimpleLogEntry(events.LogLevelWARN, deploymentID).RegisterAsString(fmt.Sprintf("[WARNING] %s", err))
				continue
			}
			return err
		}
		if isImpl {
			extensions, err := GetArtifactTypeExtensions(kv, deploymentID, t)
			if err != nil {
				return err
			}
			for _, ext := range extensions {
				ext = strings.ToLower(ext)
				check, err := GetImplementationArtifactForExtension(kv, deploymentID, ext)
				if err != nil {
					return err
				}
				if check != "" {
					return errors.Errorf("Duplicate implementation artifact file extension %q found in artifact %q and %q", ext, check, t)
				}
				extPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", implementationArtifactsExtensionsPath, ext)
				err = consulutil.StoreConsulKeyAsString(extPath, t)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// enhanceNodes walk through the topology nodes an for each of them if needed it creates the instances and fix alien BlockStorage declaration
func enhanceNodes(ctx context.Context, kv *api.KV, deploymentID string) error {
	ctxStore, errGroup, consulStore := consulutil.WithContext(ctx)
	ctxStore = context.WithValue(ctxStore, consulStoreKey, consulStore)
	nodes, err := GetNodes(kv, deploymentID)
	if err != nil {
		return err
	}
	computes := make([]string, 0)
	for _, nodeName := range nodes {
		err = fixGetOperationOutputForRelationship(ctx, kv, deploymentID, nodeName)
		if err != nil {
			return err
		}
		err = fixGetOperationOutputForHost(ctxStore, kv, deploymentID, nodeName)
		if err != nil {
			return err
		}

		substitutable, err := isSubstitutableNode(kv, deploymentID, nodeName)
		if err != nil {
			return err
		}
		if !substitutable {
			err = createInstancesForNode(ctxStore, kv, deploymentID, nodeName)
			if err != nil {
				return err
			}
			err = fixAlienBlockStorages(ctxStore, kv, deploymentID, nodeName)
			if err != nil {
				return err
			}
			var isCompute bool
			isCompute, err = IsNodeDerivedFrom(kv, deploymentID, nodeName, "tosca.nodes.Compute")
			if err != nil {
				return err
			}
			if isCompute {
				computes = append(computes, nodeName)
			}
		}
	}
	for _, nodeName := range computes {

		err = createMissingBlockStorageForNode(consulStore, kv, deploymentID, nodeName)
		if err != nil {
			return err
		}
	}
	err = errGroup.Wait()
	if err != nil {
		return err
	}

	_, errGroup, consulStore = consulutil.WithContext(ctx)
	for _, nodeName := range nodes {
		err = createRelationshipInstances(consulStore, kv, deploymentID, nodeName)
		if err != nil {
			return err
		}
	}

	enhanceWorkflows(consulStore, kv, deploymentID)
	return errGroup.Wait()
}

// In this function we iterate over all node to know which node need to have a HOST output and search for this HOST and tell him to export this output
func fixGetOperationOutputForHost(ctx context.Context, kv *api.KV, deploymentID, nodeName string) error {
	nodeType, err := GetNodeType(kv, deploymentID, nodeName)
	if nodeType != "" && err == nil {
		interfacesPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "types", nodeType, "interfaces")
		interfacesNamesPaths, _, err := kv.Keys(interfacesPrefix+"/", "/", nil)
		if err != nil {
			return err
		}
		for _, interfaceNamePath := range interfacesNamesPaths {
			operationsPaths, _, err := kv.Keys(interfaceNamePath+"/", "/", nil)
			if err != nil {
				return err
			}
			for _, operationPath := range operationsPaths {
				outputsPrefix := path.Join(operationPath, "outputs", "HOST")
				outputsNamesPaths, _, err := kv.Keys(outputsPrefix+"/", "/", nil)
				if err != nil {
					return err
				}
				if outputsNamesPaths == nil || len(outputsNamesPaths) == 0 {
					continue
				}
				for _, outputNamePath := range outputsNamesPaths {
					hostedOn, err := GetHostedOnNode(kv, deploymentID, nodeName)
					if err != nil {
						return nil
					} else if hostedOn == "" {
						return errors.New("Fail to get the hostedOn to fix the output")
					}
					if hostedNodeType, err := GetNodeType(kv, deploymentID, hostedOn); hostedNodeType != "" && err == nil {
						consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "types", hostedNodeType, "interfaces", path.Base(interfaceNamePath), path.Base(operationPath), "outputs", "SELF", path.Base(outputNamePath), "expression"), "get_operation_output: [SELF,"+path.Base(interfaceNamePath)+","+path.Base(operationPath)+","+path.Base(outputNamePath)+"]")
					}
				}
			}
		}
	}
	if err != nil {
		return err
	}

	return nil
}

// This function help us to fix the get_operation_output when it on a relationship, to tell to the SOURCE or TARGET to store the exported value in consul
// Ex: To get an variable from a past operation or a future operation
func fixGetOperationOutputForRelationship(ctx context.Context, kv *api.KV, deploymentID, nodeName string) error {
	reqPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName, "requirements")
	reqName, _, err := kv.Keys(reqPath+"/", "/", nil)
	if err != nil {
		return err
	}
	for _, reqKeyIndex := range reqName {
		relationshipType, err := GetRelationshipForRequirement(kv, deploymentID, nodeName, path.Base(reqKeyIndex))
		if err != nil {
			return err
		}
		relationshipPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "types", relationshipType, "interfaces")
		interfaceNamesPaths, _, err := kv.Keys(relationshipPrefix+"/", "/", nil)
		if err != nil {
			return err
		}
		for _, interfaceNamePath := range interfaceNamesPaths {
			operationsNamesPaths, _, err := kv.Keys(interfaceNamePath+"/", "/", nil)
			if err != nil {
				return err
			}
			for _, operationNamePath := range operationsNamesPaths {
				modEntityNamesPaths, _, err := kv.Keys(operationNamePath+"/outputs/", "/", nil)
				if err != nil {
					return err
				}
				for _, modEntityNamePath := range modEntityNamesPaths {
					outputsNamesPaths, _, _ := kv.Keys(modEntityNamePath+"/", "/", nil)
					if err != nil {
						return err
					}
					for _, outputNamePath := range outputsNamesPaths {
						if path.Base(modEntityNamePath) != "SOURCE" || path.Base(modEntityNamePath) != "TARGET" {
							continue
						}
						var nodeType string
						if path.Base(modEntityNamePath) == "SOURCE" {
							nodeType, _ = GetNodeType(kv, deploymentID, nodeName)
						} else if path.Base(modEntityNamePath) == "TARGET" {
							targetNode, err := GetTargetNodeForRequirement(kv, deploymentID, nodeName, reqKeyIndex)
							if err != nil {
								return err
							}
							nodeType, _ = GetNodeType(kv, deploymentID, targetNode)
						}
						consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "types", nodeType, "interfaces", path.Base(interfaceNamePath), path.Base(operationNamePath), "outputs", "SELF", path.Base(outputNamePath), "expression"), "get_operation_output: [SELF,"+path.Base(interfaceNamePath)+","+path.Base(operationNamePath)+","+path.Base(outputNamePath)+"]")
					}
				}
			}
		}

	}
	return nil
}

// fixAlienBlockStorages rewrites the relationship between a BlockStorage and a Compute to match the TOSCA specification
func fixAlienBlockStorages(ctx context.Context, kv *api.KV, deploymentID, nodeName string) error {
	isBS, err := IsNodeDerivedFrom(kv, deploymentID, nodeName, "tosca.nodes.BlockStorage")
	if err != nil {
		return err
	}
	if isBS {
		attachReqs, err := GetRequirementsKeysByTypeForNode(kv, deploymentID, nodeName, "attachment")
		if err != nil {
			return err
		}
		for _, attachReq := range attachReqs {
			req := tosca.RequirementAssignment{}
			req.Node = nodeName
			kvp, _, err := kv.Get(path.Join(attachReq, "node"), nil)
			if err != nil {
				return errors.Wrapf(err, "Failed to fix Alien-specific BlockStorage %q", nodeName)
			}
			var computeNodeName string
			if kvp != nil {
				computeNodeName = string(kvp.Value)
			}
			kvp, _, err = kv.Get(path.Join(attachReq, "capability"), nil)
			if err != nil {
				return errors.Wrapf(err, "Failed to fix Alien-specific BlockStorage %q", nodeName)
			}
			if kvp != nil {
				req.Capability = string(kvp.Value)
			}
			kvp, _, err = kv.Get(path.Join(attachReq, "relationship"), nil)
			if err != nil {
				return errors.Wrapf(err, "Failed to fix Alien-specific BlockStorage %q", nodeName)
			}
			if kvp != nil {
				req.Relationship = string(kvp.Value)
			}
			device, err := GetNodePropertyValue(kv, deploymentID, nodeName, "device")
			if err != nil {
				return errors.Wrapf(err, "Failed to fix Alien-specific BlockStorage %q", nodeName)
			}

			req.RelationshipProps = make(map[string]*tosca.ValueAssignment)

			if device != nil {
				va := &tosca.ValueAssignment{}
				if device.RawString() != "" {
					err = yaml.Unmarshal([]byte(device.RawString()), &va)
					if err != nil {
						return errors.Wrapf(err, "Failed to fix Alien-specific BlockStorage %q, failed to parse device property", nodeName)
					}
				}
				req.RelationshipProps["device"] = va
			}

			// Get all requirement properties
			kvps, _, err := kv.List(path.Join(attachReq, "properties"), nil)
			if err != nil {
				return errors.Wrapf(err, "Failed to fix Alien-specific BlockStorage %q", nodeName)
			}
			for _, kvp := range kvps {
				va := &tosca.ValueAssignment{}
				err := yaml.Unmarshal(kvp.Value, va)
				if err != nil {
					return errors.Wrapf(err, "Failed to fix Alien-specific BlockStorage %q", nodeName)
				}
				req.RelationshipProps[path.Base(kvp.Key)] = va
			}
			newReqID, err := GetNbRequirementsForNode(kv, deploymentID, computeNodeName)
			if err != nil {
				return err
			}

			// Do not share the consul store as we have to compute the number of requirements for nodes and as we will modify it asynchronously it may lead to overwriting
			ctxStore, errgroup, consulStore := consulutil.WithContext(ctx)
			ctxStore = context.WithValue(ctxStore, consulStoreKey, consulStore)

			storeRequirementAssignment(ctxStore, req, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", computeNodeName, "requirements", fmt.Sprint(newReqID)), "local_storage")

			err = errgroup.Wait()
			if err != nil {
				return err
			}
		}

	}

	return nil
}

/**
This function create a given number of floating IP instances
*/
func createNodeInstances(consulStore consulutil.ConsulStore, kv *api.KV, numberInstances uint32, deploymentID, nodeName string) {

	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)

	consulStore.StoreConsulKeyAsString(path.Join(nodePath, "nbInstances"), strconv.FormatUint(uint64(numberInstances), 10))

	for i := uint32(0); i < numberInstances; i++ {
		instanceName := strconv.FormatUint(uint64(i), 10)
		createNodeInstance(kv, consulStore, deploymentID, nodeName, instanceName)
	}
}

// createInstancesForNode checks if the given node is hosted on a Scalable node, stores the number of required instances and sets the instance's status to INITIAL
func createMissingBlockStorageForNode(consulStore consulutil.ConsulStore, kv *api.KV, deploymentID, nodeName string) error {
	requirementsKey, err := GetRequirementsKeysByTypeForNode(kv, deploymentID, nodeName, "local_storage")
	if err != nil {
		return err
	}

	nbInstances, err := GetNbInstancesForNode(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}

	var bsName []string

	for _, requirement := range requirementsKey {
		capability, _, err := kv.Get(path.Join(requirement, "capability"), nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		} else if capability == nil {
			continue
		}

		bsNode, _, err := kv.Get(path.Join(requirement, "node"), nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)

		}

		bsName = append(bsName, string(bsNode.Value))
	}

	for _, name := range bsName {
		createNodeInstances(consulStore, kv, nbInstances, deploymentID, name)
	}

	return nil
}

/**
This function check if a nodes need a block storage, and return the name of BlockStorage node.
*/
func checkBlockStorage(kv *api.KV, deploymentID, nodeName string) (bool, []string, error) {
	requirementsKey, err := GetRequirementsKeysByTypeForNode(kv, deploymentID, nodeName, "local_storage")
	if err != nil {
		return false, nil, err
	}

	var bsName []string

	for _, requirement := range requirementsKey {
		capability, _, err := kv.Get(path.Join(requirement, "capability"), nil)
		if err != nil {
			return false, nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		} else if capability == nil {
			continue
		}

		bsNode, _, err := kv.Get(path.Join(requirement, "node"), nil)
		if err != nil {
			return false, nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)

		}

		bsName = append(bsName, string(bsNode.Value))
	}

	return true, bsName, nil
}

func checkOperationHost(operationHost string, isRelationshipType bool) error {
	operationHostUpper := strings.ToUpper(operationHost)
	if isRelationshipType {
		// Allowed values are SOURCE, TARGET and ORCHESTRATOR
		switch operationHostUpper {
		case "ORCHESTRATOR":
		case "SOURCE":
		case "TARGET":
			return nil
		default:
			return errors.Errorf("Invalid value for Implementation operation_host property associated to a relationship type :%q", operationHost)

		}
	} else {
		// Allowed values are SELF, HOST and ORCHESTRATOR
		switch operationHostUpper {
		case "HOST":
		case "ORCHESTRATOR":
		case "SELF":
			return nil
		default:
			return errors.Errorf("Invalid value for Implementation operation_host property associated to non-relationship type:%q", operationHost)

		}
	}
	return nil
}

func enhanceWorkflows(consulStore consulutil.ConsulStore, kv *api.KV, deploymentID string) error {
	wf, err := ReadWorkflow(kv, deploymentID, "run")
	if err != nil {
		return err
	}
	var wasUpdated bool
	for sn, s := range wf.Steps {
		var isCancellable bool
		for _, a := range s.Activities {
			switch strings.ToLower(a.CallOperation) {
			case tosca.RunnableSubmitOperationName, tosca.RunnableRunOperationName:
				isCancellable = true
			}
		}
		if isCancellable && len(s.OnCancel) == 0 {
			// Cancellable and on-cancel not defined
			// Check if there is an cancel op
			hasCancelOp, err := IsOperationImplemented(kv, deploymentID, s.Target, tosca.RunnableCancelOperationName)
			if err != nil {
				return err
			}
			if hasCancelOp {
				cancelStep := &tosca.Step{
					Target:             s.Target,
					TargetRelationShip: s.TargetRelationShip,
					OperationHost:      s.OperationHost,
					Activities: []tosca.Activity{
						tosca.Activity{
							CallOperation: tosca.RunnableCancelOperationName,
						},
					},
				}
				csName := "yorc_automatic_cancellation_of_" + sn
				wf.Steps[csName] = cancelStep
				s.OnCancel = []string{csName}
				wasUpdated = true
			}
		}
	}
	if wasUpdated {
		storeWorkflow(consulStore, deploymentID, "run", wf)
	}
	return nil
}
