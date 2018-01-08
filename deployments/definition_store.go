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
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/registry"
	"novaforge.bull.com/starlings-janus/janus/tosca"
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
		return errors.Wrapf(err, "Failed to open definition file %q", defPath)
	}
	defBytes, err := ioutil.ReadAll(definition)
	if err != nil {
		return errors.Wrapf(err, "Failed to open definition file %q", defPath)
	}

	err = yaml.Unmarshal(defBytes, &topology)
	if err != nil {
		return errors.Wrapf(err, "Failed to unmarshal yaml definition for file %q", defPath)
	}

	err = storeDeployment(ctx, topology, deploymentID, filepath.Dir(defPath))
	if err != nil {
		return errors.Wrapf(err, "Failed to store TOSCA Definition for deployment with id %q, (file path %q)", deploymentID, defPath)
	}
	err = registerImplementationTypes(ctx, kv, deploymentID)
	if err != nil {
		return err
	}

	return enhanceNodes(ctx, kv, deploymentID)
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
	log.Debugf("Storing topology with name %q (Import prefix %q)", topology.Name, importPrefix)
	storeTopologyTopLevelKeyNames(ctx, topology, path.Join(topologyPrefix, importPrefix))
	if err := storeImports(ctx, topology, deploymentID, topologyPrefix, importPath, rootDefPath); err != nil {
		return err
	}
	storeRepositories(ctx, topology, topologyPrefix)
	storeDataTypes(ctx, topology, topologyPrefix)
	storeInputs(ctx, topology, topologyPrefix)
	storeOutputs(ctx, topology, topologyPrefix)
	storeNodes(ctx, topology, topologyPrefix, importPath, rootDefPath)
	if err := storeTypes(ctx, topology, topologyPrefix, importPath); err != nil {
		return err
	}
	if err := storeRelationshipTypes(ctx, topology, topologyPrefix, importPath); err != nil {
		return err
	}
	storeCapabilityTypes(ctx, topology, topologyPrefix)
	storeArtifactTypes(ctx, topology, topologyPrefix)
	storeWorkflows(ctx, topology, deploymentID)
	return nil
}

// storeTopologyTopLevelKeyNames stores top level keynames for a topology.
//
// This may be done under the import path in case of imports.
func storeTopologyTopLevelKeyNames(ctx context.Context, topology tosca.Topology, topologyPrefix string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	consulStore.StoreConsulKeyAsString(topologyPrefix+"/tosca_version", topology.TOSCAVersion)
	consulStore.StoreConsulKeyAsString(topologyPrefix+"/description", topology.Description)
	consulStore.StoreConsulKeyAsString(topologyPrefix+"/name", topology.Name)
	consulStore.StoreConsulKeyAsString(topologyPrefix+"/version", topology.Version)
	consulStore.StoreConsulKeyAsString(topologyPrefix+"/author", topology.Author)
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

func storeCommonType(consulStore consulutil.ConsulStore, commonType tosca.Type, typePrefix string) {
	consulStore.StoreConsulKeyAsString(path.Join(typePrefix, "derived_from"), commonType.DerivedFrom)
	consulStore.StoreConsulKeyAsString(path.Join(typePrefix, "version"), commonType.Version)
	consulStore.StoreConsulKeyAsString(path.Join(typePrefix, "description"), commonType.Description)
	for metaName, metaValue := range commonType.Metadata {
		consulStore.StoreConsulKeyAsString(path.Join(typePrefix, "metadata", metaName), metaValue)
	}
}

//storeDataTypes store data types
func storeDataTypes(ctx context.Context, topology tosca.Topology, topologyPrefix string) error {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	dataTypesPrefix := path.Join(topologyPrefix, "types")
	for dataTypeName, dataType := range topology.DataTypes {
		dtPrefix := path.Join(dataTypesPrefix, dataTypeName)
		storeCommonType(consulStore, dataType.Type, dtPrefix)
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
		for importName, value := range element {
			importedTopology := tosca.Topology{}
			importValue := strings.Trim(value.File, " \t")
			if strings.HasPrefix(importValue, "<") && strings.HasSuffix(importValue, ">") {
				// Internal import
				importValue = strings.Trim(importValue, "<>")
				var defBytes []byte
				var err error
				if defBytes, err = reg.GetToscaDefinition(importValue); err != nil {
					return errors.Errorf("Failed to import internal definition %s: %v", importValue, err)
				}
				if err = yaml.Unmarshal(defBytes, &importedTopology); err != nil {
					return errors.Errorf("Failed to parse internal definition %s: %v", importValue, err)
				}
				errGroup.Go(func() error {
					return storeTopology(ctx, importedTopology, deploymentID, topologyPrefix, path.Join("imports", importName), "", rootDefPath)
				})
			} else {
				uploadFile := filepath.Join(rootDefPath, filepath.FromSlash(importPath), filepath.FromSlash(value.File))

				definition, err := os.Open(uploadFile)
				if err != nil {
					return errors.Errorf("Failed to parse internal definition %s: %v", importValue, err)
				}

				defBytes, err := ioutil.ReadAll(definition)
				if err != nil {
					return errors.Errorf("Failed to parse internal definition %s: %v", importValue, err)
				}

				if err = yaml.Unmarshal(defBytes, &importedTopology); err != nil {
					return errors.Errorf("Failed to parse internal definition %s: %v", importValue, err)
				}

				errGroup.Go(func() error {
					return storeTopology(ctx, importedTopology, deploymentID, topologyPrefix, path.Join("imports", importPath, importName), path.Dir(path.Join(importPath, value.File)), rootDefPath)
				})
			}

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
			consulStore.StoreConsulKeyAsString(artPrefix+"/file", path.Join(importPath, artDef.File))
			consulStore.StoreConsulKeyAsString(artPrefix+"/type", artDef.Type)
			consulStore.StoreConsulKeyAsString(artPrefix+"/repository", artDef.Repository)
			consulStore.StoreConsulKeyAsString(artPrefix+"/deploy_path", artDef.DeployPath)
		}
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

// storeTypes stores topology types
func storeTypes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath string) error {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	typesPrefix := path.Join(topologyPrefix, "types")
	for nodeTypeName, nodeType := range topology.NodeTypes {
		nodeTypePrefix := typesPrefix + "/" + nodeTypeName
		storeCommonType(consulStore, nodeType.Type, nodeTypePrefix)
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
				propPrefix := capabilityPropsPrefix + "/" + propName
				storePropertyDefinition(ctx, propPrefix, propName, propValue)
			}
			capabilityAttrPrefix := capabilityPrefix + "/attributes"
			for attrName, attrValue := range capability.Attributes {
				attrPrefix := capabilityAttrPrefix + "/" + attrName
				storeAttributeDefinition(ctx, attrPrefix, attrName, attrValue)
			}
		}

		interfacesPrefix := nodeTypePrefix + "/interfaces"
		for intTypeName, intMap := range nodeType.Interfaces {
			intTypeName = strings.ToLower(intTypeName)
			for intName, intDef := range intMap {
				intName = strings.ToLower(intName)
				intPrefix := path.Join(interfacesPrefix, intTypeName, intName)
				consulStore.StoreConsulKeyAsString(intPrefix+"/name", intName)
				consulStore.StoreConsulKeyAsString(intPrefix+"/description", intDef.Description)

				for inputName, inputDef := range intDef.Inputs {
					inputPrefix := path.Join(intPrefix, "inputs", inputName)
					consulStore.StoreConsulKeyAsString(inputPrefix+"/name", inputName)
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
								consulStore.StoreConsulKeyAsString(nodeTypePrefix+"/interfaces/"+interfaceName+"/"+operationName+"/outputs/"+entityName+"/"+outputVariableName+"/expression", oof.String())
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
				}
				if intDef.Implementation.Artifact != (tosca.ArtifactDefinition{}) {
					consulStore.StoreConsulKeyAsString(intPrefix+"/implementation/file", intDef.Implementation.Artifact.File)
					consulStore.StoreConsulKeyAsString(intPrefix+"/implementation/type", intDef.Implementation.Artifact.Type)
					consulStore.StoreConsulKeyAsString(intPrefix+"/implementation/repository", intDef.Implementation.Artifact.Repository)
					consulStore.StoreConsulKeyAsString(intPrefix+"/implementation/description", intDef.Implementation.Artifact.Description)
					consulStore.StoreConsulKeyAsString(intPrefix+"/implementation/deploy_path", intDef.Implementation.Artifact.DeployPath)

				} else {
					consulStore.StoreConsulKeyAsString(intPrefix+"/implementation/primary", path.Join(importPath, intDef.Implementation.Primary))
					consulStore.StoreConsulKeyAsString(intPrefix+"/implementation/dependencies", strings.Join(intDef.Implementation.Dependencies, ","))
					if err := checkOperationHost(intDef.Implementation.OperationHost); err != nil {
						return err
					}
					consulStore.StoreConsulKeyAsString(intPrefix+"/implementation/operation_host", intDef.Implementation.OperationHost)
				}
			}
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

		artifactsPrefix := nodeTypePrefix + "/artifacts"
		for artName, artDef := range nodeType.Artifacts {
			artPrefix := artifactsPrefix + "/" + artName
			consulStore.StoreConsulKeyAsString(artPrefix+"/name", artName)
			consulStore.StoreConsulKeyAsString(artPrefix+"/description", artDef.Description)
			consulStore.StoreConsulKeyAsString(artPrefix+"/file", path.Join(importPath, artDef.File))
			consulStore.StoreConsulKeyAsString(artPrefix+"/type", artDef.Type)
			consulStore.StoreConsulKeyAsString(artPrefix+"/repository", artDef.Repository)
			consulStore.StoreConsulKeyAsString(artPrefix+"/deploy_path", artDef.DeployPath)
		}

	}
	return nil
}

// storeRelationshipTypes stores topology relationships types
func storeRelationshipTypes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath string) error {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	for relationName, relationType := range topology.RelationshipTypes {
		relationTypePrefix := path.Join(topologyPrefix, "types", relationName)
		storeCommonType(consulStore, relationType.Type, relationTypePrefix)
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

		interfacesPrefix := relationTypePrefix + "/interfaces"
		for intTypeName, intMap := range relationType.Interfaces {
			intTypeName = strings.ToLower(intTypeName)
			for intName, intDef := range intMap {
				intName = strings.ToLower(intName)
				intPrefix := path.Join(interfacesPrefix, intTypeName, intName)
				consulStore.StoreConsulKeyAsString(intPrefix+"/name", intName)
				consulStore.StoreConsulKeyAsString(intPrefix+"/description", intDef.Description)

				for inputName, inputDef := range intDef.Inputs {
					inputPrefix := path.Join(intPrefix, "inputs", inputName)
					consulStore.StoreConsulKeyAsString(inputPrefix+"/name", inputName)
					isValueAssignement := false
					isPropertyDefinition := false
					if inputDef.ValueAssign != nil {
						storeValueAssignment(consulStore, inputPrefix+"/data", inputDef.ValueAssign)
						isValueAssignement = true
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
								consulStore.StoreConsulKeyAsString(relationTypePrefix+"/interfaces/"+interfaceName+"/"+operationName+"/outputs/"+entityName+"/"+outputVariableName+"/expression", oof.String())
							}
						}
					}
					if inputDef.PropDef != nil {
						consulStore.StoreConsulKeyAsString(inputPrefix+"/type", inputDef.PropDef.Type)
						storeValueAssignment(consulStore, inputPrefix+"/default", inputDef.PropDef.Default)
						consulStore.StoreConsulKeyAsString(inputPrefix+"/description", inputDef.PropDef.Description)
						consulStore.StoreConsulKeyAsString(inputPrefix+"/status", inputDef.PropDef.Status)
						if inputDef.PropDef.Required == nil {
							consulStore.StoreConsulKeyAsString(inputPrefix+"/required", "true")
						} else {
							consulStore.StoreConsulKeyAsString(inputPrefix+"/required", strconv.FormatBool(*inputDef.PropDef.Required))
						}

						isPropertyDefinition = true
					}

					consulStore.StoreConsulKeyAsString(inputPrefix+"/is_value_assignment", strconv.FormatBool(isValueAssignement))
					consulStore.StoreConsulKeyAsString(inputPrefix+"/is_property_definition", strconv.FormatBool(isPropertyDefinition))
				}
				consulStore.StoreConsulKeyAsString(intPrefix+"/implementation/primary", path.Join(importPath, intDef.Implementation.Primary))
				consulStore.StoreConsulKeyAsString(intPrefix+"/implementation/dependencies", strings.Join(intDef.Implementation.Dependencies, ","))
				if err := checkOperationHost(intDef.Implementation.OperationHost); err != nil {
					return err
				}
				consulStore.StoreConsulKeyAsString(intPrefix+"/implementation/operation_host", intDef.Implementation.OperationHost)
			}
		}

		artifactsPrefix := relationTypePrefix + "/artifacts"
		for artName, artDef := range relationType.Artifacts {
			artPrefix := artifactsPrefix + "/" + artName
			consulStore.StoreConsulKeyAsString(artPrefix+"/name", artName)
			consulStore.StoreConsulKeyAsString(artPrefix+"/description", artDef.Description)
			consulStore.StoreConsulKeyAsString(artPrefix+"/file", path.Join(importPath, artDef.File))
			consulStore.StoreConsulKeyAsString(artPrefix+"/type", artDef.Type)
			consulStore.StoreConsulKeyAsString(artPrefix+"/repository", artDef.Repository)
			consulStore.StoreConsulKeyAsString(artPrefix+"/deploy_path", artDef.DeployPath)
		}

		consulStore.StoreConsulKeyAsString(relationTypePrefix+"/valid_target_type", strings.Join(relationType.ValidTargetTypes, ", "))

	}
	return nil
}

// storeCapabilityTypes stores topology capabilities types
func storeCapabilityTypes(ctx context.Context, topology tosca.Topology, topologyPrefix string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	for capabilityTypeName, capabilityType := range topology.CapabilityTypes {
		capabilityTypePrefix := path.Join(topologyPrefix, "types", capabilityTypeName)
		storeCommonType(consulStore, capabilityType.Type, capabilityTypePrefix)
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
func storeArtifactTypes(ctx context.Context, topology tosca.Topology, topologyPrefix string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	typesPrefix := path.Join(topologyPrefix, "types")
	for artTypeName, artType := range topology.ArtifactTypes {
		artTypePrefix := path.Join(typesPrefix, artTypeName)
		storeCommonType(consulStore, artType.Type, artTypePrefix)
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

// storeWorkflows stores topology workflows
func storeWorkflows(ctx context.Context, topology tosca.Topology, deploymentID string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	workflowsPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "workflows")
	for wfName, workflow := range topology.TopologyTemplate.Workflows {
		workflowPrefix := workflowsPrefix + "/" + url.QueryEscape(wfName)
		for stepName, step := range workflow.Steps {
			stepPrefix := workflowPrefix + "/steps/" + url.QueryEscape(stepName)
			consulStore.StoreConsulKeyAsString(stepPrefix+"/target", step.Target)
			if step.TargetRelationShip != "" {
				consulStore.StoreConsulKeyAsString(stepPrefix+"/target_relationship", step.TargetRelationShip)
			}
			if step.OperationHost != "" {
				consulStore.StoreConsulKeyAsString(stepPrefix+"/operation_host", step.OperationHost)
			}
			if step.Activity.CallOperation != "" {
				// Preserve case for requirement and target node name in case of relationship operation
				opSlice := strings.SplitN(step.Activity.CallOperation, "/", 2)
				opSlice[0] = strings.ToLower(opSlice[0])
				consulStore.StoreConsulKeyAsString(stepPrefix+"/activity/call-operation", strings.Join(opSlice, "/"))
			}
			if step.Activity.Delegate != "" {
				consulStore.StoreConsulKeyAsString(stepPrefix+"/activity/delegate", strings.ToLower(step.Activity.Delegate))
			}
			if step.Activity.SetState != "" {
				consulStore.StoreConsulKeyAsString(stepPrefix+"/activity/set-state", strings.ToLower(step.Activity.SetState))
			}
			for _, next := range step.OnSuccess {
				consulStore.StoreConsulKeyAsString(fmt.Sprintf("%s/next/%s", stepPrefix, url.QueryEscape(next)), "")
			}
		}
	}
}

// createInstancesForNode checks if the given node is hosted on a Scalable node, stores the number of required instances and sets the instance's status to INITIAL
func createInstancesForNode(ctx context.Context, kv *api.KV, deploymentID, nodeName string) error {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	nbInstances, err := GetDefaultNbInstancesForNode(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	createNodeInstances(consulStore, kv, nbInstances, deploymentID, nodeName)
	ip, networkNodeName, err := checkFloattingIP(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if ip {
		createNodeInstances(consulStore, kv, nbInstances, deploymentID, networkNodeName)
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
				events.SimpleLogEntry(events.WARN, deploymentID).RegisterAsString(fmt.Sprintf("[WARNING] %s", err))
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
				_, err = kv.Put(&api.KVPair{Key: extPath, Value: []byte(t)}, nil)
				if err != nil {
					return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
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
	return errGroup.Wait()
}

// In this function we iterate over all node to know which node need to have an HOST output and search for this HOST and tell him to export this output
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
		attachReqs, err := GetRequirementsKeysByNameForNode(kv, deploymentID, nodeName, "attachment")
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
			found, device, err := GetNodeProperty(kv, deploymentID, nodeName, "device")
			if err != nil {
				return errors.Wrapf(err, "Failed to fix Alien-specific BlockStorage %q", nodeName)
			}
			if found {
				va := &tosca.ValueAssignment{}
				req.RelationshipProps = make(map[string]*tosca.ValueAssignment)
				if device != "" {
					err = yaml.Unmarshal([]byte(device), &va)
					if err != nil {
						return errors.Wrapf(err, "Failed to fix Alien-specific BlockStorage %q, failed to parse device property", nodeName)
					}
				}
				req.RelationshipProps["device"] = va
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
		createNodeInstance(consulStore, deploymentID, nodeName, instanceName)
	}
}

/**
This function check if a nodes need a floating IP, and return the name of Floating IP node.
*/
func checkFloattingIP(kv *api.KV, deploymentID, nodeName string) (bool, string, error) {
	requirementsKey, err := GetRequirementsKeysByNameForNode(kv, deploymentID, nodeName, "network")
	if err != nil {
		return false, "", err
	}

	for _, requirement := range requirementsKey {
		capability, _, err := kv.Get(path.Join(requirement, "capability"), nil)
		if err != nil {
			return false, "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		} else if capability == nil {
			continue
		}

		res, err := IsTypeDerivedFrom(kv, deploymentID, string(capability.Value), "janus.capabilities.openstack.FIPConnectivity")
		if err != nil {
			return false, "", err
		}

		if res {
			networkNode, _, err := kv.Get(path.Join(requirement, "node"), nil)
			if err != nil {
				return false, "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)

			}
			return true, string(networkNode.Value), nil
		}
	}

	return false, "", nil
}

// createInstancesForNode checks if the given node is hosted on a Scalable node, stores the number of required instances and sets the instance's status to INITIAL
func createMissingBlockStorageForNode(consulStore consulutil.ConsulStore, kv *api.KV, deploymentID, nodeName string) error {
	requirementsKey, err := GetRequirementsKeysByNameForNode(kv, deploymentID, nodeName, "local_storage")
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
This function check if a nodes need a floating IP, and return the name of Floating IP node.
*/
func checkBlockStorage(kv *api.KV, deploymentID, nodeName string) (bool, []string, error) {
	requirementsKey, err := GetRequirementsKeysByNameForNode(kv, deploymentID, nodeName, "local_storage")
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

func checkOperationHost(operationHost string) error {
	if operationHost != "" {
		switch strings.ToUpper(operationHost) {
		case "SELF":
		case "HOST":
		case "ORCHESTRATOR":
		case "SOURCE":
		case "TARGET":
			return nil
		default:
			return errors.Errorf("Invalid value for Implementation operation_host property:%q", operationHost)

		}
	}
	return nil
}
