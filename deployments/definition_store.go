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
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/ystia/yorc/v4/deployments/store"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/storage"
	storageTypes "github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"
)

// StoreDeploymentDefinition takes a defPath and parse it as a tosca.Topology then it store it in consul under
// consulutil.DeploymentKVPrefix/deploymentID
func StoreDeploymentDefinition(ctx context.Context, deploymentID string, defPath string) error {
	if err := SetDeploymentStatus(ctx, deploymentID, INITIAL); err != nil {
		return handleDeploymentStatus(ctx, deploymentID, err)
	}

	topology := tosca.Topology{}
	definition, err := os.Open(defPath)
	if err != nil {
		return handleDeploymentStatus(ctx, deploymentID, errors.Wrapf(err, "Failed to open definition file %q", defPath))
	}
	defBytes, err := ioutil.ReadAll(definition)
	if err != nil {
		return handleDeploymentStatus(ctx, deploymentID, errors.Wrapf(err, "Failed to open definition file %q", defPath))
	}

	err = yaml.Unmarshal(defBytes, &topology)
	if err != nil {
		return handleDeploymentStatus(ctx, deploymentID, errors.Wrapf(err, "Failed to unmarshal yaml definition for file %q", defPath))
	}

	consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "status"), fmt.Sprint(INITIAL))

	err = store.Deployment(ctx, topology, deploymentID, filepath.Dir(defPath))
	if err != nil {
		return handleDeploymentStatus(ctx, deploymentID, errors.Wrapf(err, "Failed to store TOSCA Definition for deployment with id %q, (file path %q)", deploymentID, defPath))
	}
	err = registerImplementationTypes(ctx, deploymentID)
	if err != nil {
		return handleDeploymentStatus(ctx, deploymentID, err)
	}

	// Enhance nodes
	nodes, err := GetNodes(ctx, deploymentID)
	if err != nil {
		return err
	}
	return handleDeploymentStatus(ctx, deploymentID, enhanceNodes(ctx, deploymentID, nodes))
}

func handleDeploymentStatus(ctx context.Context, deploymentID string, err error) error {
	if err != nil {
		SetDeploymentStatus(ctx, deploymentID, DEPLOYMENT_FAILED)
	}
	return err
}

// createInstancesForNode checks if the given node is hosted on a Scalable node, stores the number of required instances and sets the instance's status to INITIAL
func createInstancesForNode(ctx context.Context, consulStore consulutil.ConsulStore, deploymentID, nodeName string) error {
	nbInstances, err := GetDefaultNbInstancesForNode(ctx, deploymentID, nodeName)
	if err != nil {
		return err
	}
	createNodeInstances(consulStore, nbInstances, deploymentID, nodeName)

	// Check for FIPConnectivity capabilities
	is, capabilityNodeName, err := HasAnyRequirementCapability(ctx, deploymentID, nodeName, "network", "yorc.capabilities.openstack.FIPConnectivity")
	if err != nil {
		return err
	}
	if is {
		createNodeInstances(consulStore, nbInstances, deploymentID, capabilityNodeName)
	}

	// Check for Assignable capabilities
	is, capabilityNodeName, err = HasAnyRequirementCapability(ctx, deploymentID, nodeName, "assignment", "yorc.capabilities.Assignable")
	if err != nil {
		return err
	}
	if is {
		createNodeInstances(consulStore, nbInstances, deploymentID, capabilityNodeName)
	}

	bs, bsNames, err := checkBlockStorage(ctx, deploymentID, nodeName)
	if err != nil {
		return err
	}

	if bs {
		for _, name := range bsNames {
			createNodeInstances(consulStore, nbInstances, deploymentID, name)
		}

	}
	return nil
}

func registerImplementationTypes(ctx context.Context, deploymentID string) error {
	// We use synchronous communication with consul here to allow to check for duplicates
	types, err := GetTypes(ctx, deploymentID)
	if err != nil {
		return err
	}
	extensionsMap := make(map[string]string)
	for _, t := range types {
		isImpl, err := IsTypeDerivedFrom(ctx, deploymentID, t, "tosca.artifacts.Implementation")
		if err != nil {
			if IsTypeMissingError(err) {
				// Bypassing this error it may happen in case of an used type let's trust Alien
				events.SimpleLogEntry(events.LogLevelWARN, deploymentID).RegisterAsString(fmt.Sprintf("[WARNING] %s", err))
				continue
			}
			return err
		}
		if isImpl {
			extensions, err := GetArtifactTypeExtensions(ctx, deploymentID, t)
			if err != nil {
				return err
			}

			for _, ext := range extensions {
				ext = strings.ToLower(ext)
				extensionsMap[ext] = t
			}
		}
	}

	if len(extensionsMap) > 0 {
		return storage.GetStore(storageTypes.StoreTypeDeployment).Set(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", implementationArtifactsExtensionsPath), extensionsMap)
	}

	return nil
}

// EnhanceNodes walk through the provided nodes an for each of them if needed it creates the instances and fix alien BlockStorage declaration
func enhanceNodes(ctx context.Context, deploymentID string, nodes []string) error {
	ctxStore, errGroup, consulStore := consulutil.WithContext(ctx)
	computes := make([]string, 0)
	for _, nodeName := range nodes {
		isCompute, err := createInstanceAndFixModel(ctxStore, consulStore, deploymentID, nodeName)
		if err != nil {
			return err
		}
		if isCompute {
			computes = append(computes, nodeName)
		}
	}
	err := createMissingBlockStorageForNodes(ctx, consulStore, deploymentID, computes)
	if err != nil {
		return err
	}
	err = errGroup.Wait()
	if err != nil {
		return err
	}

	_, errGroup, consulStore = consulutil.WithContext(ctx)
	for _, nodeName := range nodes {
		err = createRelationshipInstances(ctx, consulStore, deploymentID, nodeName)
		if err != nil {
			return err
		}
	}

	err = enhanceWorkflows(ctx, consulStore, deploymentID)
	if err != nil {
		return err
	}

	err = enhanceAttributes(ctx, deploymentID, nodes)
	if err != nil {
		return err
	}
	return errGroup.Wait()
}

func createInstanceAndFixModel(ctx context.Context, consulStore consulutil.ConsulStore, deploymentID string, nodeName string) (bool, error) {

	var isCompute bool
	err := fixGetOperationOutputForRelationship(ctx, deploymentID, nodeName)
	if err != nil {
		return isCompute, err
	}
	err = fixGetOperationOutputForHost(ctx, deploymentID, nodeName)
	if err != nil {
		return isCompute, err
	}

	substitutable, err := isSubstitutableNode(ctx, deploymentID, nodeName)
	if err != nil {
		return isCompute, err
	}
	if !substitutable {
		err = createInstancesForNode(ctx, consulStore, deploymentID, nodeName)
		if err != nil {
			return isCompute, err
		}
		err = fixAlienBlockStorages(ctx, deploymentID, nodeName)
		if err != nil {
			return isCompute, err
		}
		isCompute, err = IsNodeDerivedFrom(ctx, deploymentID, nodeName, "tosca.nodes.Compute")
	}

	return isCompute, err
}

// In this function we iterate over all node to know which node need to have a HOST output and search for this HOST and tell him to export this output
func fixGetOperationOutputForHost(ctx context.Context, deploymentID, nodeName string) error {
	nodeType, err := GetNodeType(ctx, deploymentID, nodeName)
	if nodeType != "" && err == nil {
		typePath, err := locateTypePath(deploymentID, nodeType)
		if err != nil {
			return err
		}
		interfacesPrefix := path.Join(typePath, "interfaces")
		interfacesNamesPaths, err := consulutil.GetKeys(interfacesPrefix)
		if err != nil {
			return err
		}
		for _, interfaceNamePath := range interfacesNamesPaths {
			operationsPaths, err := consulutil.GetKeys(interfaceNamePath)
			if err != nil {
				return err
			}
			for _, operationPath := range operationsPaths {
				outputsPrefix := path.Join(operationPath, "outputs", "HOST")
				outputsNamesPaths, err := consulutil.GetKeys(outputsPrefix)
				if err != nil {
					return err
				}
				if outputsNamesPaths == nil || len(outputsNamesPaths) == 0 {
					continue
				}
				for _, outputNamePath := range outputsNamesPaths {
					hostedOn, err := GetHostedOnNode(ctx, deploymentID, nodeName)
					if err != nil {
						return nil
					} else if hostedOn == "" {
						return errors.New("Fail to get the hostedOn to fix the output")
					}
					if hostedNodeType, err := GetNodeType(ctx, deploymentID, hostedOn); hostedNodeType != "" && err == nil {
						hostedTypePath, err := locateTypePath(deploymentID, hostedNodeType)
						if err != nil {
							return err
						}
						consulutil.StoreConsulKeyAsString(path.Join(hostedTypePath, "interfaces", path.Base(interfaceNamePath), path.Base(operationPath), "outputs", "SELF", path.Base(outputNamePath), "expression"), "get_operation_output: [SELF,"+path.Base(interfaceNamePath)+","+path.Base(operationPath)+","+path.Base(outputNamePath)+"]")
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
func fixGetOperationOutputForRelationship(ctx context.Context, deploymentID, nodeName string) error {
	reqPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName, "requirements")
	reqName, err := consulutil.GetKeys(reqPath)
	if err != nil {
		return err
	}
	for _, reqKeyIndex := range reqName {
		relationshipType, err := GetRelationshipForRequirement(ctx, deploymentID, nodeName, path.Base(reqKeyIndex))
		if err != nil {
			return err
		}
		if relationshipType == "" {
			continue
		}
		relTypePath, err := locateTypePath(deploymentID, relationshipType)
		if err != nil {
			return err
		}
		relationshipPrefix := path.Join(relTypePath, "interfaces")
		interfaceNamesPaths, err := consulutil.GetKeys(relationshipPrefix)
		if err != nil {
			return err
		}
		for _, interfaceNamePath := range interfaceNamesPaths {
			operationsNamesPaths, err := consulutil.GetKeys(interfaceNamePath + "/")
			if err != nil {
				return err
			}
			for _, operationNamePath := range operationsNamesPaths {
				modEntityNamesPaths, err := consulutil.GetKeys(operationNamePath + "/outputs")
				if err != nil {
					return err
				}
				for _, modEntityNamePath := range modEntityNamesPaths {
					outputsNamesPaths, err := consulutil.GetKeys(modEntityNamePath)
					if err != nil {
						return err
					}
					for _, outputNamePath := range outputsNamesPaths {
						if path.Base(modEntityNamePath) != "SOURCE" || path.Base(modEntityNamePath) != "TARGET" {
							continue
						}
						var nodeType string
						if path.Base(modEntityNamePath) == "SOURCE" {
							nodeType, _ = GetNodeType(ctx, deploymentID, nodeName)
						} else if path.Base(modEntityNamePath) == "TARGET" {
							targetNode, err := GetTargetNodeForRequirement(ctx, deploymentID, nodeName, reqKeyIndex)
							if err != nil {
								return err
							}
							nodeType, _ = GetNodeType(ctx, deploymentID, targetNode)
						}
						typePath, err := locateTypePath(deploymentID, nodeType)
						if err != nil {
							return err
						}
						consulutil.StoreConsulKeyAsString(path.Join(typePath, "interfaces", path.Base(interfaceNamePath), path.Base(operationNamePath), "outputs", "SELF", path.Base(outputNamePath), "expression"), "get_operation_output: [SELF,"+path.Base(interfaceNamePath)+","+path.Base(operationNamePath)+","+path.Base(outputNamePath)+"]")
					}
				}
			}
		}

	}
	return nil
}

// fixAlienBlockStorages rewrites the relationship between a BlockStorage and a Compute to match the TOSCA specification
func fixAlienBlockStorages(ctx context.Context, deploymentID, nodeName string) error {
	isBS, err := IsNodeDerivedFrom(ctx, deploymentID, nodeName, "tosca.nodes.BlockStorage")
	if err != nil {
		return err
	}
	if isBS {
		attachReqs, err := GetRequirementsByTypeForNode(ctx, deploymentID, nodeName, "attachment")
		if err != nil {
			return err
		}
		for _, attachReq := range attachReqs {
			req := attachReq.RequirementAssignment
			// Reverse the target node
			computeNodeName := req.Node
			req.Node = nodeName
			device, err := GetNodePropertyValue(ctx, deploymentID, nodeName, "device")
			if err != nil {
				return errors.Wrapf(err, "Failed to fix Alien-specific BlockStorage %q", nodeName)
			}
			if device != nil {
				va := &tosca.ValueAssignment{}
				if device.RawString() != "" {
					err = yaml.Unmarshal([]byte(device.RawString()), &va)
					if err != nil {
						return errors.Wrapf(err, "Failed to fix Alien-specific BlockStorage %q, failed to parse device property", nodeName)
					}
				}
				// Add device requirement property
				if req.RelationshipProps == nil {
					req.RelationshipProps = make(map[string]*tosca.ValueAssignment)
				}
				req.RelationshipProps["device"] = va
			}

			// Update the compute node with new requirement
			node, err := getNodeTemplateStruct(ctx, deploymentID, nodeName)
			if err != nil {
				return err
			}
			nodePrefix := path.Join(consulutil.DeploymentKVPrefix, "topology", "nodes", computeNodeName)
			storage.GetStore(storageTypes.StoreTypeDeployment).Set(nodePrefix, node)
		}
	}

	return nil
}

/**
This function create a given number of floating IP instances
*/
func createNodeInstances(consulStore consulutil.ConsulStore, numberInstances uint32, deploymentID, nodeName string) {

	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)

	consulStore.StoreConsulKeyAsString(path.Join(nodePath, "nbInstances"), strconv.FormatUint(uint64(numberInstances), 10))

	for i := uint32(0); i < numberInstances; i++ {
		instanceName := strconv.FormatUint(uint64(i), 10)
		createNodeInstance(consulStore, deploymentID, nodeName, instanceName)
	}
}

// createInstancesForNodes checks if the given nodes are hosted on a Scalable node,
// stores the number of required instances and sets the instance's status to INITIAL
func createMissingBlockStorageForNodes(ctx context.Context, consulStore consulutil.ConsulStore, deploymentID string, nodeNames []string) error {

	for _, nodeName := range nodeNames {
		requirements, err := GetRequirementsByTypeForNode(ctx, deploymentID, nodeName, "local_storage")
		if err != nil {
			return err
		}

		nbInstances, err := GetNbInstancesForNode(ctx, deploymentID, nodeName)
		if err != nil {
			return err
		}

		var bsName []string

		for _, requirement := range requirements {
			if requirement.Capability != "" {
				bsName = append(bsName, requirement.Node)
			}
		}

		for _, name := range bsName {
			createNodeInstances(consulStore, nbInstances, deploymentID, name)
		}
	}

	return nil
}

/**
This function check if a nodes need a block storage, and return the name of BlockStorage node.
*/
func checkBlockStorage(ctx context.Context, deploymentID, nodeName string) (bool, []string, error) {
	requirements, err := GetRequirementsByTypeForNode(ctx, deploymentID, nodeName, "local_storage")
	if err != nil {
		return false, nil, err
	}

	var bsName []string
	for _, requirement := range requirements {
		if requirement.Capability != "" {
			bsName = append(bsName, requirement.Node)
		}

	}

	return true, bsName, nil
}

// enhanceAttributes walk through the topology nodes an for each of them if needed it creates instances attributes notifications
// to allow resolving any attribute when one is updated
func enhanceAttributes(ctx context.Context, deploymentID string, nodes []string) error {
	for _, nodeName := range nodes {
		// retrieve all node attributes
		attributes, err := GetNodeAttributesNames(ctx, deploymentID, nodeName)
		if err != nil {
			return err
		}

		// retrieve all node instances
		instances, err := GetNodeInstancesIds(ctx, deploymentID, nodeName)
		if err != nil {
			return err
		}

		// 1. Add attribute notifications
		// 2. Resolve attributes and publish default values when not nil or empty
		for _, instanceName := range instances {
			for _, attribute := range attributes {
				err := addAttributeNotifications(ctx, deploymentID, nodeName, instanceName, attribute)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
