package operations

import (
	"github.com/stretchr/testify/require"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/hashicorp/consul/testutil"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"github.com/hashicorp/consul/api"
	"path"
)



func TestOperation(t *testing.T) {
	t.Run("operation", func(t *testing.T) {
		t.Run("IsTargetOperationTest", IsTargetOperationTest)
		t.Run("IsOperationNotImplemented", IsOperationNotImplementedTest)
		t.Run("ResolveIsPerInstanceOperationTest", ResolveIsPerInstanceOperationTest)
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
