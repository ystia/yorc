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
	"strings"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Testing a topology template defining capabilities substitution mappings
func testSubstitutionServiceCapabilityMappings(t *testing.T, kv *api.KV) {
	t.Parallel()

	// Storing the Deployment definition
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID,
		"testdata/test_topology_service.yml")
	require.NoError(t, err, "Failed to store test topology service deployment definition")

	mapping, err := getDeploymentSubstitutionMapping(kv, deploymentID)
	require.NoError(t, err, "Failed to get substitution mappings")

	assert.Equal(t, "org.ystia.yorc.test.pub.AppAType", mapping.NodeType,
		"Unexpected node type in substitution mappings")

	require.Equal(t, 2, len(mapping.Capabilities),
		"Wrong number of capability mappings")
	capName := "appA_capA"
	capMap := mapping.Capabilities[capName].Mapping
	require.Equal(t, 2, len(capMap), "Wrong number of elements in mapping %v", capMap)
	assert.Equal(t, "AppAInstance", capMap[0], "Wrong node template name in mapping %v", capMap)
	assert.Equal(t, capName, capMap[1], "Wrong capability name in mapping %v", capMap)

	props := mapping.Properties
	assert.Equal(t, 1, len(props), "Wrong number of elements in properties mapping %v", props)
	value := props["aProp"]
	require.Equal(t, 2, len(value.Mapping), "Wrong number of elements in properties mapping value %v", value)
	assert.Equal(t, "AppAInstance", value.Mapping[0], "Wrong node template name in properties mapping value %v", value)
	assert.Equal(t, "appA_propBString", value.Mapping[1], "Wrong property name in properties mapping value %v", value)

	attrs := mapping.Attributes
	require.Equal(t, 1, len(attrs), "Wrong number of elements in properties mapping %v", attrs)
	value = attrs["addrAttr"]
	require.Equal(t, 2, len(value.Mapping), "Wrong number of elements in attributes mapping value %v", value)
	assert.Equal(t, "AppAInstance", value.Mapping[0], "Wrong node template name in attributes mapping value %v", value)
	assert.Equal(t, "join_address", value.Mapping[1], "Wrong property name in attributes mapping value %v", value)

}

// Testing a topology template defining reauirements substitution mappings
func testSubstitutionServiceRequirementMappings(t *testing.T, kv *api.KV) {
	t.Parallel()

	// Storing the Deployment definition
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID,
		"testdata/test_topology_service.yml")
	require.NoError(t, err, "Failed to store test topology service deployment definition")

	mapping, err := getDeploymentSubstitutionMapping(kv, deploymentID)
	require.NoError(t, err, "Failed to get substitution mappings")

	assert.Equal(t, "org.ystia.yorc.test.pub.AppAType", mapping.NodeType,
		"Unexpected node type in substitution mappings")

	assert.Equal(t, 1, len(mapping.Requirements),
		"Wrong number of capability mappings")
	reqName := "hosted"
	reqMap := mapping.Requirements[reqName].Mapping
	assert.Equal(t, 2, len(reqMap), "Wrong number of elements in mapping %v", reqMap)
	assert.Equal(t, "AppAInstance", reqMap[0], "Wrong node template name in mapping %v", reqMap)
	assert.Equal(t, "hostedOnComputeHost", reqMap[1], "Wrong requirement name in mapping %v", reqMap)
}

// Testing the topology template of a client using a service
func testSubstitutionClientDirective(t *testing.T, kv *api.KV) {
	t.Parallel()

	// Storing the Deployment definition
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID,
		"testdata/test_topology_client_service.yml")
	require.NoError(t, err, "Failed to store test topology service deployment definition")

	serviceName := "AppAService"
	substitutable, err := isSubstitutableNode(kv, deploymentID, serviceName)
	require.NoError(t, err, "Failed to check substitutability of %s", serviceName)
	assert.True(t, substitutable, "Node template %s should be substitutable", serviceName)

	clientName := "AppBInstance"
	substitutable, err = isSubstitutableNode(kv, deploymentID, clientName)
	require.NoError(t, err, "Failed to check substitutability of %s", clientName)
	assert.False(t, substitutable, "Node template %s should not be substitutable", clientName)
}

// Testing requests on a service instance referenced by a client topology
func testSubstitutionClientServiceInstance(t *testing.T, kv *api.KV) {
	t.Parallel()

	// Storing the Deployment definition
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID,
		"testdata/test_topology_client_service.yml")
	require.NoError(t, err, "Failed to store test topology service deployment definition")

	serviceName := "AppAService"
	substitutable, err := isSubstitutableNode(kv, deploymentID, serviceName)
	require.NoError(t, err, "Failed to check substitutability of %s", serviceName)
	assert.True(t, substitutable, "Node template %s should be substitutable", serviceName)

	clientName := "AppBInstance"
	substitutable, err = isSubstitutableNode(kv, deploymentID, clientName)
	require.NoError(t, err, "Failed to check substitutability of %s", clientName)
	assert.False(t, substitutable, "Node template %s should not be substitutable", clientName)

	// Get the node type of the service node template
	// provided in an import generated from the Application Deployment
	nodeType, err := GetNodeType(kv, deploymentID, serviceName)
	require.NoError(t, err, "Failed to get the node type of service of %s", clientName)
	assert.Equal(t, "org.ystia.yorc.test.pub.AppAType", nodeType, "Wrong node type for service %s", serviceName)

	// Get instances of the service node template (fake instance)
	instances, err := GetNodeInstancesIds(kv, deploymentID, serviceName)
	require.NoError(t, err, "Failed to get service nod einstance id for %s", serviceName)

	assert.Equal(t, 1, len(instances), "Expected to get one node instance, go %v", instances)

	assert.Equal(t, substitutableNodeInstance, instances[0], "Unexpected instance for service %s", serviceName)

	// Get a capability attribute for the capability exposed by this service
	// here the ip_adress attribute exists as the capability derives from an
	// endpoint capability
	found, value, err := GetInstanceCapabilityAttribute(kv, deploymentID, serviceName,
		substitutableNodeInstance, "appA_capA", "ip_address")
	require.NoError(t, err, "Failed to get service capability attribute")
	assert.True(t, found, "Found no ip_address attribute in capability appA_capA exposed by the service")
	assert.Equal(t, "10.0.0.2", value, "Wrong ip_address attribute in capability appA_capA exposed by the service")
}
