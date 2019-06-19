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

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/ystia/yorc/v4/deployments/internal"
	"github.com/ystia/yorc/v4/deployments/store"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/registry"
	"github.com/ystia/yorc/v4/tosca"
)

var reg = registry.GetRegistry()

// StoreDeploymentDefinition takes a defPath and parse it as a tosca.Topology then it store it in consul under
// consulutil.DeploymentKVPrefix/deploymentID
func StoreDeploymentDefinition(ctx context.Context, kv *api.KV, deploymentID string, defPath string) error {
	if err := SetDeploymentStatus(ctx, kv, deploymentID, INITIAL); err != nil {
		return handleDeploymentStatus(ctx, kv, deploymentID, err)
	}

	topology := tosca.Topology{}
	definition, err := os.Open(defPath)
	if err != nil {
		return handleDeploymentStatus(ctx, kv, deploymentID, errors.Wrapf(err, "Failed to open definition file %q", defPath))
	}
	defBytes, err := ioutil.ReadAll(definition)
	if err != nil {
		return handleDeploymentStatus(ctx, kv, deploymentID, errors.Wrapf(err, "Failed to open definition file %q", defPath))
	}

	err = yaml.Unmarshal(defBytes, &topology)
	if err != nil {
		return handleDeploymentStatus(ctx, kv, deploymentID, errors.Wrapf(err, "Failed to unmarshal yaml definition for file %q", defPath))
	}

	consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "status"), fmt.Sprint(INITIAL))

	err = store.Deployment(ctx, topology, deploymentID, filepath.Dir(defPath))
	if err != nil {
		return handleDeploymentStatus(ctx, kv, deploymentID, errors.Wrapf(err, "Failed to store TOSCA Definition for deployment with id %q, (file path %q)", deploymentID, defPath))
	}
	err = registerImplementationTypes(ctx, kv, deploymentID)
	if err != nil {
		return handleDeploymentStatus(ctx, kv, deploymentID, err)
	}

	return handleDeploymentStatus(ctx, kv, deploymentID, enhanceNodes(ctx, kv, deploymentID))
}

func handleDeploymentStatus(ctx context.Context, kv *api.KV, deploymentID string, err error) error {
	if err != nil {
		SetDeploymentStatus(ctx, kv, deploymentID, DEPLOYMENT_FAILED)
	}
	return err
}

// createInstancesForNode checks if the given node is hosted on a Scalable node, stores the number of required instances and sets the instance's status to INITIAL
func createInstancesForNode(ctx context.Context, consulStore consulutil.ConsulStore, kv *api.KV, deploymentID, nodeName string) error {
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
			err = createInstancesForNode(ctxStore, consulStore, kv, deploymentID, nodeName)
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

	err = enhanceWorkflows(consulStore, kv, deploymentID)
	if err != nil {
		return err
	}

	err = enhanceAttributes(kv, deploymentID, nodes)
	if err != nil {
		return err
	}
	return errGroup.Wait()
}

// In this function we iterate over all node to know which node need to have a HOST output and search for this HOST and tell him to export this output
func fixGetOperationOutputForHost(ctx context.Context, kv *api.KV, deploymentID, nodeName string) error {
	nodeType, err := GetNodeType(kv, deploymentID, nodeName)
	if nodeType != "" && err == nil {
		typePath, err := locateTypePath(kv, deploymentID, nodeType)
		if err != nil {
			return err
		}
		interfacesPrefix := path.Join(typePath, "interfaces")
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
						hostedTypePath, err := locateTypePath(kv, deploymentID, hostedNodeType)
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
		if relationshipType == "" {
			continue
		}
		relTypePath, err := locateTypePath(kv, deploymentID, relationshipType)
		if err != nil {
			return err
		}
		relationshipPrefix := path.Join(relTypePath, "interfaces")
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
						typePath, err := locateTypePath(kv, deploymentID, nodeType)
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
			_, errgroup, consulStore := consulutil.WithContext(ctx)
			internal.StoreRequirementAssignment(consulStore, req, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", computeNodeName, "requirements", fmt.Sprint(newReqID)), "local_storage")

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
		internal.StoreWorkflow(consulStore, deploymentID, "run", wf)
	}
	return nil
}

// enhanceAttributes walk through the topology nodes an for each of them if needed it creates instances attributes notifications
// to allow resolving any attribute when one is updated
func enhanceAttributes(kv *api.KV, deploymentID string, nodes []string) error {
	for _, nodeName := range nodes {
		// retrieve all node attributes
		attributes, err := GetNodeAttributesNames(kv, deploymentID, nodeName)
		if err != nil {
			return err
		}

		// retrieve all node instances
		instances, err := GetNodeInstancesIds(kv, deploymentID, nodeName)
		if err != nil {
			return err
		}

		// 1. Add attribute notifications
		// 2. Resolve attributes and publish default values when not nil or empty
		for _, instanceName := range instances {
			for _, attribute := range attributes {
				err := addAttributeNotifications(kv, deploymentID, nodeName, instanceName, attribute)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
