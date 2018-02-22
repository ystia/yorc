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
