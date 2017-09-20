package tosca

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestGroupedAssetsParallel(t *testing.T) {
	t.Run("groupAssets", func(t *testing.T) {
		t.Run("TestAssetNormativeParsing", assetNormativeParsing)
		t.Run("TestAssetJanusOpenStackParsing", assetJanusOpenStackParsing)
		t.Run("TestAssetJanusAwsParsing", assetJanusAwsParsing)
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

func assetJanusOpenStackParsing(t *testing.T) {
	t.Parallel()
	data, err := Asset("janus-openstack-types.yml")
	assert.Nil(t, err, "Can't load janus openstack types")
	assert.NotNil(t, data, "Can't load janus openstack types")
	var topo Topology

	err = yaml.Unmarshal(data, &topo)
	assert.Nil(t, err, "Can't parse janus openstack types")
}

func assetJanusAwsParsing(t *testing.T) {
	t.Parallel()
	data, err := Asset("janus-aws-types.yml")
	assert.Nil(t, err, "Can't load janus aws types")
	assert.NotNil(t, data, "Can't load janus aws types")
	var topo Topology

	err = yaml.Unmarshal(data, &topo)
	assert.Nil(t, err, "Can't parse janus aws types")
}
