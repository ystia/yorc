package operations

import (
	"github.com/stretchr/testify/require"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/hashicorp/consul/testutil"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"github.com/hashicorp/consul/api"
	"path"
	"novaforge.bull.com/starlings-janus/janus/prov"
	"novaforge.bull.com/starlings-janus/janus/deployments"
)



func TestOperation(t *testing.T) {
	t.Run("operation", func(t *testing.T) {
		t.Run("IsTargetOperationTest", IsTargetOperationTest)
		t.Run("IsOperationNotImplemented", IsOperationNotImplementedTest)
		t.Run("ResolveIsPerInstanceOperationTest", ResolveIsPerInstanceOperationTest)
		t.Run("testInputs", testInputs)
	})
}


func IsTargetOperationTest(t *testing.T) {
	t.Parallel()

	res := IsTargetOperation("tosca.interfaces.node.lifecycle.Configure.pre_configure_target")
	require.True(t, res)

	res = IsTargetOperation("tosca.interfaces.node.lifecycle.Configure.add_source")
	require.True(t, res)

	res = IsTargetOperation("tosca.interfaces.node.lifecycle.Configure.pre_configure_source")
	require.False(t, res)
}

func IsOperationNotImplementedTest(t *testing.T) {
	t.Parallel()
	var err operationNotImplemented
	res := IsOperationNotImplemented(err)
	assert.True(t, res)
}

func ResolveIsPerInstanceOperationTest(t *testing.T) {
	t.Parallel()
	srv1, err := testutil.NewTestServer()
	if err != nil {
		t.Fatalf("Failed to create consul server: %v", err)
	}
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	require.Nil(t, err)

	kv := client.KV()

	deploymentID := "d1"
	nodeName := "NodeA"
	nodeTypeName := "janus.types.A"
	operation1 := "tosca.interfaces.node.lifecycle.standard.start"
	operation2 := "tosca.interfaces.node.lifecycle.standard.add_target"

	srv1.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/start/implementation/file"): []byte("test"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements/0/relationship"): []byte("tosca.relationships.HostedOn"),
	})

	res, err := ResolveIsPerInstanceOperation(operation1, deploymentID, "typeA", kv)
	assert.Nil(t, err)
	assert.False(t, res)

	res, err = ResolveIsPerInstanceOperation(operation2, deploymentID, "tosca.relationships.HostedOn", kv)
	assert.Nil(t, err)
	assert.False(t, res)
}

func getOperation(kv *api.KV, deploymentID, nodeName, operationName string) (prov.Operation, error) {
	isRelationshipOp, operationRealName, requirementIndex, targetNodeName, err := deployments.DecodeOperation(kv, deploymentID, nodeName, operationName)
	if err != nil {
		return prov.Operation{}, err
	}
	implArt, err := deployments.GetImplementationArtifactForOperation(kv, deploymentID, nodeName, operationRealName, isRelationshipOp, requirementIndex)
	if err != nil {
		return prov.Operation{}, err
	}
	op := prov.Operation{
		Name: operationRealName,
		ImplementationArtifact: implArt,
		RelOp: prov.RelationshipOperation{
			IsRelationshipOperation: isRelationshipOp,
			RequirementIndex:        requirementIndex,
			TargetNodeName:          targetNodeName,
		},
	}
	return op, nil
}

func testInputs(t *testing.T) {
	t.Parallel()
	srv1, err := testutil.NewTestServer()
	if err != nil {
		t.Fatalf("Failed to create consul server: %v", err)
	}
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	require.Nil(t, err)

	kv := client.KV()
	deploymentID := "d1"
	nodeName := "NodeA"
	nodeTypeName := "janus.types.A"
	operation := "tosca.interfaces.node.lifecycle.standard.create"

	srv1.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/implementation_artifacts_extensions/sh"):         []byte("tosca.artifacts.Implementation.Bash"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types/tosca.artifacts.Implementation.Bash/name"): []byte("tosca.artifacts.Implementation.Bash"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "name"):                                                        []byte(nodeTypeName),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A1/name"):                   []byte("A1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A1/expression"):             []byte("get_property: [SELF, document_root]"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A3/name"):                   []byte("A3"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A3/expression"):             []byte("get_property: [SELF, empty]"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A2/name"):                   []byte("A2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A2/expression"):             []byte("get_attribute: [HOST, ip_address]"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/implementation/primary"):           []byte("/tmp/create.sh"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/implementation/dependencies"):      []byte(""),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A1/is_property_definition"): []byte("false"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A3/is_property_definition"): []byte("false"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A2/is_property_definition"): []byte("false"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types/tosca.nodes.Compute/name"): []byte("tosca.nodes.Compute"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "type"): []byte(nodeTypeName),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "properties/document_root"):    []byte("/var/www"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "properties/empty"):            []byte(""),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements/0/capability"):   []byte("tosca.capabilities.Container"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements/0/name"):         []byte("host"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements/0/node"):         []byte("Compute"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements/0/relationship"): []byte("tosca.relationships.HostedOn"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes/Compute/type"):                             []byte("tosca.nodes.Compute"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/Compute/0/attributes/ip_address"):      []byte("10.10.10.1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/Compute/1/attributes/ip_address"):      []byte("10.10.10.2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/Compute/2/attributes/ip_address"):      []byte("10.10.10.3"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/Compute/0/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/Compute/1/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/Compute/2/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.3"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, "0/attributes/state"): []byte("initial"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, "1/attributes/state"): []byte("initial"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, "2/attributes/state"): []byte("initial"),
	})

	//InputsResolver(kv *api.KV, operationPath, deploymentID, nodeName, taskID, operation string)
	op, err := getOperation(kv, deploymentID, nodeName, operation)
	require.Nil(t, err)

	_,_, err = InputsResolver(kv,path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create"), deploymentID, nodeName, "",op.Name )
	assert.Nil(t, err)
}
