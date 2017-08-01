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

	client, err := api.NewClient(consulConfig)
	require.Nil(t, err)

	kv := client.KV()

	myGenerator := NewGenerator(kv, config.Configuration{})

	deploymentID := "d1"
	nodeName := "NodeA"
	nodeTypeName := "janus.types.A"
	operation := "tosca.interfaces.node.lifecycle.standard.start"

	srv1.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/implementation/file"): []byte("test"),
	})

	_, _, err = myGenerator.GenerateDeployment(deploymentID, nodeName, operation, nodeTypeName, "test", nil, 1)

	assert.Nil(t, err)
}
