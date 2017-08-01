package kubernetes

import (
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"path"
	"testing"
)

func TestGenerator(t *testing.T) {
	t.Run("kubernetes", func(t *testing.T) {
		t.Run("newGenerator", createSecretRepoTest)
		t.Run("newGeneratorDeployment", generateDeploymentTest)
		t.Run("newGeneratorLimitResources", generateLimitResourcesTest)
	})
}

func createSecretRepoTest(t *testing.T) {
	t.Parallel()
	var clientset *kubernetes.Clientset
	conf, err := clientcmd.BuildConfigFromFlags("test", "")
	clientset, err = kubernetes.NewForConfig(conf)

	consulConfig := api.DefaultConfig()

	client, err := api.NewClient(consulConfig)
	require.Nil(t, err)

	kv := client.KV()

	myGenerator := NewGenerator(kv, config.Configuration{})

	secret := myGenerator.GenerateNewRepoSecret(clientset, "myRepo", []byte{})
	assert.Equal(t, secret.Name, "myRepo")
	assert.Equal(t, secret.Data[v1.DockerConfigKey], []byte{})
}

func generateDeploymentTest(t *testing.T) {
	t.Parallel()
	conf, err := clientcmd.BuildConfigFromFlags("test", "")
	_, err = kubernetes.NewForConfig(conf)

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

	myGenerator := NewGenerator(kv, config.Configuration{})

	deploymentID := "d1"
	nodeName := "NodeA"
	nodeTypeName := "janus.types.A"
	operation := "tosca.interfaces.node.lifecycle.standard.start"

	srv1.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/start/implementation/file"): []byte("test"),
	})

	_, _, err = myGenerator.GenerateDeployment(deploymentID, nodeName, operation, nodeTypeName, "test", nil, 1)

	assert.Nil(t, err)
}


func generateLimitResourcesTest(t *testing.T) {
	t.Parallel()

	resourceList, err := generateLimitsRessources("", "")
	require.Nil(t, err)
	require.Nil(t, resourceList)


	resourceList, err = generateLimitsRessources("", "1M")
	require.Nil(t, err)
	require.NotNil(t, resourceList)
	i, _ := resourceList.Cpu().AsInt64()
	require.Equal(t, i, int64(0))

	resourceList, err = generateLimitsRessources("1M", "")
	require.Nil(t, err)
	require.NotNil(t, resourceList)
	i, _ = resourceList.Memory().AsInt64()
	require.Equal(t, i, int64(0))

	resourceList, err = generateLimitsRessources("qsd", "1M")
	require.Nil(t, resourceList)
	require.Error(t, err)

	resourceList, err = generateLimitsRessources("1M", "qsd")
	require.Nil(t, resourceList)
	require.Error(t, err)

	resourceList, err = generateLimitsRessources("2M", "1M")
	require.Nil(t, err)
	require.NotNil(t, resourceList.Cpu())
	i, _ = resourceList.Cpu().AsInt64()
	require.Equal(t, i, int64(2000000))
	require.NotNil(t, resourceList.Memory())
	i, _ = resourceList.Memory().AsInt64()
	require.Equal(t, i, int64(1000000))

	resourceList, err = generateLimitsRessources("0", "0")
	require.Nil(t, err)
	require.NotNil(t, resourceList.Cpu())
	i, _ = resourceList.Cpu().AsInt64()
	require.Equal(t, i, int64(0))
	require.NotNil(t, resourceList.Memory())
	i, _ = resourceList.Memory().AsInt64()
	require.Equal(t, i, int64(0))
}