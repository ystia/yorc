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

package tosca

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestGroupedAssetsParallel(t *testing.T) {
	t.Run("groupAssets", func(t *testing.T) {
		t.Run("TestAssetNormativeParsing", assetNormativeParsing)
		t.Run("TestAssetYorcOpenStackParsing", assetYorcOpenStackParsing)
		t.Run("TestAssetYorcAwsParsing", assetYorcAwsParsing)
	})
}

func assetNormativeParsing(t *testing.T) {
	t.Parallel()
	data, err := Asset("normative-types.yml")
	assert.Nil(t, err, "Can't load normative types")
	assert.NotNil(t, data, "Can't load normative types")
	var topo Topology

	err = yaml.Unmarshal(data, &topo)
	assert.Nil(t, err, "Can't parse normative types")
}

func assetYorcOpenStackParsing(t *testing.T) {
	t.Parallel()
	data, err := Asset("yorc-openstack-types.yml")
	assert.Nil(t, err, "Can't load yorc openstack types")
	assert.NotNil(t, data, "Can't load yorc openstack types")
	var topo Topology

	err = yaml.Unmarshal(data, &topo)
	assert.Nil(t, err, "Can't parse yorc openstack types")
}

func assetYorcAwsParsing(t *testing.T) {
	t.Parallel()
	data, err := Asset("yorc-aws-types.yml")
	assert.Nil(t, err, "Can't load yorc aws types")
	assert.NotNil(t, data, "Can't load yorc aws types")
	var topo Topology

	err = yaml.Unmarshal(data, &topo)
	assert.Nil(t, err, "Can't parse yorc aws types")
}
