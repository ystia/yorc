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

package ansible

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/gofrs/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/moby/moby/client"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/helper/executil"
	"github.com/ystia/yorc/v4/helper/provutil"
	"github.com/ystia/yorc/v4/helper/sshutil"
	"github.com/ystia/yorc/v4/helper/stringutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/prov/operations"
	"github.com/ystia/yorc/v4/tasks"
	"github.com/ystia/yorc/v4/tosca"
	"github.com/ystia/yorc/v4/tosca/types"
)

const taskContextOutput = "task_context"

const vaultPassScriptFormat = `#!/usr/bin/env %s

import os
print(os.environ['VAULT_PASSWORD'])
`

const ansibleConfigDefaultsHeader = "defaults"
const ansibleInventoryHostsHeader = "target_hosts"
const ansibleInventoryHostedHeader = "hosted_operations"
const ansibleInventoryHostsVarsHeader = ansibleInventoryHostsHeader + ":vars"

var ansibleDefaultConfig = map[string]map[string]string{
	ansibleConfigDefaultsHeader: map[string]string{
		"host_key_checking": "False",
		"timeout":           "30",
		"stdout_callback":   "yaml",
		"nocows":            "1",
	},
}

var ansibleFactCaching = map[string]string{
	"gathering":    "smart",
	"fact_caching": "jsonfile",
}

var ansibleInventoryConfig = map[string][]string{
	ansibleInventoryHostsVarsHeader: []string{
		"ansible_ssh_common_args=\"-o ConnectionAttempts=20\"",
		"ansible_python_interpreter=\"auto_silent\"",
	},
}

type ansibleRetriableError struct {
	root error
}

func (are ansibleRetriableError) Error() string {
	return are.root.Error()
}

// IsRetriable checks if a given error is an Ansible retriable error
func IsRetriable(err error) bool {
	_, ok := err.(ansibleRetriableError)
	return ok
}

// IsOperationNotImplemented checks if a given error is an error indicating that an operation is not implemented
func IsOperationNotImplemented(err error) bool {
	_, ok := err.(operationNotImplemented)
	return ok
}

type operationNotImplemented struct {
	msg string
}

func (oni operationNotImplemented) Error() string {
	return oni.msg
}

type hostConnection struct {
	host        string
	port        int
	user        string
	instanceID  string
	privateKeys map[string]*sshutil.PrivateKey
	password    string
	bastion     *sshutil.BastionHostConfig
	osType      string
}

type sshCredentials struct {
	user        string
	privateKeys map[string]*sshutil.PrivateKey
	password    string
}

type execution interface {
	resolveExecution(ctx context.Context) error
	execute(ctx context.Context, retry bool) error
}

type ansibleRunner interface {
	runAnsible(ctx context.Context, retry bool, currentInstance, ansibleRecipePath string) error
}

type executionCommon struct {
	cfg          config.Configuration
	ctx          context.Context
	deploymentID string
	taskID       string
	NodeName     string
	operation    prov.Operation
	NodeType     string
	// Description              string
	OperationRemoteBaseDir   string
	OperationRemotePath      string
	KeepOperationRemotePath  bool
	ArchiveArtifacts         string
	CacheFacts               bool
	EnvInputs                []*operations.EnvInput
	VarInputsNames           []string
	Primary                  string
	BasePrimary              string
	Dependencies             []string
	hosts                    map[string]*hostConnection
	OperationImplementation  *tosca.Implementation
	Artifacts                map[string]string
	OverlayPath              string
	Context                  map[string]string
	CapabilitiesCtx          map[string]*deployments.TOSCAValue
	Outputs                  map[string]string
	HaveOutput               bool
	isRelationshipTargetNode bool
	isPerInstanceOperation   bool
	isOrchestratorOperation  bool
	IsCustomCommand          bool
	relationshipType         string
	ansibleRunner            ansibleRunner
	sourceNodeInstances      []string
	targetNodeInstances      []string
	cli                      *client.Client
	containerID              string
	vaultToken               string
}

// Handling a command standard output and standard error
type outputHandler interface {
	start(cmd *exec.Cmd) error
	stop() error
}

func newExecution(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation, cli *client.Client) (execution, error) {

	u, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	execCommon := &executionCommon{
		cfg:          cfg,
		ctx:          ctx,
		deploymentID: deploymentID,
		NodeName:     nodeName,
		//KeepOperationRemotePath property is required to be public when resolving templates.
		KeepOperationRemotePath: cfg.Ansible.KeepOperationRemotePath,
		ArchiveArtifacts:        strconv.FormatBool(cfg.Ansible.ArchiveArtifacts),
		CacheFacts:              cfg.Ansible.CacheFacts,
		operation:               operation,
		VarInputsNames:          make([]string, 0),
		EnvInputs:               make([]*operations.EnvInput, 0),
		taskID:                  taskID,
		Outputs:                 make(map[string]string),
		cli:                     cli,
		vaultToken:              u.String(),
	}
	if err := execCommon.resolveOperation(ctx); err != nil {
		return nil, err
	}
	isBash, err := deployments.IsTypeDerivedFrom(ctx, deploymentID, operation.ImplementationArtifact, implementationArtifactBash)
	if err != nil {
		return nil, err
	}
	isPython, err := deployments.IsTypeDerivedFrom(ctx, deploymentID, operation.ImplementationArtifact, implementationArtifactPython)
	if err != nil {
		return nil, err
	}
	isAnsible, err := deployments.IsTypeDerivedFrom(ctx, deploymentID, operation.ImplementationArtifact, implementationArtifactAnsible)
	if err != nil {
		return nil, err
	}
	isAlienAnsible, err := deployments.IsTypeDerivedFrom(ctx, deploymentID, operation.ImplementationArtifact, "org.alien4cloud.artifacts.AnsiblePlaybook")
	if err != nil {
		return nil, err
	}
	var exec execution
	if isBash || isPython {
		execScript := &executionScript{executionCommon: execCommon, isPython: isPython}
		execCommon.ansibleRunner = execScript
		exec = execScript
	} else if isAnsible || isAlienAnsible {
		execAnsible := &executionAnsible{executionCommon: execCommon, isAlienAnsible: isAlienAnsible}
		execCommon.ansibleRunner = execAnsible
		exec = execAnsible
	} else {
		return nil, errors.Errorf("Unsupported artifact implementation for node: %q, operation: %s, primary implementation: %q", nodeName, operation.Name, execCommon.Primary)
	}

	return exec, exec.resolveExecution(ctx)
}

func (e *executionCommon) resolveOperation(ctx context.Context) error {
	var err error
	e.NodeType, err = deployments.GetNodeType(ctx, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}
	if e.operation.RelOp.IsRelationshipOperation {
		e.relationshipType, err = deployments.GetRelationshipForRequirement(ctx, e.deploymentID, e.NodeName, e.operation.RelOp.RequirementIndex)
		if err != nil {
			return err
		}
		err = e.resolveIsPerInstanceOperation(ctx, e.operation.Name)
		if err != nil {
			return err
		}
	} else if strings.Contains(e.operation.Name, "custom") {
		e.IsCustomCommand = true
	}

	operationNodeType := e.NodeType
	if e.operation.RelOp.IsRelationshipOperation {
		operationNodeType = e.relationshipType
	}
	e.OperationImplementation, err = deployments.GetOperationImplementation(ctx, e.deploymentID, e.operation.ImplementedInNodeTemplate, operationNodeType, e.operation.Name)
	if err != nil {
		return err
	}
	if e.OperationImplementation.Primary == "" {
		return operationNotImplemented{msg: fmt.Sprintf("primary implementation missing for operation %q of type %q in deployment %q is missing", e.operation.Name, e.NodeType, e.deploymentID)}
	}
	e.Primary = strings.TrimSpace(e.OperationImplementation.Primary)
	log.Debugf("Operation Definition: %+v, primary implementation: %q", e.OperationImplementation, e.Primary)
	e.BasePrimary = path.Base(e.Primary)

	if e.OperationImplementation.Dependencies != nil {
		e.Dependencies = e.OperationImplementation.Dependencies
	} else {
		e.Dependencies = make([]string, 0)
	}

	// if operation_host is not overridden by requirement, we retrieve operation/implementation definition info
	if e.operation.OperationHost == "" {
		if e.OperationImplementation.OperationHost != "" {
			e.operation.OperationHost = e.OperationImplementation.OperationHost
		}
	}

	e.isOrchestratorOperation = operations.IsOrchestratorHostOperation(e.operation)
	e.isRelationshipTargetNode = operations.IsRelationshipTargetNodeOperation(e.operation)
	return e.resolveInstances(ctx)
}

func (e *executionCommon) resolveInstances(ctx context.Context) error {
	var err error
	if e.operation.RelOp.IsRelationshipOperation {
		e.targetNodeInstances, err = tasks.GetInstances(ctx, e.taskID, e.deploymentID, e.operation.RelOp.TargetNodeName)
		if err != nil {
			return err
		}
	}
	e.sourceNodeInstances, err = tasks.GetInstances(ctx, e.taskID, e.deploymentID, e.NodeName)

	return err
}

func (e *executionCommon) resolveArtifacts(ctx context.Context) error {
	log.Debugf("Resolving artifacts")
	var err error
	if e.operation.RelOp.IsRelationshipOperation {
		// First get linked node artifacts
		if e.isRelationshipTargetNode {
			e.Artifacts, err = deployments.GetFileArtifactsForNode(ctx, e.deploymentID, e.operation.RelOp.TargetNodeName)
			if err != nil {
				return err
			}
		} else {
			e.Artifacts, err = deployments.GetFileArtifactsForNode(ctx, e.deploymentID, e.NodeName)
			if err != nil {
				return err
			}
		}
		// Then get relationship type artifacts
		var arts map[string]string
		arts, err = deployments.GetFileArtifactsForType(ctx, e.deploymentID, e.relationshipType)
		if err != nil {
			return err
		}
		for artName, art := range arts {
			e.Artifacts[artName] = art
		}
	} else {
		e.Artifacts, err = deployments.GetFileArtifactsForNode(ctx, e.deploymentID, e.NodeName)
		if err != nil {
			return err
		}
	}
	log.Debugf("Resolved artifacts: %v", e.Artifacts)
	return nil
}

func (e *executionCommon) setHostConnection(ctx context.Context, host, instanceID, capType string, conn *hostConnection) error {
	hasEndpoint, err := deployments.IsTypeDerivedFrom(ctx, e.deploymentID, capType, "yorc.capabilities.Endpoint.ProvisioningAdmin")
	if err != nil {
		return err
	}
	if hasEndpoint {
		credentialValue, err := deployments.GetInstanceCapabilityAttributeValue(ctx, e.deploymentID, host, instanceID, "endpoint", "credentials")
		if err != nil {
			return err
		}
		credentials := new(types.Credential)
		if credentialValue != nil && credentialValue.RawString() != "" {
			err = mapstructure.Decode(credentialValue.Value, credentials)
			if err != nil {
				return errors.Wrapf(err, "failed to decode credentials for node %q", host)
			}
		}
		if credentials.User != "" {
			conn.user = config.DefaultConfigTemplateResolver.ResolveValueWithTemplates("host.user", credentials.User).(string)
		} else {
			mess := fmt.Sprintf("[Warning] No user set for connection:%+v", conn)
			log.Printf(mess)
			events.WithContextOptionalFields(e.ctx).NewLogEntry(events.LogLevelWARN, e.deploymentID).RegisterAsString(mess)
		}
		if credentials.Token != "" {
			conn.password = config.DefaultConfigTemplateResolver.ResolveValueWithTemplates("host.password", credentials.Token).(string)
		}

		conn.privateKeys, err = sshutil.GetKeysFromCredentialsDataType(credentials)
		if err != nil {
			return err
		}

		port, err := deployments.GetInstanceCapabilityAttributeValue(ctx, e.deploymentID, host, instanceID, "endpoint", "port")
		if err != nil {
			return err
		}
		if port != nil && port.RawString() != "" {
			conn.port, err = strconv.Atoi(port.RawString())
			if err != nil {
				return errors.Wrapf(err, "Failed to convert port value:%q to int", port)
			}
		}
		// Need to get the os type as Windows target hosts need specific ansible settings
		osType, err := deployments.GetInstanceCapabilityAttributeValue(ctx, e.deploymentID, host, instanceID, "os", "type")
		if err != nil {
			return err
		}
		if osType != nil {
			conn.osType = strings.ToLower(osType.RawString())
		}
	}
	return nil
}

func (e *executionCommon) resolveHostsOrchestratorLocal(nodeName string, instances []string) error {
	e.hosts = make(map[string]*hostConnection, len(instances))
	for i := range instances {
		instanceName := operations.GetInstanceName(nodeName, instances[i])
		e.hosts[instanceName] = &hostConnection{host: instanceName, instanceID: instances[i]}
	}
	return nil
}

func (e *executionCommon) resolveHostsOnCompute(ctx context.Context, nodeName string, instances []string) error {
	hostedOnList := make([]string, 0)
	hostedOnList = append(hostedOnList, nodeName)
	parentHost, err := deployments.GetHostedOnNode(ctx, e.deploymentID, nodeName)
	if err != nil {
		return err
	}
	for parentHost != "" {
		hostedOnList = append(hostedOnList, parentHost)
		parentHost, err = deployments.GetHostedOnNode(ctx, e.deploymentID, parentHost)
		if err != nil {
			return err
		}
	}

	hosts := make(map[string]*hostConnection)
	var found bool
	for i := len(hostedOnList) - 1; i >= 0 && !found; i-- {
		host := hostedOnList[i]
		capType, err := deployments.GetNodeCapabilityType(ctx, e.deploymentID, host, "endpoint")
		if err != nil {
			return err
		}

		hasEndpoint, err := deployments.IsTypeDerivedFrom(ctx, e.deploymentID, capType, "tosca.capabilities.Endpoint")
		if err != nil {
			return err
		}
		if hasEndpoint {
			for _, instance := range instances {
				ipAddress, err := deployments.GetInstanceCapabilityAttributeValue(ctx, e.deploymentID, host, instance, "endpoint", "ip_address")
				if err != nil {
					return err
				}
				if ipAddress != nil && ipAddress.RawString() != "" {
					ipAddressStr := config.DefaultConfigTemplateResolver.ResolveValueWithTemplates("host.ip_address", ipAddress.RawString()).(string)
					instanceName := operations.GetInstanceName(nodeName, instance)
					hostConn := &hostConnection{host: ipAddressStr, instanceID: instance}
					hostConn.bastion, err = provutil.GetInstanceBastionHost(e.ctx, e.deploymentID, host)
					if err != nil {
						return err
					}
					err = e.setHostConnection(ctx, host, instance, capType, hostConn)
					if err != nil {
						mess := fmt.Sprintf("[ERROR] failed to set host connection with error: %+v", err)
						log.Debug(mess)
						events.WithContextOptionalFields(e.ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(mess)
						return err
					}
					hosts[instanceName] = hostConn
					found = true
				}
			}
		}
	}

	if len(hosts) == 0 {
		return errors.Errorf("Failed to resolve hosts for node %q", nodeName)
	}
	e.hosts = hosts
	return nil
}

func (e *executionCommon) resolveHosts(ctx context.Context, nodeName string) error {
	// Resolve hosts from the hostedOn hierarchy from bottom to top by finding the first node having a capability
	// named endpoint and derived from "tosca.capabilities.Endpoint"

	log.Debugf("Resolving hosts for node %q", nodeName)

	instances := e.sourceNodeInstances
	if e.isRelationshipTargetNode {
		instances = e.targetNodeInstances
	}

	if e.isOrchestratorOperation {
		return e.resolveHostsOrchestratorLocal(nodeName, instances)
	}
	return e.resolveHostsOnCompute(ctx, nodeName, instances)
}

func (e *executionCommon) resolveContext(ctx context.Context) error {
	execContext := make(map[string]string)

	newNode := provutil.SanitizeForShell(e.NodeName)
	if !e.operation.RelOp.IsRelationshipOperation {
		execContext["NODE"] = newNode
	}
	var instances []string
	if e.isRelationshipTargetNode {
		instances = e.targetNodeInstances
	} else {
		instances = e.sourceNodeInstances
	}

	names := make([]string, len(instances))
	for i := range instances {
		instanceName := operations.GetInstanceName(e.NodeName, instances[i])
		names[i] = instanceName
	}
	if !e.operation.RelOp.IsRelationshipOperation {
		e.VarInputsNames = append(e.VarInputsNames, "INSTANCE")
		execContext["INSTANCES"] = strings.Join(names, ",")
		if host, err := deployments.GetHostedOnNode(ctx, e.deploymentID, e.NodeName); err != nil {
			return err
		} else if host != "" {
			execContext["HOST"] = host
		}
	} else {

		if host, err := deployments.GetHostedOnNode(ctx, e.deploymentID, e.NodeName); err != nil {
			return err
		} else if host != "" {
			execContext["SOURCE_HOST"] = host
		}
		if host, err := deployments.GetHostedOnNode(ctx, e.deploymentID, e.operation.RelOp.TargetNodeName); err != nil {
			return err
		} else if host != "" {
			execContext["TARGET_HOST"] = host
		}
		execContext["SOURCE_NODE"] = newNode
		if e.isRelationshipTargetNode && !e.isPerInstanceOperation {
			execContext["SOURCE_INSTANCE"] = names[0]
		} else {
			e.VarInputsNames = append(e.VarInputsNames, "SOURCE_INSTANCE")
		}

		sourceNames := make([]string, len(e.sourceNodeInstances))
		for i := range e.sourceNodeInstances {
			sourceNames[i] = operations.GetInstanceName(e.NodeName, e.sourceNodeInstances[i])
		}
		execContext["SOURCE_INSTANCES"] = strings.Join(sourceNames, ",")
		execContext["TARGET_NODE"] = provutil.SanitizeForShell(e.operation.RelOp.TargetNodeName)

		targetNames := make([]string, len(e.targetNodeInstances))
		for i := range e.targetNodeInstances {
			targetNames[i] = operations.GetInstanceName(e.operation.RelOp.TargetNodeName, e.targetNodeInstances[i])
		}
		execContext["TARGET_INSTANCES"] = strings.Join(targetNames, ",")

		if !e.isRelationshipTargetNode && !e.isPerInstanceOperation {
			if len(targetNames) == 0 {
				log.Debugf("No target instance defined in context %+v", e)
			} else {
				execContext["TARGET_INSTANCE"] = targetNames[0]
			}
		} else {
			e.VarInputsNames = append(e.VarInputsNames, "TARGET_INSTANCE")
		}

	}

	execContext["DEPLOYMENT_ID"] = e.deploymentID

	var err error
	e.CapabilitiesCtx, err = operations.GetTargetCapabilityPropertiesAndAttributesValues(e.ctx, e.deploymentID, e.NodeName, e.operation)
	if err != nil {
		return err
	}

	e.Context = execContext
	return nil
}

func (e *executionCommon) resolveOperationOutputPath() error {
	//Here we get the modelable entity output of the operation
	log.Debugf("resolving operation outputs")
	mapOutputs, err := deployments.GetOperationOutputs(e.ctx, e.deploymentID, e.operation.ImplementedInNodeTemplate, e.operation.ImplementedInType, e.operation.Name)
	if err != nil {
		return err
	}
	if len(mapOutputs) == 0 {
		return nil
	}

	e.HaveOutput = true
	//We iterate over all entity of the output in this operation
	for outputName, outputValue := range mapOutputs {
		// either output can be associated to a get_operation_output value assignment or to an attribute mapping
		if outputValue.ValueAssign != nil {
			va := outputValue.ValueAssign
			if va.Type != tosca.ValueAssignmentFunction {
				return errors.Errorf("Output %q for operation %v is not a valid get_operation_output TOSCA function", outputName, e.operation)
			}
			oof := va.GetFunction()
			if oof.Operator != tosca.GetOperationOutputOperator {
				return errors.Errorf("Output %q for operation %v (%v) is not a valid get_operation_output TOSCA function", outputName, e.operation, oof)
			}
			targetContext := oof.Operands[0].String() == tosca.Target
			sourceContext := oof.Operands[0].String() == tosca.Source
			interfaceName := strings.ToLower(url.QueryEscape(oof.Operands[1].String()))
			operationName := strings.ToLower(url.QueryEscape(oof.Operands[2].String()))
			outputVariableName := url.QueryEscape(oof.Operands[3].String())
			nodeKeyword := oof.Operands[0].String()
			err = e.addOutputs(targetContext, sourceContext, nodeKeyword, interfaceName, operationName, outputVariableName)
			if err != nil {
				return err
			}
		} else if outputValue.AttributeMapping != nil && outputValue.AttributeMapping.Parameters != nil {
			parameters := outputValue.AttributeMapping.Parameters
			targetContext := strings.ToUpper(parameters[0]) == tosca.Target
			sourceContext := strings.ToUpper(parameters[0]) == tosca.Source
			err = e.addAttributeMappingOutputs(targetContext, sourceContext, outputName, parameters[1:])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *executionCommon) addOutputs(isTargetContext, isSourceContext bool, nodeKeyword, interfaceName, operationName, outputVariableName string) error {
	var instancesIds []string
	if isTargetContext {
		instancesIds = e.targetNodeInstances
	} else {
		instancesIds = e.sourceNodeInstances
	}
	if (isTargetContext || isSourceContext) && !e.operation.RelOp.IsRelationshipOperation {
		return errors.Errorf("Can't resolve an output in SOURCE or TARGET context without a relationship operation for output:%q", outputVariableName)
	}

	//For each instance of the node we create a new entry in the output map
	for _, instanceID := range instancesIds {
		// TODO(loicalbertin) This part should be refactored to store properly the instance ID
		// don't to it for now as it is for a quickfix
		b := instanceID
		if isTargetContext {
			e.Outputs[outputVariableName+"_"+fmt.Sprint(b)] = path.Join("instances", e.operation.RelOp.TargetNodeName, instanceID, "outputs", interfaceName, operationName, outputVariableName)
		} else {
			//If we are with an expression type {get_operation_output : [ SELF, ...]} in a relationship we store the result in the corresponding relationship instance
			if nodeKeyword == tosca.Self && e.operation.RelOp.IsRelationshipOperation {
				relationShipPrefix := path.Join("relationship_instances", e.NodeName, e.operation.RelOp.RequirementIndex, instanceID)
				e.Outputs[outputVariableName+"_"+fmt.Sprint(b)] = path.Join(relationShipPrefix, "outputs", interfaceName, operationName, outputVariableName)
			} else if nodeKeyword == tosca.Host {
				// In this case we continue because the parsing has change this type on {get_operation_output : [ SELF, ...]}  on the host node
				continue

			} else {
				//In all others case we simply save the result of the output on the instance directory of the node
				e.Outputs[outputVariableName+"_"+fmt.Sprint(b)] = path.Join("instances", e.NodeName, instanceID, "outputs", interfaceName, operationName, outputVariableName)
			}
		}
	}
	return nil
}

// attribute_mapping/<node_name>/<instance_name>/<attribute_name_or_capability_name>/<nested_attribute_name><nested_attribute_name><nested_attribute_name>...
func (e *executionCommon) addAttributeMappingOutputs(isTargetContext, isSourceContext bool, outputVariableName string, parameters []string) error {
	var instancesIds []string
	if isTargetContext {
		instancesIds = e.targetNodeInstances
	} else {
		instancesIds = e.sourceNodeInstances
	}
	if (isTargetContext || isSourceContext) && !e.operation.RelOp.IsRelationshipOperation {
		return errors.Errorf("Can't resolve an output in SOURCE or TARGET context without a relationship operation for output:%q", outputVariableName)
	}

	path.Join(parameters...)
	//For each instance of the node we create a new entry in the output map
	for _, instanceID := range instancesIds {
		b := instanceID
		if isTargetContext {
			e.Outputs[outputVariableName+"_"+fmt.Sprint(b)] = path.Join("attribute_mapping", e.operation.RelOp.TargetNodeName, instanceID, path.Join(parameters...))
		} else {
			e.Outputs[outputVariableName+"_"+fmt.Sprint(b)] = path.Join("attribute_mapping", e.NodeName, instanceID, path.Join(parameters...))
		}
	}
	return nil
}

func (e *executionCommon) addRunnablesSpecificInputsAndOutputs() error {
	opName := strings.ToLower(e.operation.Name)
	if !strings.HasPrefix(opName, tosca.RunnableInterfaceName) {
		return nil
	}
	for i, instanceID := range e.sourceNodeInstances {
		// TODO(loicalbertin) This part should be refactored to store properly the instance ID
		// don't to it for now as it is for a quickfix
		if opName == tosca.RunnableSubmitOperationName {
			e.Outputs["TOSCA_JOB_ID_"+fmt.Sprint(instanceID)] = taskContextOutput
			e.HaveOutput = true
		} else if opName == tosca.RunnableRunOperationName {
			e.Outputs["TOSCA_JOB_STATUS_"+fmt.Sprint(instanceID)] = taskContextOutput
			e.HaveOutput = true
		}

		// Now store jobID as input for run and cancel ops
		if opName == tosca.RunnableRunOperationName || opName == tosca.RunnableCancelOperationName {
			jobID, err := tasks.GetTaskData(e.taskID, e.NodeName+"-"+instanceID+"-TOSCA_JOB_ID")
			if err != nil {
				return errors.Wrap(err, "failed to retrieve job id for monitoring, this is likely that the submit operation does not properly export a TOSCA_JOB_ID output")
			}
			e.EnvInputs = append(e.EnvInputs, &operations.EnvInput{
				Name:         "TOSCA_JOB_ID",
				InstanceName: operations.GetInstanceName(e.NodeName, instanceID),
				Value:        jobID,
			})
			if i == 0 {
				e.VarInputsNames = append(e.VarInputsNames, "TOSCA_JOB_ID")
			}
		}
	}
	return nil
}

// resolveIsPerInstanceOperation sets e.isPerInstanceOperation to true if the given operationName contains one of the following patterns (case doesn't matter):
//
//	add_target, remove_target, add_source, remove_source, target_changed
//
// And in case of a relationship operation the relationship does not derive from "tosca.relationships.HostedOn" as it makes no sense till we scale at compute level
func (e *executionCommon) resolveIsPerInstanceOperation(ctx context.Context, operationName string) error {
	op := strings.ToLower(operationName)
	if strings.Contains(op, "add_target") || strings.Contains(op, "remove_target") || strings.Contains(op, "target_changed") || strings.Contains(op, "add_source") || strings.Contains(op, "remove_source") {
		// Do not call the call the operation several time for a "HostedOn" relationship (makes no sense till we scale at compute level)
		if hostedOn, err := deployments.IsTypeDerivedFrom(ctx, e.deploymentID, e.relationshipType, "tosca.relationships.HostedOn"); err != nil || hostedOn {
			e.isPerInstanceOperation = false
			return err
		}
		e.isPerInstanceOperation = true
		return nil
	}
	e.isPerInstanceOperation = false
	return nil
}

func (e *executionCommon) resolveInputs(ctx context.Context) error {
	var err error
	e.EnvInputs, e.VarInputsNames, err = operations.ResolveInputsWithInstances(ctx, e.deploymentID, e.NodeName, e.taskID, e.operation, e.sourceNodeInstances, e.targetNodeInstances)
	return err
}

func (e *executionCommon) resolveExecution(ctx context.Context) error {
	log.Debugf("Preparing execution of operation %q on node %q for deployment %q", e.operation.Name, e.NodeName, e.deploymentID)
	ovPath, err := operations.GetOverlayPath(e.cfg, e.taskID, e.deploymentID)
	if err != nil {
		return err
	}
	e.OverlayPath = ovPath

	if err = e.resolveInputs(ctx); err != nil {
		return err
	}

	if err = e.resolveArtifacts(ctx); err != nil {
		return err
	}
	if e.isRelationshipTargetNode {
		err = e.resolveHosts(ctx, e.operation.RelOp.TargetNodeName)
	} else {
		err = e.resolveHosts(ctx, e.NodeName)
	}
	if err != nil {
		return err
	}
	if err = e.resolveOperationOutputPath(); err != nil {
		return err
	}
	if err = e.addRunnablesSpecificInputsAndOutputs(); err != nil {
		return err
	}
	return e.resolveContext(ctx)

}

func (e *executionCommon) execute(ctx context.Context, retry bool) error {
	if e.isPerInstanceOperation {
		var nodeName string
		var instances []string
		if !e.isRelationshipTargetNode {
			nodeName = e.operation.RelOp.TargetNodeName
			instances = e.targetNodeInstances
		} else {
			nodeName = e.NodeName
			instances = e.sourceNodeInstances
		}

		for _, instanceID := range instances {
			instanceName := operations.GetInstanceName(nodeName, instanceID)
			log.Debugf("Executing operation %q, on node %q, with current instance %q", e.operation.Name, e.NodeName, instanceName)
			ctx = events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.InstanceID: instanceID})
			err := e.executeWithCurrentInstance(ctx, retry, instanceName)
			if err != nil {
				return err
			}
		}
	} else {
		return e.executeWithCurrentInstance(ctx, retry, "")
	}
	return nil
}

func (e *executionCommon) generateHostConnectionForOrchestratorOperation(ctx context.Context, buffer *bytes.Buffer) error {
	if e.cli != nil && e.cfg.Ansible.HostedOperations.DefaultSandbox != nil {
		var err error
		e.containerID, err = createSandbox(ctx, e.cli, e.cfg.Ansible.HostedOperations.DefaultSandbox, e.deploymentID)
		if err != nil {
			return err
		}
		buffer.WriteString(" ansible_connection=docker ansible_host=")
		buffer.WriteString(e.containerID)
	} else if e.cfg.Ansible.HostedOperations.UnsandboxedOperationsAllowed {
		buffer.WriteString(" ansible_connection=local")
	} else {
		actualRootCause := "there is no sandbox configured to handle it"
		if e.cli == nil {
			actualRootCause = "connection to docker failed (see logs)"
		}

		err := errors.Errorf("Ansible provisioning: you are trying to execute an operation on the orchestrator host but %s and execution on the actual orchestrator host is disallowed by configuration", actualRootCause)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).Registerf("%v", err)
		return err
	}
	return nil
}

func (e *executionCommon) getSSHCredentials(ctx context.Context, host *hostConnection) (sshCredentials, error) {

	creds := sshCredentials{}
	sshUser := host.user
	if sshUser == "" {
		// Use root as default user
		sshUser = "root"
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, e.deploymentID).RegisterAsString("Ansible provisioning: Missing ssh user information, trying to use root user.")
	}
	creds.user = sshUser
	sshPassword := host.password
	sshPrivateKeys := host.privateKeys
	if len(sshPrivateKeys) == 0 && sshPassword == "" {
		defaultKey, err := sshutil.GetDefaultKey()
		if err != nil {
			return creds, err
		}
		sshPrivateKeys = map[string]*sshutil.PrivateKey{"0": defaultKey}
		host.privateKeys = sshPrivateKeys
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, e.deploymentID).RegisterAsString("Ansible provisioning: Missing ssh password or private key information, trying to use default private key ~/.ssh/yorc.pem.")
	}
	creds.privateKeys = sshPrivateKeys
	creds.password = sshPassword
	return creds, nil
}

func (e *executionCommon) generateHostConnection(ctx context.Context, buffer *bytes.Buffer, host *hostConnection) error {
	buffer.WriteString(host.host)

	if host.bastion != nil {
		if host.bastion.Password != "" {
			return errors.New("ansible provider does not support password authentication with bastion hosts")
		}
		if host.bastion.Port == "" {
			host.bastion.Port = "22"
		}
		buffer.WriteString(fmt.Sprintf(" ansible_ssh_common_args='-o ProxyCommand=\"ssh -W %%h:%%p "+
			// disable host key checking on bastion host completeley
			"-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "+
			"-p %s %s@%s\"'", host.bastion.Port, host.bastion.User, host.bastion.Host))
	}

	if e.isOrchestratorOperation {
		err := e.generateHostConnectionForOrchestratorOperation(ctx, buffer)
		if err != nil {
			return err
		}
	} else {
		sshCredentials, err := e.getSSHCredentials(ctx, host)
		if err != nil {
			return err
		}
		buffer.WriteString(fmt.Sprintf(" ansible_ssh_user=%s", sshCredentials.user))
		// Set with priority private key against password
		if e.cfg.DisableSSHAgent && len(sshCredentials.privateKeys) > 0 {
			key := sshutil.SelectPrivateKeyOnName(sshCredentials.privateKeys, true)
			if key == nil {
				return errors.Errorf("%d private keys provided (may include the default key %q) but none are stored on disk. As ssh-agent is disabled by configuration we can't use direct key content.", len(sshCredentials.privateKeys), sshutil.DefaultSSHPrivateKeyFilePath)
			}
			buffer.WriteString(fmt.Sprintf(" ansible_ssh_private_key_file=%s", key.Path))
		} else if sshCredentials.password != "" {
			// TODO use ansible vault
			buffer.WriteString(fmt.Sprintf(" ansible_ssh_pass=%s", sshCredentials.password))
		}
		// Specify SSH port when different than default 22
		if host.port != 0 && host.port != 22 {
			buffer.WriteString(fmt.Sprintf(" ansible_ssh_port=%d", host.port))
		}
		// Specific ansible connection settings are needed on windows targets
		if strings.Contains(host.osType, "windows") {
			buffer.WriteString(" ansible_connection=ssh ansible_shell_type=cmd")
		}
	}
	buffer.WriteString("\n")
	return nil
}

func (e *executionCommon) executeWithCurrentInstance(ctx context.Context, retry bool, currentInstance string) error {
	// Create a cancel func here to remove docker sandboxes as soon as we exit this function
	ctx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	ansiblePath := filepath.Join(e.cfg.WorkingDirectory, "deployments", e.deploymentID, "ansible")
	ansiblePath, err := filepath.Abs(ansiblePath)
	if err != nil {
		return err
	}

	ansibleExecutionRootDir := filepath.Join(ansiblePath, stringutil.UniqueTimestampedName(e.taskID+"_", ""))
	ansibleRecipePath := filepath.Join(ansibleExecutionRootDir, e.NodeName)
	if e.operation.RelOp.IsRelationshipOperation {
		ansibleRecipePath = filepath.Join(ansibleRecipePath, e.relationshipType, e.operation.RelOp.TargetRelationship, e.operation.Name, currentInstance)
	} else {
		ansibleRecipePath = filepath.Join(ansibleRecipePath, e.operation.Name, currentInstance)
	}

	defer func() {
		if !e.cfg.Ansible.KeepGeneratedRecipes {
			err := os.RemoveAll(ansibleExecutionRootDir)
			if err != nil {
				err = errors.Wrapf(err, "Failed to remove ansible execution directory %q for node %q operation %q", ansibleExecutionRootDir, e.NodeName, e.operation.Name)
				log.Debugf("%+v", err)
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(err.Error())
			}
		}
	}()
	ansibleHostVarsPath := filepath.Join(ansibleRecipePath, "host_vars")
	if err = os.MkdirAll(ansibleHostVarsPath, 0775); err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(err.Error())
		return err
	}

	pythonInterpreter := "python"
	if _, err := exec.LookPath(pythonInterpreter); err != nil {
		log.Debug("Found no python intepreter, attempting to use python3")
		pythonInterpreter = "python3"
		if _, err = exec.LookPath(pythonInterpreter); err != nil {
			return fmt.Errorf("Found no python or python3 interpret in path")
		}
	}

	vaultPassScript := fmt.Sprintf(vaultPassScriptFormat, pythonInterpreter)
	if err = ioutil.WriteFile(filepath.Join(ansibleRecipePath, ".vault_pass"), []byte(vaultPassScript), 0764); err != nil {
		err = errors.Wrap(err, "Failed to write .vault_pass file")
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(err.Error())
		return err
	}

	var buffer bytes.Buffer
	var header, emptySectionHeader string
	if e.isOrchestratorOperation {
		header = ansibleInventoryHostedHeader
		emptySectionHeader = ansibleInventoryHostsHeader
	} else {
		header = ansibleInventoryHostsHeader
		emptySectionHeader = ansibleInventoryHostedHeader
	}
	buffer.WriteString(fmt.Sprintf("[%s]\n", emptySectionHeader))
	buffer.WriteString(fmt.Sprintf("[%s]\n", header))
	for instanceName, host := range e.hosts {
		err = e.generateHostConnection(ctx, &buffer, host)
		if err != nil {
			return err
		}
		var perInstanceInputsBuffer bytes.Buffer
		for _, varInput := range e.VarInputsNames {
			if varInput == "INSTANCE" {
				perInstanceInputsBuffer.WriteString(fmt.Sprintf("INSTANCE: %q\n", instanceName))
			} else if varInput == "SOURCE_INSTANCE" {
				if !e.isPerInstanceOperation {
					perInstanceInputsBuffer.WriteString(fmt.Sprintf("SOURCE_INSTANCE: %q\n", instanceName))
				} else {
					if e.isRelationshipTargetNode {
						perInstanceInputsBuffer.WriteString(fmt.Sprintf("SOURCE_INSTANCE: %q\n", currentInstance))
					} else {
						perInstanceInputsBuffer.WriteString(fmt.Sprintf("SOURCE_INSTANCE: %q\n", instanceName))
					}
				}
			} else if varInput == "TARGET_INSTANCE" {
				if !e.isPerInstanceOperation {
					perInstanceInputsBuffer.WriteString(fmt.Sprintf("TARGET_INSTANCE: %q\n", instanceName))
				} else {
					if e.isRelationshipTargetNode {
						perInstanceInputsBuffer.WriteString(fmt.Sprintf("TARGET_INSTANCE: %q\n", instanceName))
					} else {
						perInstanceInputsBuffer.WriteString(fmt.Sprintf("TARGET_INSTANCE: %q\n", currentInstance))
					}
				}
			} else {
				for _, envInput := range e.EnvInputs {
					if envInput.Name == varInput && (envInput.InstanceName == instanceName || e.isPerInstanceOperation && envInput.InstanceName == currentInstance) {
						v, err := e.encodeEnvInputValue(envInput, ansibleRecipePath)
						if err != nil {
							return err
						}
						perInstanceInputsBuffer.WriteString(fmt.Sprintf("%s: %s\n", varInput, v))
						goto NEXT
					}
				}
				if e.operation.RelOp.IsRelationshipOperation {
					var hostedOn bool
					hostedOn, err = deployments.IsTypeDerivedFrom(ctx, e.deploymentID, e.relationshipType, "tosca.relationships.HostedOn")
					if err != nil {
						return err
					} else if hostedOn {
						// In case of operation for relationships derived from HostedOn we should match the inputs with the same instanceID
						instanceIDIdx := strings.LastIndex(instanceName, "_")
						// Get index
						if instanceIDIdx > 0 {
							instanceID := instanceName[instanceIDIdx:]
							for _, envInput := range e.EnvInputs {
								if envInput.Name == varInput && strings.HasSuffix(envInput.InstanceName, instanceID) {
									v, err := e.encodeEnvInputValue(envInput, ansibleRecipePath)
									if err != nil {
										return err
									}
									perInstanceInputsBuffer.WriteString(fmt.Sprintf("%s: %s\n", varInput, v))
									goto NEXT
								}
							}
						}
					}
				}
				// Not found with the combination inputName/instanceName let's use the first that matches the input name
				for _, envInput := range e.EnvInputs {
					if envInput.Name == varInput {
						v, err := e.encodeEnvInputValue(envInput, ansibleRecipePath)
						if err != nil {
							return err
						}
						perInstanceInputsBuffer.WriteString(fmt.Sprintf("%s: %s\n", varInput, v))
						goto NEXT
					}
				}
				return errors.Errorf("Unable to find a suitable input for input name %q and instance %q", varInput, instanceName)
			}
		NEXT:
		}
		if perInstanceInputsBuffer.Len() > 0 {
			if err = ioutil.WriteFile(filepath.Join(ansibleHostVarsPath, host.host+".yml"), perInstanceInputsBuffer.Bytes(), 0664); err != nil {
				return errors.Wrapf(err, "Failed to write vars for host %q file: %v", host.host, err)
			}
		}
	}

	// Add inventory settings
	inventoryConfig := make(map[string][]string)
	for header, vars := range ansibleInventoryConfig {
		inventoryConfig[header] = append(inventoryConfig[header], vars...)
	}
	// Add variables in Yorc configuration, potentially overriding Yorc
	// default values
	for header, vars := range e.cfg.Ansible.Inventory {
		// The header can be quoted in configuration if it contains a colon
		key, err := strconv.Unquote(header)
		if err != nil {
			key = header
		}
		inventoryConfig[key] = append(inventoryConfig[header], vars...)
	}

	// Create corresponding entries in inventory
	for header, vars := range inventoryConfig {
		buffer.WriteString(fmt.Sprintf("[%s]\n", header))
		for _, val := range vars {
			buffer.WriteString(fmt.Sprintf("%s\n", val))
		}
	}

	if err = ioutil.WriteFile(filepath.Join(ansibleRecipePath, "hosts"), buffer.Bytes(), 0664); err != nil {
		err = errors.Wrap(err, "Failed to write hosts file")
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(err.Error())
		return err
	}

	// Generating Ansible config
	if err = e.generateAnsibleConfigurationFile(ansiblePath, ansibleRecipePath); err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(err.Error())
		return err
	}

	// e.OperationRemoteBaseDir is an unique base temp directory for multiple executions
	e.OperationRemoteBaseDir = stringutil.UniqueTimestampedName(e.cfg.Ansible.OperationRemoteBaseDir+"_", "")
	if e.operation.RelOp.IsRelationshipOperation {
		e.OperationRemotePath = path.Join(e.OperationRemoteBaseDir, e.NodeName, e.relationshipType, e.operation.Name)
	} else {
		e.OperationRemotePath = path.Join(e.OperationRemoteBaseDir, e.NodeName, e.operation.Name)
	}
	log.Debugf("OperationRemotePath:%s", e.OperationRemotePath)
	// Build archives for artifacts
	for artifactName, artifactPath := range e.Artifacts {
		tarPath := filepath.Join(ansibleRecipePath, artifactName+".tar")
		err := buildArchive(e.OverlayPath, artifactPath, tarPath)
		if err != nil {
			return err
		}
	}

	err = e.ansibleRunner.runAnsible(ctx, retry, currentInstance, ansibleRecipePath)
	if err != nil {
		return err
	}
	if e.HaveOutput {
		outputsFiles, err := filepath.Glob(filepath.Join(ansibleRecipePath, "*-out.csv"))
		if err != nil {
			err = errors.Wrapf(err, "Output retrieving of Ansible execution for node %q failed", e.NodeName)
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(err.Error())
			return err
		}
		for _, outFile := range outputsFiles {
			baseFileName := filepath.Base(outFile)
			hostname := strings.TrimSuffix(baseFileName, "-out.csv")
			fileInstanceID := getInstanceIDForHost(hostname, e.hosts)
			fi, err := os.Open(outFile)
			if err != nil {
				err = errors.Wrapf(err, "Output retrieving of Ansible execution for node %q failed", e.NodeName)
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(err.Error())
				return err
			}
			r := csv.NewReader(fi)

			// If LazyQuotes is true, a quote may appear in an unquoted field
			// and non-doubled quote may appear in a quoted field.
			// fix issue: https://github.com/golang/go/issues/21672
			r.LazyQuotes = true
			records, err := r.ReadAll()
			if err != nil {
				err = errors.Wrapf(err, "Output retrieving of Ansible execution for node %q failed", e.NodeName)
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(err.Error())
				return err
			}
			for _, line := range records {
				splits := strings.Split(line[0], "_")
				instanceID := splits[len(splits)-1]
				if instanceID != fileInstanceID {
					continue
				}
				if e.Outputs[line[0]] != taskContextOutput {
					if strings.HasPrefix(e.Outputs[line[0]], "attribute_mapping") {
						// attribute_mapping/<node_name>/<instance_name>/<attribute_name_or_capability_name>/<nested_attribute_name><nested_attribute_name><nested_attribute_name>...
						data := strings.Split(e.Outputs[line[0]], "/")
						nodeName := data[1]
						instanceName := data[2]
						attributeOrCapabilityName := data[3]
						var parameters []string
						if len(data) > 3 {
							parameters = data[4:]
						}
						if err = deployments.ResolveAttributeMapping(ctx, e.deploymentID, nodeName, instanceName, attributeOrCapabilityName, line[1], parameters...); err != nil {
							return err
						}
					} else {
						// TODO this should be part of the deployments package
						if err = consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, e.deploymentID, "topology", e.Outputs[line[0]]), line[1]); err != nil {
							return err
						}
						// Notify attributes on value change
						ind := strings.LastIndex(e.Outputs[line[0]], "/outputs/")
						if ind != -1 {
							outputPath := e.Outputs[line[0]][ind+len("/outputs/"):]
							data := strings.Split(outputPath, "/")
							if len(data) > 2 {
								notifier := &deployments.OperationOutputNotifier{
									InstanceName:  instanceID,
									NodeName:      e.NodeName,
									InterfaceName: data[0],
									OperationName: data[1],
									OutputName:    data[2],
								}
								err = notifier.NotifyValueChange(ctx, e.deploymentID)
								if err != nil {
									return err
								}
							}
						}
					}
				} else {
					err := tasks.SetTaskData(e.taskID, e.NodeName+"-"+instanceID+"-"+strings.Join(splits[0:len(splits)-1], "_"), line[1])
					if err != nil {
						return err
					}

				}
			}
		}
	}
	return nil

}

func (e *executionCommon) checkAnsibleRetriableError(ctx context.Context, err error) error {
	log.Debugf(err.Error())
	if exiterr, ok := err.(*exec.ExitError); ok {
		// The program has exited with an exit code != 0

		// This works on both Unix and Windows. Although package
		// syscall is generally platform dependent, WaitStatus is
		// defined for both Unix and Windows and in both cases has
		// an ExitStatus() method with the same signature.
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			// Exit Code 4 is corresponding to unreachable host and is eligible for connection retries
			// https://github.com/ansible/ansible/blob/devel/lib/ansible/executor/task_queue_manager.py
			if status.ExitStatus() == 4 {
				return ansibleRetriableError{root: err}
			}
		}

	}
	return err
}

func (e *executionCommon) getInstanceIDFromHost(host string) (string, error) {
	for _, hostConn := range e.hosts {
		if hostConn.host == host {
			return hostConn.instanceID, nil
		}
	}
	return "", errors.Errorf("Unknown host %q", host)
}

func (e *executionCommon) executePlaybook(ctx context.Context, retry bool,
	ansibleRecipePath string, handler outputHandler) error {
	cmd := executil.Command(ctx, "ansible-playbook", "-i", "hosts", "run.ansible.yml", "--vault-password-file", filepath.Join(ansibleRecipePath, ".vault_pass"))
	env := os.Environ()
	env = append(env, "VAULT_PASSWORD="+e.vaultToken)
	if _, err := os.Stat(filepath.Join(ansibleRecipePath, "run.ansible.retry")); retry && (err == nil || !os.IsNotExist(err)) {
		cmd.Args = append(cmd.Args, "--limit", filepath.Join("@", ansibleRecipePath, "run.ansible.retry"))
	}
	if e.cfg.Ansible.DebugExec {
		cmd.Args = append(cmd.Args, "-vvvv")
	} else {
		// One verbosity level is needed to get tasks output in playbooks yaml
		// output
		cmd.Args = append(cmd.Args, "-v")
	}

	if !e.isOrchestratorOperation {
		if e.cfg.Ansible.UseOpenSSH {
			cmd.Args = append(cmd.Args, "-c", "ssh")
		} else {
			cmd.Args = append(cmd.Args, "-c", "paramiko")
		}

		if !e.cfg.DisableSSHAgent {
			// Check if SSHAgent is needed
			sshAgent, err := e.configureSSHAgent(ctx)
			if err != nil {
				return errors.Wrap(err, "failed to configure SSH agent for ansible-playbook execution")
			}
			if sshAgent != nil {
				log.Debugf("Add SSH_AUTH_SOCK env var for ssh-agent")
				env = append(env, "SSH_AUTH_SOCK="+sshAgent.Socket)
				defer func() {
					err = sshAgent.RemoveAllKeys()
					if err != nil {
						log.Debugf("Warning: failed to remove all SSH agents keys due to error:%+v", err)
					}
					err = sshAgent.Stop()
					if err != nil {
						log.Debugf("Warning: failed to stop SSH agent due to error:%+v", err)
					}
				}()
			}
		}
	}
	cmd.Dir = ansibleRecipePath
	cmd.Env = env
	errbuf := events.NewBufferedLogEntryWriter()
	cmd.Stderr = errbuf

	errCloseCh := make(chan bool)
	defer close(errCloseCh)

	// Register log entry via error buffer
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RunBufferedRegistration(errbuf, errCloseCh)

	// Start handling the stdout and stderr for this command
	if err := handler.start(cmd.Cmd); err != nil {
		log.Printf("Error starting output handler: %s", err.Error())
	}

	err := cmd.Run()
	if handlerErr := handler.stop(); handlerErr != nil {
		log.Printf("Error stopping output handler: %s", handlerErr.Error())
	}
	if err != nil {
		return e.checkAnsibleRetriableError(ctx, err)
	}
	return nil
}

func (e *executionCommon) configureSSHAgent(ctx context.Context) (*sshutil.SSHAgent, error) {
	var addSSHAgent bool
	for _, host := range e.hosts {
		if len(host.privateKeys) > 0 {
			addSSHAgent = true
			break
		}
	}
	if !addSSHAgent {
		return nil, nil
	}

	agent, err := sshutil.NewSSHAgent(ctx)
	if err != nil {
		return nil, err
	}
	for _, host := range e.hosts {
		for _, key := range host.privateKeys {
			if err = agent.AddPrivateKey(key, 3600); err != nil {
				return nil, err
			}
		}
		if host.bastion != nil && len(host.bastion.PrivateKeys) > 0 {
			for _, key := range host.bastion.PrivateKeys {
				if err = agent.AddPrivateKey(key, 3600); err != nil {
					return nil, err
				}
			}
		}
	}
	return agent, nil
}

func buildArchive(rootDir, artifactDir, tarPath string) error {

	srcDir := filepath.Join(rootDir, artifactDir)
	tarFile, err := os.Create(tarPath)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	var fileWriter io.WriteCloser = tarFile

	tarfileWriter := tar.NewWriter(fileWriter)
	defer tarfileWriter.Close()

	_, err = os.Stat(srcDir)
	if err != nil {
		return nil
	}

	return filepath.Walk(srcDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}

			if rootDir != "" {
				header.Name = strings.TrimPrefix(strings.Replace(path, rootDir, "", -1), string(filepath.Separator))
			}

			if err := tarfileWriter.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(tarfileWriter, file)
			return err
		})
}

func (e *executionCommon) encodeTOSCAValue(value *deployments.TOSCAValue, ansibleRecipePath string) (string, error) {
	if !value.IsSecret {
		return fmt.Sprintf("%q", value.RawString()), nil
	}
	return e.vaultEncodeString(value.RawString(), ansibleRecipePath)

}

func (e *executionCommon) encodeEnvInputValue(env *operations.EnvInput, ansibleRecipePath string) (string, error) {
	if !env.IsSecret {
		return fmt.Sprintf("%q", env.Value), nil
	}

	return e.vaultEncodeString(env.Value, ansibleRecipePath)
}
func (e *executionCommon) vaultEncodeString(s, ansibleRecipePath string) (string, error) {
	cmd := executil.Command(e.ctx, "ansible-vault", "encrypt_string", "--vault-password-file", filepath.Join(ansibleRecipePath, ".vault_pass"))

	cmd.Env = append(os.Environ(), "VAULT_PASSWORD="+e.vaultToken)
	cmd.Stdin = strings.NewReader(s)
	outBuf := new(bytes.Buffer)
	cmd.Stdout = outBuf
	errBuf := new(bytes.Buffer)
	cmd.Stderr = errBuf

	err := cmd.Run()
	return outBuf.String(), errors.Wrapf(err, "failed to encode ansible vault token, stderr: %q", errBuf.String())

}

// generateAnsibleConfigurationFile generates an ansible.cfg file using default
// settings which can be completed/overriden by settings providing in Yorc Server
// configuration
func (e *executionCommon) generateAnsibleConfigurationFile(
	ansiblePath, ansibleRecipePath string) error {

	ansibleConfig := getAnsibleConfigFromDefault()

	// Adding settings whose values are known at runtime, related to the deployment
	// directory path
	ansibleConfig[ansibleConfigDefaultsHeader]["retry_files_save_path"] = ansibleRecipePath
	if e.CacheFacts {
		ansibleFactCaching["fact_caching_connection"] = path.Join(ansiblePath, "facts_cache")

		for k, v := range ansibleFactCaching {
			ansibleConfig[ansibleConfigDefaultsHeader][k] = v
		}
	}

	// Ansible configuration user-defined values provided in Yorc Server configuration
	// can override default settings
	for header, settings := range e.cfg.Ansible.Config {
		if _, ok := ansibleConfig[header]; !ok {
			ansibleConfig[header] = make(map[string]string, len(settings))
		}
		for k, v := range settings {
			ansibleConfig[header][k] = v
		}
	}

	var ansibleCfgContentBuilder strings.Builder
	for header, settings := range ansibleConfig {
		ansibleCfgContentBuilder.WriteString(fmt.Sprintf("[%s]\n", header))
		for k, v := range settings {
			ansibleCfgContentBuilder.WriteString(fmt.Sprintf("%s=%s\n", k, v))
		}
	}

	var err error
	if err = ioutil.WriteFile(
		filepath.Join(ansibleRecipePath, "ansible.cfg"),
		[]byte(ansibleCfgContentBuilder.String()), 0664); err != nil {

		err = errors.Wrap(err, "Failed to write ansible.cfg file")
	}

	return err
}

func getAnsibleConfigFromDefault() map[string]map[string]string {
	ansibleConfig := make(map[string]map[string]string, len(ansibleDefaultConfig))
	for k, mapVal := range ansibleDefaultConfig {
		newVal := make(map[string]string, len(mapVal))
		for internalK, internalV := range mapVal {
			newVal[internalK] = internalV
		}
		ansibleConfig[k] = newVal
	}
	return ansibleConfig
}
