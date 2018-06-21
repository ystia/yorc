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
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/moby/moby/client"
	"github.com/pkg/errors"

	"gopkg.in/yaml.v2"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/helper/executil"
	"github.com/ystia/yorc/helper/provutil"
	"github.com/ystia/yorc/helper/stringutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
	"github.com/ystia/yorc/prov/operations"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/tosca"
)

const ansibleConfig = `[defaults]
host_key_checking=False
timeout=30
stdout_callback = json
retry_files_save_path = #PLAY_PATH#
`

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
	host       string
	port       int
	user       string
	instanceID string
	privateKey string
	password   string
}

type execution interface {
	resolveExecution() error
	execute(ctx context.Context, retry bool) error
}

type ansibleRunner interface {
	runAnsible(ctx context.Context, retry bool, currentInstance, ansibleRecipePath string) error
}
type executionCommon struct {
	kv                       *api.KV
	cfg                      config.Configuration
	ctx                      context.Context
	deploymentID             string
	taskID                   string
	NodeName                 string
	operation                prov.Operation
	NodeType                 string
	Description              string
	OperationRemoteBaseDir   string
	OperationRemotePath      string
	KeepOperationRemotePath  bool
	EnvInputs                []*operations.EnvInput
	VarInputsNames           []string
	Primary                  string
	BasePrimary              string
	Dependencies             []string
	hosts                    map[string]hostConnection
	OperationPath            string
	NodePath                 string
	NodeTypePath             string
	Artifacts                map[string]string
	OverlayPath              string
	Context                  map[string]string
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
}

func newExecution(ctx context.Context, kv *api.KV, cfg config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation, cli *client.Client) (execution, error) {
	execCommon := &executionCommon{kv: kv,
		cfg:          cfg,
		ctx:          ctx,
		deploymentID: deploymentID,
		NodeName:     nodeName,
		//KeepOperationRemotePath property is required to be public when resolving templates.
		KeepOperationRemotePath: cfg.Ansible.KeepOperationRemotePath,
		operation:               operation,
		VarInputsNames:          make([]string, 0),
		EnvInputs:               make([]*operations.EnvInput, 0),
		taskID:                  taskID,
		Outputs:                 make(map[string]string),
		cli:                     cli,
	}
	if err := execCommon.resolveOperation(); err != nil {
		return nil, err
	}
	isBash, err := deployments.IsTypeDerivedFrom(kv, deploymentID, operation.ImplementationArtifact, implementationArtifactBash)
	if err != nil {
		return nil, err
	}
	isPython, err := deployments.IsTypeDerivedFrom(kv, deploymentID, operation.ImplementationArtifact, implementationArtifactPython)
	if err != nil {
		return nil, err
	}
	isAnsible, err := deployments.IsTypeDerivedFrom(kv, deploymentID, operation.ImplementationArtifact, implementationArtifactAnsible)
	if err != nil {
		return nil, err
	}
	var exec execution
	if isBash || isPython {
		execScript := &executionScript{executionCommon: execCommon, isPython: isPython}
		execCommon.ansibleRunner = execScript
		exec = execScript
	} else if isAnsible {
		execAnsible := &executionAnsible{executionCommon: execCommon}
		execCommon.ansibleRunner = execAnsible
		exec = execAnsible
	} else {
		return nil, errors.Errorf("Unsupported artifact implementation for node: %q, operation: %s, primary implementation: %q", nodeName, operation.Name, execCommon.Primary)
	}

	return exec, exec.resolveExecution()
}

func (e *executionCommon) resolveOperation() error {
	e.NodePath = path.Join(consulutil.DeploymentKVPrefix, e.deploymentID, "topology/nodes", e.NodeName)
	var err error
	e.NodeType, err = deployments.GetNodeType(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}
	e.NodeTypePath = path.Join(consulutil.DeploymentKVPrefix, e.deploymentID, "topology/types", e.NodeType)
	if e.operation.RelOp.IsRelationshipOperation {
		e.relationshipType, err = deployments.GetRelationshipForRequirement(e.kv, e.deploymentID, e.NodeName, e.operation.RelOp.RequirementIndex)
		if err != nil {
			return err
		}
		err = e.resolveIsPerInstanceOperation(e.operation.Name)
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
	e.OperationPath, e.Primary, err = deployments.GetOperationPathAndPrimaryImplementationForNodeType(e.kv, e.deploymentID, operationNodeType, e.operation.Name)
	if err != nil {
		return err
	}
	if e.OperationPath == "" || e.Primary == "" {
		return operationNotImplemented{msg: fmt.Sprintf("primary implementation missing for operation %q of type %q in deployment %q is missing", e.operation.Name, e.NodeType, e.deploymentID)}
	}
	e.Primary = strings.TrimSpace(e.Primary)
	log.Debugf("Operation Path: %q, primary implementation: %q", e.OperationPath, e.Primary)
	e.BasePrimary = path.Base(e.Primary)
	kvPair, _, err := e.kv.Get(e.OperationPath+"/implementation/dependencies", nil)
	if err != nil {
		return err
	}

	if kvPair != nil {
		e.Dependencies = strings.Split(string(kvPair.Value), ",")
	} else {
		e.Dependencies = make([]string, 0)
	}
	kvPair, _, err = e.kv.Get(e.OperationPath+"/description", nil)
	if err != nil {
		return errors.Wrap(err, "Consul query failed: ")
	}
	if kvPair != nil && len(kvPair.Value) > 0 {
		e.Description = string(kvPair.Value)
	}

	// if operation_host is not overridden by requirement, we retrieve operation/implementation definition info
	if e.operation.OperationHost == "" {
		kvPair, _, err = e.kv.Get(e.OperationPath+"/implementation/operation_host", nil)
		if err != nil {
			return errors.Wrap(err, "Consul query failed: ")
		}
		if kvPair != nil && len(kvPair.Value) > 0 {
			e.operation.OperationHost = string(kvPair.Value)
		}
	}

	e.isOrchestratorOperation = operations.IsOrchestratorHostOperation(e.operation)
	e.isRelationshipTargetNode = operations.IsRelationshipTargetNodeOperation(e.operation)
	return e.resolveInstances()
}

func (e *executionCommon) resolveInstances() error {
	var err error
	if e.operation.RelOp.IsRelationshipOperation {
		e.targetNodeInstances, err = tasks.GetInstances(e.kv, e.taskID, e.deploymentID, e.operation.RelOp.TargetNodeName)
		if err != nil {
			return err
		}
	}
	e.sourceNodeInstances, err = tasks.GetInstances(e.kv, e.taskID, e.deploymentID, e.NodeName)

	return err
}

func (e *executionCommon) resolveArtifacts() error {
	log.Debugf("Resolving artifacts")
	var err error
	if e.operation.RelOp.IsRelationshipOperation {
		// First get linked node artifacts
		if e.isRelationshipTargetNode {
			e.Artifacts, err = deployments.GetArtifactsForNode(e.kv, e.deploymentID, e.operation.RelOp.TargetNodeName)
			if err != nil {
				return err
			}
		} else {
			e.Artifacts, err = deployments.GetArtifactsForNode(e.kv, e.deploymentID, e.NodeName)
			if err != nil {
				return err
			}
		}
		// Then get relationship type artifacts
		var arts map[string]string
		arts, err = deployments.GetArtifactsForType(e.kv, e.deploymentID, e.relationshipType)
		if err != nil {
			return err
		}
		for artName, art := range arts {
			e.Artifacts[artName] = art
		}
	} else {
		e.Artifacts, err = deployments.GetArtifactsForNode(e.kv, e.deploymentID, e.NodeName)
		if err != nil {
			return err
		}
	}
	log.Debugf("Resolved artifacts: %v", e.Artifacts)
	return nil
}

func (e *executionCommon) setHostConnection(kv *api.KV, host, instanceID, capType string, conn *hostConnection) error {
	hasEndpoint, err := deployments.IsTypeDerivedFrom(e.kv, e.deploymentID, capType, "yorc.capabilities.Endpoint.ProvisioningAdmin")
	if err != nil {
		return err
	}
	if hasEndpoint {
		found, user, err := deployments.GetInstanceCapabilityAttribute(e.kv, e.deploymentID, host, instanceID, "endpoint", "credentials", "user")
		if err != nil {
			return err
		}
		if found {
			conn.user = config.DefaultConfigTemplateResolver.ResolveValueWithTemplates("host.user", user).(string)
		} else {
			mess := fmt.Sprintf("[Warning] No user set for connection:%+v", conn)
			log.Printf(mess)
			events.WithContextOptionalFields(e.ctx).NewLogEntry(events.WARN, e.deploymentID).RegisterAsString(mess)
		}
		found, password, err := deployments.GetInstanceCapabilityAttribute(e.kv, e.deploymentID, host, instanceID, "endpoint", "credentials", "token")
		if err != nil {
			return err
		}
		if found && password != "" {
			conn.password = config.DefaultConfigTemplateResolver.ResolveValueWithTemplates("host.password", password).(string)
		}
		found, privateKey, err := deployments.GetInstanceCapabilityAttribute(e.kv, e.deploymentID, host, instanceID, "endpoint", "credentials", "keys", "0")
		if err != nil {
			return err
		}
		if found && privateKey != "" {
			conn.privateKey = config.DefaultConfigTemplateResolver.ResolveValueWithTemplates("host.privateKey", privateKey).(string)
		}

		found, port, err := deployments.GetInstanceCapabilityAttribute(e.kv, e.deploymentID, host, instanceID, "endpoint", "port")
		if err != nil {
			return err
		}
		if found && port != "" {
			conn.port, err = strconv.Atoi(port)
			if err != nil {
				return errors.Wrapf(err, "Failed to convert port value:%q to int", port)
			}
		}
	}
	return nil
}

func (e *executionCommon) resolveHostsOrchestratorLocal(nodeName string, instances []string) error {
	e.hosts = make(map[string]hostConnection, len(instances))
	for i := range instances {
		instanceName := operations.GetInstanceName(nodeName, instances[i])
		e.hosts[instanceName] = hostConnection{host: instanceName}
	}
	return nil
}

func (e *executionCommon) resolveHostsOnCompute(nodeName string, instances []string) error {
	hostedOnList := make([]string, 0)
	hostedOnList = append(hostedOnList, nodeName)
	parentHost, err := deployments.GetHostedOnNode(e.kv, e.deploymentID, nodeName)
	if err != nil {
		return err
	}
	for parentHost != "" {
		hostedOnList = append(hostedOnList, parentHost)
		parentHost, err = deployments.GetHostedOnNode(e.kv, e.deploymentID, parentHost)
		if err != nil {
			return err
		}
	}

	hosts := make(map[string]hostConnection)

	for i := len(hostedOnList) - 1; i >= 0; i-- {
		host := hostedOnList[i]
		capType, err := deployments.GetNodeCapabilityType(e.kv, e.deploymentID, host, "endpoint")
		if err != nil {
			return err
		}

		hasEndpoint, err := deployments.IsTypeDerivedFrom(e.kv, e.deploymentID, capType, "tosca.capabilities.Endpoint")
		if err != nil {
			return err
		}
		if hasEndpoint {
			for _, instance := range instances {
				_, ipAddress, err := deployments.GetInstanceCapabilityAttribute(e.kv, e.deploymentID, host, instance, "endpoint", "ip_address")
				if err != nil {
					return err
				}
				if ipAddress != "" {
					ipAddress = config.DefaultConfigTemplateResolver.ResolveValueWithTemplates("host.ip_address", ipAddress).(string)
					instanceName := operations.GetInstanceName(nodeName, instance)
					hostConn := hostConnection{host: ipAddress, instanceID: instance}
					err = e.setHostConnection(e.kv, host, instance, capType, &hostConn)
					if err != nil {
						mess := fmt.Sprintf("[ERROR] failed to set host connection with error: %+v", err)
						log.Debug(mess)
						events.WithContextOptionalFields(e.ctx).NewLogEntry(events.ERROR, e.deploymentID).RegisterAsString(mess)
						return err
					}
					hosts[instanceName] = hostConn
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

func (e *executionCommon) resolveHosts(nodeName string) error {
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
	return e.resolveHostsOnCompute(nodeName, instances)
}

func (e *executionCommon) resolveContext() error {
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
		if host, err := deployments.GetHostedOnNode(e.kv, e.deploymentID, e.NodeName); err != nil {
			return err
		} else if host != "" {
			execContext["HOST"] = host
		}
	} else {

		if host, err := deployments.GetHostedOnNode(e.kv, e.deploymentID, e.NodeName); err != nil {
			return err
		} else if host != "" {
			execContext["SOURCE_HOST"] = host
		}
		if host, err := deployments.GetHostedOnNode(e.kv, e.deploymentID, e.operation.RelOp.TargetNodeName); err != nil {
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

	capabilitiesCtx, err := operations.GetTargetCapabilityPropertiesAndAttributes(e.ctx, e.kv, e.deploymentID, e.NodeName, e.operation)
	if err != nil {
		return err
	}
	for k, v := range capabilitiesCtx {
		execContext[k] = v
	}

	execContext["DEPLOYMENT_ID"] = e.deploymentID
	e.Context = execContext
	return nil
}

func (e *executionCommon) resolveOperationOutputPath() error {
	//Here we get the modelable entity output of the operation
	entities, _, err := e.kv.Keys(e.OperationPath+"/outputs/", "/", nil)
	if err != nil {
		return err
	}

	if len(entities) == 0 {
		return nil
	}

	e.HaveOutput = true
	//We iterate over all entity of the output in this operation
	for _, entity := range entities {
		//We get the name of the output
		outputKeys, _, err := e.kv.Keys(entity, "/", nil)
		if err != nil {
			return err
		}
		for _, output := range outputKeys {
			//We get the expression  of the output
			kvPair, _, err := e.kv.Get(output+"/expression", nil)
			if err != nil {
				return err
			}
			if kvPair == nil {
				return errors.Errorf("Operation output expression is missing for key: %q", output)
			}
			va := &tosca.ValueAssignment{}
			err = yaml.Unmarshal(kvPair.Value, va)
			if err != nil {
				return errors.Wrap(err, "Fail to parse operation output, check the following expression : ")
			}
			if va.Type != tosca.ValueAssignmentFunction {
				return errors.Errorf("Output %q for operation %v is not a valid get_operation_output TOSCA function", path.Base(output), e.operation)
			}
			oof := va.GetFunction()
			if oof.Operator != tosca.GetOperationOutputOperator {
				return errors.Errorf("Output %q for operation %v (%v) is not a valid get_operation_output TOSCA function", path.Base(output), e.operation, oof)
			}
			targetContext := oof.Operands[0].String() == "TARGET"
			sourceContext := oof.Operands[0].String() == "SOURCE"
			if (targetContext || sourceContext) && !e.operation.RelOp.IsRelationshipOperation {
				return errors.Errorf("Can't resolve an operation output in SOURCE or TARGET context without a relationship operation: %q", va.String())
			}

			var instancesIds []string
			if targetContext {
				instancesIds = e.targetNodeInstances
			} else {
				instancesIds = e.sourceNodeInstances
			}

			//For each instance of the node we create a new entry in the output map
			for _, instanceID := range instancesIds {
				//We decide to add an in to differentiate if we export many time the same output
				b := uint32(time.Now().Nanosecond())
				interfaceName := strings.ToLower(url.QueryEscape(oof.Operands[1].String()))
				operationName := strings.ToLower(url.QueryEscape(oof.Operands[2].String()))
				outputVariableName := url.QueryEscape(oof.Operands[3].String())
				if targetContext {
					e.Outputs[outputVariableName+"_"+fmt.Sprint(b)] = path.Join("instances", e.operation.RelOp.TargetNodeName, instanceID, "outputs", interfaceName, operationName, outputVariableName)
				} else {
					//If we are with an expression type {get_operation_output : [ SELF, ...]} in a relationship we store the result in the corresponding relationship instance
					if oof.Operands[0].String() == "SELF" && e.operation.RelOp.IsRelationshipOperation {
						relationShipPrefix := filepath.Join("relationship_instances", e.NodeName, e.operation.RelOp.RequirementIndex, instanceID)
						e.Outputs[outputVariableName+"_"+fmt.Sprint(b)] = path.Join(relationShipPrefix, "outputs", interfaceName, operationName, outputVariableName)
					} else if oof.Operands[0].String() == "HOST" {
						// In this case we continue because the parsing has change this type on {get_operation_output : [ SELF, ...]}  on the host node
						continue

					} else {
						//In all others case we simply save the result of the output on the instance directory of the node
						e.Outputs[outputVariableName+"_"+fmt.Sprint(b)] = path.Join("instances", e.NodeName, instanceID, "outputs", interfaceName, operationName, outputVariableName)
					}
				}

			}
		}
	}

	return nil
}

// resolveIsPerInstanceOperation sets e.isPerInstanceOperation to true if the given operationName contains one of the following patterns (case doesn't matter):
//	add_target, remove_target, add_source, remove_source, target_changed
// And in case of a relationship operation the relationship does not derive from "tosca.relationships.HostedOn" as it makes no sense till we scale at compute level
func (e *executionCommon) resolveIsPerInstanceOperation(operationName string) error {
	op := strings.ToLower(operationName)
	if strings.Contains(op, "add_target") || strings.Contains(op, "remove_target") || strings.Contains(op, "target_changed") || strings.Contains(op, "add_source") || strings.Contains(op, "remove_source") {
		// Do not call the call the operation several time for a "HostedOn" relationship (makes no sense till we scale at compute level)
		if hostedOn, err := deployments.IsTypeDerivedFrom(e.kv, e.deploymentID, e.relationshipType, "tosca.relationships.HostedOn"); err != nil || hostedOn {
			e.isPerInstanceOperation = false
			return err
		}
		e.isPerInstanceOperation = true
		return nil
	}
	e.isPerInstanceOperation = false
	return nil
}

func (e *executionCommon) resolveInputs() error {
	var err error
	e.EnvInputs, e.VarInputsNames, err = operations.ResolveInputsWithInstances(e.kv, e.deploymentID, e.NodeName, e.taskID, e.operation, e.sourceNodeInstances, e.targetNodeInstances)
	return err
}

func (e *executionCommon) resolveExecution() error {
	log.Debugf("Preparing execution of operation %q on node %q for deployment %q", e.operation.Name, e.NodeName, e.deploymentID)
	ovPath, err := filepath.Abs(filepath.Join(e.cfg.WorkingDirectory, "deployments", e.deploymentID, "overlay"))
	if err != nil {
		return err
	}
	e.OverlayPath = ovPath

	if err = e.resolveInputs(); err != nil {
		return err
	}
	if err = e.resolveArtifacts(); err != nil {
		return err
	}
	if e.isRelationshipTargetNode {
		err = e.resolveHosts(e.operation.RelOp.TargetNodeName)
	} else {
		err = e.resolveHosts(e.NodeName)
	}
	if err != nil {
		return err
	}
	if err = e.resolveOperationOutputPath(); err != nil {
		return err
	}

	return e.resolveContext()

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
		containerID, err := createSandbox(ctx, e.cli, e.cfg.Ansible.HostedOperations.DefaultSandbox, e.deploymentID)
		if err != nil {
			return err
		}
		buffer.WriteString(" ansible_connection=docker ansible_host=")
		buffer.WriteString(containerID)
	} else if e.cfg.Ansible.HostedOperations.UnsandboxedOperationsAllowed {
		buffer.WriteString(" ansible_connection=local")
	} else {
		actualRootCause := "there is no sandbox configured to handle it"
		if e.cli == nil {
			actualRootCause = "connection to docker failed (see logs)"
		}

		err := errors.Errorf("Ansible provisioning: you are trying to execute an operation on the orchestrator host but %s and execution on the actual orchestrator host is disallowed by configuration", actualRootCause)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, e.deploymentID).Registerf("%v", err)
		return err
	}
	return nil
}

func (e *executionCommon) generateHostConnection(ctx context.Context, buffer *bytes.Buffer, host hostConnection) error {
	buffer.WriteString(host.host)
	if e.isOrchestratorOperation {
		err := e.generateHostConnectionForOrchestratorOperation(ctx, buffer)
		if err != nil {
			return err
		}
	} else {
		sshUser := host.user
		if sshUser == "" {
			// Use root as default user
			sshUser = "root"
			events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, e.deploymentID).RegisterAsString("Ansible provisioning: Missing ssh user information, trying to use root user.")
		}
		sshPassword := host.password
		sshPrivateKey := host.privateKey
		if sshPrivateKey == "" && sshPassword == "" {
			sshPrivateKey = "~/.ssh/yorc.pem"
			events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, e.deploymentID).RegisterAsString("Ansible provisioning: Missing ssh password or private key information, trying to use default private key ~/.ssh/yorc.pem.")
		}
		buffer.WriteString(fmt.Sprintf(" ansible_ssh_user=%s ansible_ssh_common_args=\"-o ConnectionAttempts=20\"", sshUser))
		if sshPrivateKey != "" {
			// TODO if not a path store it somewhere
			// Note whould be better if we can use it directly https://github.com/ansible/ansible/issues/22382
			buffer.WriteString(fmt.Sprintf(" ansible_ssh_private_key_file=%s", sshPrivateKey))
		}
		if sshPassword != "" {
			// TODO use vault
			buffer.WriteString(fmt.Sprintf(" ansible_ssh_pass=%s", sshPassword))
		}

		// Specify SSH port when different than default 22
		if host.port != 0 && host.port != 22 {
			buffer.WriteString(fmt.Sprintf(" ansible_ssh_port=%d", host.port))
		}
	}
	buffer.WriteString("\n")
	return nil
}

func (e *executionCommon) executeWithCurrentInstance(ctx context.Context, retry bool, currentInstance string) error {
	// Create a cancel func here to remove docker sandboxes as soon as we exit this function
	ctx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	// Fill log optional fields for log registration
	logOptFields, ok := events.FromContext(ctx)
	if !ok {
		return errors.New("Missing context log fields")
	}
	logOptFields[events.InstanceID] = currentInstance
	ctx = events.NewContext(ctx, logOptFields)
	events.WithContextOptionalFields(ctx).NewLogEntry(events.INFO, e.deploymentID).RegisterAsString("Start the ansible execution of : " + e.NodeName + " with operation : " + e.operation.Name)
	var ansibleRecipePath string
	if e.operation.RelOp.IsRelationshipOperation {
		ansibleRecipePath = filepath.Join(e.cfg.WorkingDirectory, "deployments", e.deploymentID, "ansible", e.NodeName, e.relationshipType, e.operation.RelOp.TargetRelationship, e.operation.Name, currentInstance)
	} else {
		ansibleRecipePath = filepath.Join(e.cfg.WorkingDirectory, "deployments", e.deploymentID, "ansible", e.NodeName, e.operation.Name, currentInstance)
	}
	ansibleRecipePath, err := filepath.Abs(ansibleRecipePath)
	if err != nil {
		return err
	}
	if err = os.RemoveAll(ansibleRecipePath); err != nil {
		err = errors.Wrapf(err, "Failed to remove ansible recipe directory %q for node %q operation %q", ansibleRecipePath, e.NodeName, e.operation.Name)
		log.Debugf("%+v", err)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, e.deploymentID).RegisterAsString(err.Error())
		return err
	}
	ansibleHostVarsPath := filepath.Join(ansibleRecipePath, "host_vars")
	if err = os.MkdirAll(ansibleHostVarsPath, 0775); err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, e.deploymentID).RegisterAsString(err.Error())
		return err
	}
	log.Debugf("Generating hosts files hosts: %+v ", e.hosts)
	var buffer bytes.Buffer
	buffer.WriteString("[all]\n")
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
						perInstanceInputsBuffer.WriteString(fmt.Sprintf("%s: %q\n", varInput, envInput.Value))
						goto NEXT
					}
				}
				if e.operation.RelOp.IsRelationshipOperation {
					var hostedOn bool
					hostedOn, err = deployments.IsTypeDerivedFrom(e.kv, e.deploymentID, e.relationshipType, "tosca.relationships.HostedOn")
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
									perInstanceInputsBuffer.WriteString(fmt.Sprintf("%s: %q\n", varInput, envInput.Value))
									goto NEXT
								}
							}
						}
					}
				}
				// Not found with the combination inputName/instanceName let's use the first that matches the input name
				for _, envInput := range e.EnvInputs {
					if envInput.Name == varInput {
						perInstanceInputsBuffer.WriteString(fmt.Sprintf("%s: %q\n", varInput, envInput.Value))
						goto NEXT
					}
				}
				return errors.Errorf("Unable to find a suitable input for input name %q and instance %q", varInput, instanceName)
			}
		NEXT:
		}
		if perInstanceInputsBuffer.Len() > 0 {
			if err = ioutil.WriteFile(filepath.Join(ansibleHostVarsPath, host.host+".yml"), perInstanceInputsBuffer.Bytes(), 0664); err != nil {
				return errors.Wrapf(err, "Failed to write vars for host %q file: %v", host, err)
			}
		}
	}

	if e.isOrchestratorOperation {
		buffer.WriteString("\n[all:vars]\n")
		buffer.WriteString("ansible_python_interpreter=/usr/bin/env python\n")
	}

	if err = ioutil.WriteFile(filepath.Join(ansibleRecipePath, "hosts"), buffer.Bytes(), 0664); err != nil {
		err = errors.Wrap(err, "Failed to write hosts file")
		events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, e.deploymentID).RegisterAsString(err.Error())
		return err
	}
	if err = ioutil.WriteFile(filepath.Join(ansibleRecipePath, "ansible.cfg"), []byte(strings.Replace(ansibleConfig, "#PLAY_PATH#", ansibleRecipePath, -1)), 0664); err != nil {
		err = errors.Wrap(err, "Failed to write ansible.cfg file")
		events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, e.deploymentID).RegisterAsString(err.Error())
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
	err = e.ansibleRunner.runAnsible(ctx, retry, currentInstance, ansibleRecipePath)
	if err != nil {
		return err
	}
	if e.HaveOutput {
		outputsFiles, err := filepath.Glob(filepath.Join(ansibleRecipePath, "*-out.csv"))
		if err != nil {
			err = errors.Wrapf(err, "Output retrieving of Ansible execution for node %q failed", e.NodeName)
			events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, e.deploymentID).RegisterAsString(err.Error())
			return err
		}
		for _, outFile := range outputsFiles {
			fi, err := os.Open(outFile)
			if err != nil {
				err = errors.Wrapf(err, "Output retrieving of Ansible execution for node %q failed", e.NodeName)
				events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, e.deploymentID).RegisterAsString(err.Error())
				return err
			}
			r := csv.NewReader(fi)
			records, err := r.ReadAll()
			if err != nil {
				err = errors.Wrapf(err, "Output retrieving of Ansible execution for node %q failed", e.NodeName)
				events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, e.deploymentID).RegisterAsString(err.Error())
				return err
			}
			for _, line := range records {
				if err = consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, e.deploymentID, "topology", e.Outputs[line[0]]), line[1]); err != nil {
					return err
				}

			}
		}
	}
	return nil

}

func (e *executionCommon) checkAnsibleRetriableError(ctx context.Context, err error) error {
	events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, e.deploymentID).RegisterAsString(errors.Wrapf(err, "Ansible execution for operation %q on node %q failed", e.operation.Name, e.NodeName).Error())
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

func (e *executionCommon) executePlaybook(ctx context.Context, retry bool, ansibleRecipePath string, logFn logAnsibleOutputInConsulFn) error {
	cmd := executil.Command(ctx, "ansible-playbook", "-i", "hosts", "run.ansible.yml")

	if _, err := os.Stat(filepath.Join(ansibleRecipePath, "run.ansible.retry")); retry && (err == nil || !os.IsNotExist(err)) {
		cmd.Args = append(cmd.Args, "--limit", filepath.Join("@", ansibleRecipePath, "run.ansible.retry"))
	}
	if e.cfg.Ansible.DebugExec {
		cmd.Args = append(cmd.Args, "-vvvv")
	}
	if !e.isOrchestratorOperation {
		if e.cfg.Ansible.UseOpenSSH {
			cmd.Args = append(cmd.Args, "-c", "ssh")
		} else {
			cmd.Args = append(cmd.Args, "-c", "paramiko")
		}
	}
	cmd.Dir = ansibleRecipePath
	var outbuf bytes.Buffer
	errbuf := events.NewBufferedLogEntryWriter()
	cmd.Stdout = &outbuf
	cmd.Stderr = errbuf

	errCloseCh := make(chan bool)
	defer close(errCloseCh)

	// Register log entry via error buffer
	events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, e.deploymentID).RunBufferedRegistration(errbuf, errCloseCh)

	defer func(buffer *bytes.Buffer) {
		if err := logFn(ctx, e.deploymentID, e.NodeName, buffer); err != nil {
			log.Printf("Failed to publish Ansible log %v", err)
			log.Debugf("%+v", err)
		}
	}(&outbuf)
	if err := cmd.Run(); err != nil {
		return e.checkAnsibleRetriableError(ctx, err)
	}

	return nil
}
