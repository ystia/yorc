package testutil

import (
	"strings"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	//"novaforge.bull.com/starlings-janus/janus/deployments"
	//"github.com/stretchr/testify/require"
	//"context"
)

// NewTestConsulInstance allows to :
//  - creates and returns a new Consul server and client
//  - starts a Consul Publisher
// Warning: You need to defer the server stop command in the caller
func NewTestConsulInstance(t *testing.T) (*testutil.TestServer, *api.Client) {
	srv1, err := testutil.NewTestServer()
	if err != nil {
		t.Fatalf("Failed to create consul server: %v", err)
	}

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	assert.Nil(t, err)

	kv := client.KV()
	consulutil.InitConsulPublisher(config.DefaultConsulPubMaxRoutines, kv)
	return srv1, client
}

// Create a deploymentID from the test name value
func BuildDeploymentID(t *testing.T, kv *api.KV) string {
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	//yamlName := "testdata/" + deploymentID + ".yaml"
	//err := deployments.StoreDeploymentDefinition(context.Background(), kv, deploymentID, yamlName)
	//require.Nil(t, err, "Failed to parse "+yamlName+" definition")
	return deploymentID
}
