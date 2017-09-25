package deployments

import (
	"context"
	"testing"

	yaml "gopkg.in/yaml.v2"

	"path"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/testutil"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

func testResolver(t *testing.T, kv *api.KV) {
	log.SetDebug(true)

	t.Run("deployments/resolver/testGetOperationOutput", func(t *testing.T) {
		testGetOperationOutput(t, kv)
	})
	t.Run("deployments/resolver/testGetOperationOutputReal", func(t *testing.T) {
		testGetOperationOutputReal(t, kv)
	})
}

func generateToscaValueAssignmentFromString(t *testing.T, valueAssignment string) *tosca.ValueAssignment {
	va := &tosca.ValueAssignment{}

	err := yaml.Unmarshal([]byte(valueAssignment), va)
	require.Nil(t, err)
	require.NotNil(t, va.Expression)
	return va
}

func testGetOperationOutput(t *testing.T, kv *api.KV) {
	// t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/get_op_output.yaml")
	require.Nil(t, err, "Failed to parse testdata/get_op_output.yaml definition")

	_, err = kv.Put(&api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/GetOPOutputsNode/0/outputs/standard/configure/MY_OUTPUT"), Value: []byte("MY_RESULT")}, nil)
	require.Nil(t, err)
	r := NewResolver(kv, deploymentID)

	result, err := r.ResolveValueAssignmentForNode(generateToscaValueAssignmentFromString(t, `{ get_operation_output: [ SELF, Standard, configure, MY_OUTPUT ] }`), "GetOPOutputsNode", "0")
	require.Nil(t, err)
	require.Equal(t, "MY_RESULT", result)

	_, err = kv.Put(&api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances/GetOPOutputsNode/janus.tests.relationships.GetOPOutputsRel/0/outputs/configure/pre_configure_source/PARTITION_NAME"), Value: []byte("part1")}, nil)
	require.Nil(t, err)
	result, err = r.ResolveValueAssignmentForRelationship(generateToscaValueAssignmentFromString(t, `{ get_operation_output: [ SELF, Configure, pre_configure_source, PARTITION_NAME ] }`), "GetOPOutputsNode", "BS", "0", "0")
	require.Nil(t, err, "%+v", err)
	require.Equal(t, "part1", result)

	result, err = r.ResolveValueAssignmentForNode(generateToscaValueAssignmentFromString(t, `{ get_attribute: [ SELF, partition_name ] }`), "GetOPOutputsNode", "0")
	require.Nil(t, err, "%+v", err)
	require.Equal(t, "part1", result)

	_, err = kv.Put(&api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances/GetOPOutputsNodeFirstReq/janus.tests.relationships.GetOPOutputsRel/0/outputs/configure/pre_configure_source/PARTITION_NAME"), Value: []byte("part2")}, nil)
	require.Nil(t, err)
	result, err = r.ResolveValueAssignmentForNode(generateToscaValueAssignmentFromString(t, `{ get_attribute: [ SELF, partition_name ] }`), "GetOPOutputsNodeFirstReq", "0")
	require.Nil(t, err, "%+v", err)
	require.Equal(t, "part2", result)

	_, err = kv.Put(&api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances/GetOPOutputsNodeSecondReq/janus.tests.relationships.GetOPOutputsRel/0/outputs/configure/pre_configure_source/PARTITION_NAME"), Value: []byte("part3")}, nil)
	require.Nil(t, err)
	result, err = r.ResolveValueAssignmentForNode(generateToscaValueAssignmentFromString(t, `{ get_attribute: [ SELF, partition_name ] }`), "GetOPOutputsNodeSecondReq", "0")
	require.Nil(t, err, "%+v", err)
	require.Equal(t, "part3", result)

}

func testGetOperationOutputReal(t *testing.T, kv *api.KV) {
	// t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/get_op_output_real.yaml")
	require.Nil(t, err, "Failed to parse testdata/get_op_output_real.yaml definition: %+v", err)

	r := NewResolver(kv, deploymentID)

	_, err = kv.Put(&api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances/PublisherFromDockerVolume/starlings.relationships.DependsOnDockerVolume/0/outputs/configure/post_configure_target/HOST_PATH"), Value: []byte("/mypath")}, nil)
	require.Nil(t, err)

	result, err := r.ResolveValueAssignmentForNode(generateToscaValueAssignmentFromString(t, `{ get_attribute: [ SELF, host_path ] }`), "PublisherFromDockerVolume", "0")
	require.Nil(t, err)
	require.Equal(t, "/mypath", result)

}
