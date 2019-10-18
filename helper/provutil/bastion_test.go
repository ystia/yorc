// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package provutil

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/ystia/yorc/v4/deployments"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var expectedKey []byte

func init() {
	var err error
	expectedKey, err = ioutil.ReadFile("./testdata/validkey.pem")
	if err != nil {
		panic(err)
	}
}

func testBastionEndpoint(t *testing.T, kv *api.KV) {
	deploymentID := loadTestYaml(t, kv)

	b, err := GetInstanceBastionHost(context.Background(), kv, deploymentID, "ComputePassword")
	require.Nil(t, err)
	if assert.NotNil(t, b, "should return bastion host configuration") {
		assert.Equal(t, "10.0.0.2", b.Host)
		assert.Equal(t, "22", b.Port, "port should default to 22")
		assert.Equal(t, "ubuntu", b.User)
		assert.Equal(t, "secret", b.Password)
		assert.Empty(t, b.PrivateKeys)
	}

	b, err = GetInstanceBastionHost(context.Background(), kv, deploymentID, "ComputeKey")
	require.Nil(t, err)
	if assert.NotNil(t, b, "should return bastion host configuration") {
		assert.Equal(t, "10.0.0.2", b.Host)
		assert.Equal(t, "8022", b.Port)
		assert.Equal(t, "ubuntu", b.User)
		assert.Equal(t, "", b.Password, "shouldn't contain password when TokenType != password")
		assert.Equal(t, expectedKey, b.PrivateKeys["0"].Content)
	}

	b, err = GetInstanceBastionHost(context.Background(), kv, deploymentID, "ComputeNoBastion")

	require.Nil(t, err)
	assert.Nil(t, b, "shouldn't return a bastion host configuration when use == false")
}

func testBastionEndpointPriority(t *testing.T, kv *api.KV) {
	deploymentID := loadTestYaml(t, kv)
	deployments.SetAttributeForAllInstances(kv, deploymentID, "Bastion", "ip_address", "10.0.0.3")

	b, err := GetInstanceBastionHost(context.Background(), kv, deploymentID, "Compute")

	require.Nil(t, err)
	if assert.NotNil(t, b, "should return bastion host configuration") {
		assert.Equal(t, "10.0.0.2", b.Host)
		assert.Equal(t, "22", b.Port)
		assert.Equal(t, "ubuntu", b.User)
		assert.Equal(t, "secret", b.Password)
		assert.Empty(t, b.PrivateKeys)
	}
}

func testBastionNotDefined(t *testing.T, kv *api.KV) {
	deploymentID := loadTestYaml(t, kv)

	bast, err := GetInstanceBastionHost(context.Background(), kv, deploymentID, "Compute")

	require.Nil(t, err)
	assert.Nil(t, bast, "shouldn't return a bastion host configuration")
}

func testBastionRelationship(t *testing.T, kv *api.KV) {
	deploymentID := loadTestYaml(t, kv)
	deployments.SetAttributeForAllInstances(kv, deploymentID, "BastionKey", "ip_address", "10.0.0.3")
	deployments.SetAttributeForAllInstances(kv, deploymentID, "BastionPassword", "ip_address", "10.0.0.4")

	b, err := GetInstanceBastionHost(context.Background(), kv, deploymentID, "ComputePassword")

	require.Nil(t, err)
	if assert.NotNil(t, b, "should return bastion host configuration") {
		assert.Equal(t, "10.0.0.4", b.Host)
		assert.Equal(t, "22", b.Port, "port should default to 22")
		assert.Equal(t, "ubuntu", b.User)
		assert.Equal(t, "secret", b.Password)
	}

	b, err = GetInstanceBastionHost(context.Background(), kv, deploymentID, "ComputeKey")

	require.Nil(t, err)
	if assert.NotNil(t, b, "should return bastion host configuration") {
		assert.Equal(t, "10.0.0.3", b.Host)
		assert.Equal(t, "8022", b.Port)
		assert.Equal(t, "ubuntu", b.User)
		assert.Equal(t, "", b.Password)
		assert.Equal(t, expectedKey, b.PrivateKeys["0"].Content)
	}
}
