package deployments

import (
	"context"
	"testing"

	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

func TestDefinitionStore(t *testing.T) {
	//log.SetDebug(true)
	srv1, err := testutil.NewTestServer()
	require.Nil(t, err)
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	require.Nil(t, err)

	kv := client.KV()
	consulutil.InitConsulPublisher(config.DefaultConsulPubMaxRoutines, kv)
	t.Run("deployments/definition_store", func(t *testing.T) {
		t.Run("implementationArtifacts", func(t *testing.T) {
			testImplementationArtifacts(t, kv)
		})
		t.Run("implementationArtifactsDuplicates", func(t *testing.T) {
			testImplementationArtifactsDuplicates(t, kv)
		})
	})
}

func testImplementationArtifacts(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/get_op_output.yaml")
	require.Nil(t, err, "Failed to parse testdata/get_op_output.yaml definition")

	impl, err := GetImplementationArtifactForExtension(kv, deploymentID, "sh")
	require.Nil(t, err)
	require.Equal(t, "tosca.artifacts.Implementation.Bash", impl)

	impl, err = GetImplementationArtifactForExtension(kv, deploymentID, "SH")
	require.Nil(t, err)
	require.Equal(t, "tosca.artifacts.Implementation.Bash", impl)

	impl, err = GetImplementationArtifactForExtension(kv, deploymentID, "py")
	require.Nil(t, err)
	require.Equal(t, "tosca.artifacts.Implementation.Python", impl)

	impl, err = GetImplementationArtifactForExtension(kv, deploymentID, "Py")
	require.Nil(t, err)
	require.Equal(t, "tosca.artifacts.Implementation.Python", impl)

	impl, err = GetImplementationArtifactForExtension(kv, deploymentID, "yaml")
	require.Nil(t, err)
	require.Equal(t, "tosca.artifacts.Implementation.Ansible", impl)

	impl, err = GetImplementationArtifactForExtension(kv, deploymentID, "yml")
	require.Nil(t, err)
	require.Equal(t, "tosca.artifacts.Implementation.Ansible", impl)

}

func testImplementationArtifactsDuplicates(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/artifacts_ext_duplicate.yaml")
	require.Error(t, err, "Expecting for a duplicate extension for artifact implementation")

}
