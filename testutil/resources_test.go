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

package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/ystia/yorc/v3/registry"
	"github.com/ystia/yorc/v3/tosca"
)

func TestAssets(t *testing.T) {
	return
	NewTestConsulInstance(t)
	t.Run("TestAssets", func(t *testing.T) {
		t.Run("NormativeTypes", testAssetNormativeParsing)
		t.Run("OpenstackTypes", testAssetYorcOpenStackParsing)
		t.Run("AWSTypes", testAssetYorcAwsParsing)
		t.Run("GoogleTypes", testAssetYorcGoogleParsing)
		t.Run("YorcTypes", testAssetYorcParsing)
		t.Run("SlurmTypes", testAssetYorcSlurmParsing)
		t.Run("HostsPoolTypes", testAssetYorcHostsPoolParsing)
		t.Run("DockerTypes", testAssetYorcDockerParsing)
	})
}

func testAssetNormativeParsing(t *testing.T) {
	t.Parallel()
	reg := registry.GetRegistry()
	data, err := reg.GetToscaDefinition("normative-types.yml")
	assert.Nil(t, err, "Can't load normative types")
	assert.NotNil(t, data, "Can't load normative types")
	var topo tosca.Topology

	err = yaml.Unmarshal(data, &topo)
	assert.Nil(t, err, "Can't parse normative types")
}

func testAssetYorcOpenStackParsing(t *testing.T) {
	t.Parallel()
	reg := registry.GetRegistry()
	data, err := reg.GetToscaDefinition("yorc-openstack-types.yml")
	assert.Nil(t, err, "Can't load yorc openstack types")
	assert.NotNil(t, data, "Can't load yorc openstack types")
	var topo tosca.Topology

	err = yaml.Unmarshal(data, &topo)
	assert.Nil(t, err, "Can't parse yorc openstack types")
}

func testAssetYorcAwsParsing(t *testing.T) {
	t.Parallel()
	reg := registry.GetRegistry()
	data, err := reg.GetToscaDefinition("yorc-aws-types.yml")
	assert.Nil(t, err, "Can't load yorc aws types")
	assert.NotNil(t, data, "Can't load yorc aws types")
	var topo tosca.Topology

	err = yaml.Unmarshal(data, &topo)
	assert.Nil(t, err, "Can't parse yorc aws types")
}

func testAssetYorcGoogleParsing(t *testing.T) {
	t.Parallel()
	reg := registry.GetRegistry()
	data, err := reg.GetToscaDefinition("yorc-google-types.yml")
	require.NoError(t, err, "Can't load yorc google types")
	assert.NotNil(t, data, "Can't load yorc google types")
	var topo tosca.Topology

	err = yaml.Unmarshal(data, &topo)
	assert.Nil(t, err, "Can't parse yorc google types")
}

func testAssetYorcParsing(t *testing.T) {
	t.Parallel()
	reg := registry.GetRegistry()
	data, err := reg.GetToscaDefinition("yorc-types.yml")
	require.NoError(t, err, "Can't load yorc types")
	assert.NotNil(t, data, "Can't load yorc types")
	var topo tosca.Topology

	err = yaml.Unmarshal(data, &topo)
	assert.Nil(t, err, "Can't parse yorc types")
}

func testAssetYorcSlurmParsing(t *testing.T) {
	t.Parallel()
	reg := registry.GetRegistry()
	data, err := reg.GetToscaDefinition("yorc-slurm-types.yml")
	require.NoError(t, err, "Can't load yorc Slurm types")
	assert.NotNil(t, data, "Can't load yorc Slurm types")
	var topo tosca.Topology

	err = yaml.Unmarshal(data, &topo)
	assert.Nil(t, err, "Can't parse yorc Slurm types")
}

func testAssetYorcHostsPoolParsing(t *testing.T) {
	t.Parallel()
	reg := registry.GetRegistry()
	data, err := reg.GetToscaDefinition("yorc-hostspool-types.yml")
	require.NoError(t, err, "Can't load yorc hostspool types")
	assert.NotNil(t, data, "Can't load yorc hostspool types")
	var topo tosca.Topology

	err = yaml.Unmarshal(data, &topo)
	assert.Nil(t, err, "Can't parse yorc hostspool types")
}

func testAssetYorcDockerParsing(t *testing.T) {
	t.Parallel()
	reg := registry.GetRegistry()
	data, err := reg.GetToscaDefinition("yorc-docker-types.yml")
	require.NoError(t, err, "Can't load yorc docker types")
	assert.NotNil(t, data, "Can't load yorc docker types")
	var topo tosca.Topology

	err = yaml.Unmarshal(data, &topo)
	assert.Nil(t, err, "Can't parse yorc docker types")
}
