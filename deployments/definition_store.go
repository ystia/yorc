package deployments

import (
	"context"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/url"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tosca"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// Internal type used to uniquely identify the errorgroup in a context
type ctxErrGrpKey struct{}

// Internal variable used to uniquely identify the errorgroup in a context
var errGrpKey ctxErrGrpKey

// Internal type used to uniquely identify the ConsulStore in a context
type ctxConsulStoreKey struct{}

// Internal variable used to uniquely identify the ConsulStore in a context
var consulStoreKey ctxConsulStoreKey

// StoreDeploymentDefinition takes a defPath and parse it as a tosca.Topology then it store it in consul under
// DeploymentKVPrefix/deploymentId
func StoreDeploymentDefinition(ctx context.Context, kv *api.KV, deploymentId string, defPath string) error {
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
		return errors.Wrapf(err, "Failed to unmarsh yaml definition for file %q", defPath)
	}

	err = storeDeployment(ctx, topology, deploymentId)
	if err != nil {
		return errors.Wrapf(err, "Failed to store TOSCA Definition for deployment with id %q, (file path %q)", deploymentId, defPath)
	}

	return createInstancesForNodes(ctx, kv, deploymentId)
}

// storeDeployment stores a whole deployment.
func storeDeployment(ctx context.Context, topology tosca.Topology, deploymentId string) error {
	errCtx, errGroup, consulStore := consulutil.WithContext(ctx)
	errCtx = context.WithValue(errCtx, errGrpKey, errGroup)
	errCtx = context.WithValue(errCtx, consulStoreKey, consulStore)
	consulStore.StoreConsulKeyAsString(path.Join(DeploymentKVPrefix, deploymentId, "status"), fmt.Sprint(INITIAL))

	errGroup.Go(func() error {
		return storeTopology(errCtx, topology, deploymentId, path.Join(DeploymentKVPrefix, deploymentId, "topology"), "", "")
	})

	return errGroup.Wait()
}

// storeTopology stores a given topology.
//
// The given topology may be an import in this case importPrefix and importPath should be specified
func storeTopology(ctx context.Context, topology tosca.Topology, deploymentId, topologyPrefix, importPrefix, importPath string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	log.Debugf("Storing topology with name %q (Import prefix %q)", topology.Name, importPrefix)
	storeTopologyTopLevelKeyNames(ctx, topology, path.Join(topologyPrefix, importPrefix))
	if err := storeImports(ctx, topology, deploymentId, topologyPrefix); err != nil {
		return err
	}
	storeOuputs(ctx, topology, topologyPrefix)
	storeNodes(ctx, topology, topologyPrefix, importPath)
	storeTypes(ctx, topology, topologyPrefix, importPath)
	storeRelationshipTypes(ctx, topology, topologyPrefix, importPath)
	storeCapabilityTypes(ctx, topology, topologyPrefix)
	storeWorkflows(ctx, topology, deploymentId)
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

// storeImports parses and store imports.
func storeImports(ctx context.Context, topology tosca.Topology, deploymentId, topologyPrefix string) error {
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
				if defBytes, err = tosca.Asset(importValue); err != nil {
					return fmt.Errorf("Failed to import internal definition %s: %v", importValue, err)
				}
				if err = yaml.Unmarshal(defBytes, &importedTopology); err != nil {
					return fmt.Errorf("Failed to parse internal definition %s: %v", importValue, err)
				}
				errGroup.Go(func() error {
					return storeTopology(ctx, importedTopology, deploymentId, topologyPrefix, path.Join("imports", importName), "")
				})
			} else {
				uploadFile := filepath.Join("work", "deployments", deploymentId, "overlay", filepath.FromSlash(value.File))

				definition, err := os.Open(uploadFile)
				if err != nil {
					return fmt.Errorf("Failed to parse internal definition %s: %v", importValue, err)

				}

				defBytes, err := ioutil.ReadAll(definition)
				if err != nil {
					return fmt.Errorf("Failed to parse internal definition %s: %v", importValue, err)
				}

				if err = yaml.Unmarshal(defBytes, &importedTopology); err != nil {
					return fmt.Errorf("Failed to parse internal definition %s: %v", importValue, err)
				}

				errGroup.Go(func() error {
					return storeTopology(ctx, importedTopology, deploymentId, topologyPrefix, path.Join("imports", importName), path.Dir(value.File))
				})
			}

		}
	}
	return nil
}

// storeOuputs stores topology outputs
func storeOuputs(ctx context.Context, topology tosca.Topology, topologyPrefix string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	outputsPrefix := path.Join(topologyPrefix, "outputs")
	for outputName, output := range topology.TopologyTemplate.Outputs {
		outputPrefix := path.Join(outputsPrefix, outputName)
		consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "name"), outputName)
		consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "description"), output.Description)
		consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "default"), output.Default)
		consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "required"), strconv.FormatBool(output.Required))
		consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "status"), output.Status)
		consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "type"), output.Type)
		consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "value"), output.Value.String())
	}
}

// storeNodes stores topology nodes
func storeNodes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	nodesPrefix := path.Join(topologyPrefix, "nodes")
	for nodeName, node := range topology.TopologyTemplate.NodeTemplates {
		nodePrefix := nodesPrefix + "/" + nodeName
		consulStore.StoreConsulKeyAsString(nodePrefix+"/status", "initial")
		consulStore.StoreConsulKeyAsString(nodePrefix+"/name", nodeName)
		consulStore.StoreConsulKeyAsString(nodePrefix+"/type", node.Type)
		propertiesPrefix := nodePrefix + "/properties"
		for propName, propValue := range node.Properties {
			consulStore.StoreConsulKeyAsString(propertiesPrefix+"/"+url.QueryEscape(propName), fmt.Sprint(propValue))
		}
		attributesPrefix := nodePrefix + "/attributes"
		for attrName, attrValue := range node.Attributes {
			consulStore.StoreConsulKeyAsString(attributesPrefix+"/"+url.QueryEscape(attrName), fmt.Sprint(attrValue))
		}
		capabilitiesPrefix := nodePrefix + "/capabilities"
		for capName, capability := range node.Capabilities {
			capabilityPrefix := capabilitiesPrefix + "/" + capName
			capabilityPropsPrefix := capabilityPrefix + "/properties"
			for propName, propValue := range capability.Properties {
				consulStore.StoreConsulKeyAsString(capabilityPropsPrefix+"/"+url.QueryEscape(propName), fmt.Sprint(propValue))
			}
			capabilityAttrPrefix := capabilityPrefix + "/attributes"
			for attrName, attrValue := range capability.Attributes {
				consulStore.StoreConsulKeyAsString(capabilityAttrPrefix+"/"+url.QueryEscape(attrName), fmt.Sprint(attrValue))
			}
		}
		requirementsPrefix := nodePrefix + "/requirements"
		for reqIndex, reqValueMap := range node.Requirements {
			for reqName, reqValue := range reqValueMap {
				reqPrefix := requirementsPrefix + "/" + strconv.Itoa(reqIndex)
				consulStore.StoreConsulKeyAsString(reqPrefix+"/name", reqName)
				consulStore.StoreConsulKeyAsString(reqPrefix+"/node", reqValue.Node)
				consulStore.StoreConsulKeyAsString(reqPrefix+"/relationship", reqValue.Relationship)
				consulStore.StoreConsulKeyAsString(reqPrefix+"/capability", reqValue.Capability)
				for propName, propValue := range reqValue.RelationshipProps {
					consulStore.StoreConsulKeyAsString(reqPrefix+"/properties/"+url.QueryEscape(propName), fmt.Sprint(propValue))
				}
			}
		}
		artifactsPrefix := nodePrefix + "/artifacts"
		for artName, artDef := range node.Artifacts {
			artPrefix := artifactsPrefix + "/" + artName
			consulStore.StoreConsulKeyAsString(artPrefix+"/name", artName)
			consulStore.StoreConsulKeyAsString(artPrefix+"/metatype", "artifact")
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
	consulStore.StoreConsulKeyAsString(propPrefix+"/default", propDefinition.Default)
	consulStore.StoreConsulKeyAsString(propPrefix+"/required", fmt.Sprint(propDefinition.Required))
}

// storeAttributeDefinition stores an attribute definition
func storeAttributeDefinition(ctx context.Context, attrPrefix, attrName string, attrDefinition tosca.AttributeDefinition) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	consulStore.StoreConsulKeyAsString(attrPrefix+"/name", attrName)
	consulStore.StoreConsulKeyAsString(attrPrefix+"/description", attrDefinition.Description)
	consulStore.StoreConsulKeyAsString(attrPrefix+"/type", attrDefinition.Type)
	consulStore.StoreConsulKeyAsString(attrPrefix+"/default", attrDefinition.Default.String())
	consulStore.StoreConsulKeyAsString(attrPrefix+"/status", attrDefinition.Status)
}

// storeTypes stores topology types
func storeTypes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	typesPrefix := path.Join(topologyPrefix, "types")
	for nodeTypeName, nodeType := range topology.NodeTypes {
		nodeTypePrefix := typesPrefix + "/" + nodeTypeName
		consulStore.StoreConsulKeyAsString(nodeTypePrefix+"/name", nodeTypeName)
		consulStore.StoreConsulKeyAsString(nodeTypePrefix+"/derived_from", nodeType.DerivedFrom)
		consulStore.StoreConsulKeyAsString(nodeTypePrefix+"/description", nodeType.Description)
		consulStore.StoreConsulKeyAsString(nodeTypePrefix+"/metatype", "Node")
		consulStore.StoreConsulKeyAsString(nodeTypePrefix+"/version", nodeType.Version)
		propertiesPrefix := nodeTypePrefix + "/properties"
		for propName, propDefinition := range nodeType.Properties {
			propPrefix := propertiesPrefix + "/" + propName
			storePropertyDefinition(ctx, propPrefix, propName, propDefinition)
		}
		attributesPrefix := nodeTypePrefix + "/attributes"
		for attrName, attrDefinition := range nodeType.Attributes {
			attrPrefix := attributesPrefix + "/" + attrName
			storeAttributeDefinition(ctx, attrPrefix, attrName, attrDefinition)
			if attrDefinition.Default.Expression != nil && attrDefinition.Default.Expression.Value == "get_operation_output" {
				interfaceName := url.QueryEscape(attrDefinition.Default.Expression.Children()[1].Value)
				operationName := url.QueryEscape(attrDefinition.Default.Expression.Children()[2].Value)
				outputVariableName := url.QueryEscape(attrDefinition.Default.Expression.Children()[3].Value)
				consulStore.StoreConsulKeyAsString(nodeTypePrefix+"/output/"+interfaceName+"/"+operationName+"/"+outputVariableName, outputVariableName)
			}
		}

		requirementsPrefix := nodeTypePrefix + "/requirements"
		for reqIndex, reqMap := range nodeType.Requirements {
			for reqName, reqDefinition := range reqMap {
				reqPrefix := requirementsPrefix + "/" + strconv.Itoa(reqIndex)
				consulStore.StoreConsulKeyAsString(reqPrefix+"/name", reqName)
				consulStore.StoreConsulKeyAsString(reqPrefix+"/node", reqDefinition.Node)
				consulStore.StoreConsulKeyAsString(reqPrefix+"/occurences/lower_bound", strconv.FormatUint(reqDefinition.Occurrences.LowerBound, 10))
				consulStore.StoreConsulKeyAsString(reqPrefix+"/occurences/upper_bound", strconv.FormatUint(reqDefinition.Occurrences.UpperBound, 10))
				consulStore.StoreConsulKeyAsString(reqPrefix+"/relationship", reqDefinition.Relationship)
				consulStore.StoreConsulKeyAsString(reqPrefix+"/capability", reqDefinition.Capability)
			}
		}
		capabilitiesPrefix := nodeTypePrefix + "/capabilities"
		for capName, capability := range nodeType.Capabilities {
			capabilityPrefix := capabilitiesPrefix + "/" + capName

			consulStore.StoreConsulKeyAsString(capabilityPrefix+"/name", capName)
			consulStore.StoreConsulKeyAsString(capabilityPrefix+"/type", capability.Type)
			consulStore.StoreConsulKeyAsString(capabilityPrefix+"/description", capability.Description)
			consulStore.StoreConsulKeyAsString(capabilityPrefix+"/occurences/lower_bound", strconv.FormatUint(capability.Occurrences.LowerBound, 10))
			consulStore.StoreConsulKeyAsString(capabilityPrefix+"/occurences/upper_bound", strconv.FormatUint(capability.Occurrences.UpperBound, 10))
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
			for intName, intDef := range intMap {
				intPrefix := path.Join(interfacesPrefix, intTypeName, intName)
				consulStore.StoreConsulKeyAsString(intPrefix+"/name", intName)
				consulStore.StoreConsulKeyAsString(intPrefix+"/description", intDef.Description)

				for inputName, inputDef := range intDef.Inputs {
					inputPrefix := path.Join(intPrefix, "inputs", inputName)
					consulStore.StoreConsulKeyAsString(inputPrefix+"/name", inputName)
					isValueAssignement := false
					isPropertyDefinition := false
					if inputDef.ValueAssign != nil {
						consulStore.StoreConsulKeyAsString(inputPrefix+"/expression", inputDef.ValueAssign.String())
						isValueAssignement = true
					}
					if inputDef.PropDef != nil {
						consulStore.StoreConsulKeyAsString(inputPrefix+"/type", inputDef.PropDef.Type)
						consulStore.StoreConsulKeyAsString(inputPrefix+"/default", inputDef.PropDef.Default)
						consulStore.StoreConsulKeyAsString(inputPrefix+"/description", inputDef.PropDef.Description)
						consulStore.StoreConsulKeyAsString(inputPrefix+"/status", inputDef.PropDef.Status)
						consulStore.StoreConsulKeyAsString(inputPrefix+"/required", strconv.FormatBool(inputDef.PropDef.Required))
						isPropertyDefinition = true
					}

					consulStore.StoreConsulKeyAsString(inputPrefix+"/is_value_assignment", strconv.FormatBool(isValueAssignement))
					consulStore.StoreConsulKeyAsString(inputPrefix+"/is_property_definition", strconv.FormatBool(isPropertyDefinition))
				}
				consulStore.StoreConsulKeyAsString(intPrefix+"/implementation/primary", path.Join(importPath, intDef.Implementation.Primary))
				consulStore.StoreConsulKeyAsString(intPrefix+"/implementation/dependencies", strings.Join(intDef.Implementation.Dependencies, ","))
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
}

// storeRelationshipTypes stores topology relationships types
func storeRelationshipTypes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	for relationName, relationType := range topology.RelationshipTypes {
		relationTypePrefix := path.Join(topologyPrefix, "types", relationName)
		consulStore.StoreConsulKeyAsString(relationTypePrefix+"/name", relationName)
		consulStore.StoreConsulKeyAsString(relationTypePrefix+"/derived_from", relationType.DerivedFrom)
		consulStore.StoreConsulKeyAsString(relationTypePrefix+"/description", relationType.Description)
		consulStore.StoreConsulKeyAsString(relationTypePrefix+"/version", relationType.Version)
		consulStore.StoreConsulKeyAsString(relationTypePrefix+"/metatype", "Relationship")
		propertiesPrefix := relationTypePrefix + "/properties"
		for propName, propDefinition := range relationType.Properties {
			propPrefix := propertiesPrefix + "/" + propName
			storePropertyDefinition(ctx, propPrefix, propName, propDefinition)
		}
		attributesPrefix := relationTypePrefix + "/attributes"
		for attrName, attrDefinition := range relationType.Attributes {
			attrPrefix := attributesPrefix + "/" + attrName
			storeAttributeDefinition(ctx, attrPrefix, attrName, attrDefinition)
			if attrDefinition.Default.Expression != nil && attrDefinition.Default.Expression.Value == "get_operation_output" {
				interfaceName := url.QueryEscape(attrDefinition.Default.Expression.Children()[1].Value)
				operationName := url.QueryEscape(attrDefinition.Default.Expression.Children()[2].Value)
				outputVariableName := url.QueryEscape(attrDefinition.Default.Expression.Children()[3].Value)
				consulStore.StoreConsulKeyAsString(relationTypePrefix+"/output/"+interfaceName+"/"+operationName+"/"+outputVariableName, outputVariableName)
			}
		}

		interfacesPrefix := relationTypePrefix + "/interfaces"
		for intTypeName, intMap := range relationType.Interfaces {
			for intName, intDef := range intMap {
				intPrefix := path.Join(interfacesPrefix, intTypeName, intName)
				consulStore.StoreConsulKeyAsString(intPrefix+"/name", intName)
				consulStore.StoreConsulKeyAsString(intPrefix+"/description", intDef.Description)

				for inputName, inputDef := range intDef.Inputs {
					inputPrefix := path.Join(intPrefix, "inputs", inputName)
					consulStore.StoreConsulKeyAsString(inputPrefix+"/name", inputName)
					isValueAssignement := false
					isPropertyDefinition := false
					if inputDef.ValueAssign != nil {
						consulStore.StoreConsulKeyAsString(inputPrefix+"/expression", inputDef.ValueAssign.String())
						isValueAssignement = true
					}
					if inputDef.PropDef != nil {
						consulStore.StoreConsulKeyAsString(inputPrefix+"/type", inputDef.PropDef.Type)
						consulStore.StoreConsulKeyAsString(inputPrefix+"/default", inputDef.PropDef.Default)
						consulStore.StoreConsulKeyAsString(inputPrefix+"/description", inputDef.PropDef.Description)
						consulStore.StoreConsulKeyAsString(inputPrefix+"/status", inputDef.PropDef.Status)
						consulStore.StoreConsulKeyAsString(inputPrefix+"/required", strconv.FormatBool(inputDef.PropDef.Required))
						isPropertyDefinition = true
					}

					consulStore.StoreConsulKeyAsString(inputPrefix+"/is_value_assignment", strconv.FormatBool(isValueAssignement))
					consulStore.StoreConsulKeyAsString(inputPrefix+"/is_property_definition", strconv.FormatBool(isPropertyDefinition))
				}
				consulStore.StoreConsulKeyAsString(intPrefix+"/implementation/primary", path.Join(importPath, intDef.Implementation.Primary))
				consulStore.StoreConsulKeyAsString(intPrefix+"/implementation/dependencies", strings.Join(intDef.Implementation.Dependencies, ","))
			}
		}

		artifactsPrefix := relationTypePrefix + "/artifacts"
		for artName, artDef := range relationType.Artifacts {
			artPrefix := artifactsPrefix + "/" + artName
			consulStore.StoreConsulKeyAsString(artPrefix+"/name", artName)
			consulStore.StoreConsulKeyAsString(artPrefix+"/metatype", "artifact")
			consulStore.StoreConsulKeyAsString(artPrefix+"/description", artDef.Description)
			consulStore.StoreConsulKeyAsString(artPrefix+"/file", path.Join(importPath, artDef.File))
			consulStore.StoreConsulKeyAsString(artPrefix+"/type", artDef.Type)
			consulStore.StoreConsulKeyAsString(artPrefix+"/repository", artDef.Repository)
			consulStore.StoreConsulKeyAsString(artPrefix+"/deploy_path", artDef.DeployPath)
		}

		consulStore.StoreConsulKeyAsString(relationTypePrefix+"/valid_target_type", strings.Join(relationType.ValidTargetTypes, ", "))

	}
}

// storeCapabilityTypes stores topology capabilities types
func storeCapabilityTypes(ctx context.Context, topology tosca.Topology, topologyPrefix string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	for capabilityTypeName, capabilityType := range topology.CapabilityTypes {
		capabilityTypePrefix := path.Join(topologyPrefix, "types", capabilityTypeName)
		consulStore.StoreConsulKeyAsString(capabilityTypePrefix+"/name", capabilityTypeName)
		consulStore.StoreConsulKeyAsString(capabilityTypePrefix+"/derived_from", capabilityType.DerivedFrom)
		consulStore.StoreConsulKeyAsString(capabilityTypePrefix+"/description", capabilityType.Description)
		consulStore.StoreConsulKeyAsString(capabilityTypePrefix+"/version", capabilityType.Version)
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

// storeWorkflows stores topology workflows
func storeWorkflows(ctx context.Context, topology tosca.Topology, deploymentId string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	workflowsPrefix := path.Join(DeploymentKVPrefix, deploymentId, "workflows")
	for wfName, workflow := range topology.TopologyTemplate.Workflows {
		workflowPrefix := workflowsPrefix + "/" + url.QueryEscape(wfName)
		for stepName, step := range workflow.Steps {
			stepPrefix := workflowPrefix + "/steps/" + url.QueryEscape(stepName)
			consulStore.StoreConsulKeyAsString(stepPrefix+"/node", step.Node)
			if step.Activity.CallOperation != "" {
				consulStore.StoreConsulKeyAsString(stepPrefix+"/activity/operation", step.Activity.CallOperation)
			}
			if step.Activity.Delegate != "" {
				consulStore.StoreConsulKeyAsString(stepPrefix+"/activity/delegate", step.Activity.Delegate)
			}
			if step.Activity.SetState != "" {
				consulStore.StoreConsulKeyAsString(stepPrefix+"/activity/set-state", step.Activity.SetState)
			}
			for _, next := range step.OnSuccess {
				consulStore.StoreConsulKeyAsString(fmt.Sprintf("%s/next/%s", stepPrefix, url.QueryEscape(next)), "")
			}
		}
	}
}

// createInstancesForNodes walk through the topology nodes an for each of those hosted on a Scalable node stores the number of
// required instances and sets the instance's status to INITIAL
func createInstancesForNodes(ctx context.Context, kv *api.KV, deploymentId string) error {
	_, errGroup, consulStore := consulutil.WithContext(ctx)

	depPath := path.Join(DeploymentKVPrefix, deploymentId)
	nodesPath := path.Join(depPath, "topology", "nodes")
	instancesPath := path.Join(depPath, "topology", "instances")

	nodes, _, err := kv.Keys(nodesPath+"/", "/", nil)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		node = path.Base(node)
		scalable, nbInstances, err := GetNbInstancesForNode(kv, deploymentId, node)
		if err != nil {
			return err
		}
		if scalable {
			consulStore.StoreConsulKeyAsString(path.Join(nodesPath, node, "nbInstances"), strconv.FormatUint(uint64(nbInstances), 10))
			for i := uint32(0); i < nbInstances; i++ {
				consulStore.StoreConsulKeyAsString(path.Join(instancesPath, node, strconv.FormatUint(uint64(i), 10), "status"), INITIAL.String())
			}
		}
	}

	return errGroup.Wait()
}
