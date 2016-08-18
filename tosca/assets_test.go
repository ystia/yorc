package tosca

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestAssetNormativeParsing(t *testing.T) {
	data, err := Asset("normative-types.yml")
	assert.Nil(t, err, "Can't load normative types")
	assert.NotNil(t, data, "Can't load normative types")
	var topo Topology

	err = yaml.Unmarshal(data, &topo)
	assert.Nil(t, err, "Can't parse normative types")
}

func TestAssetJanusOpenStackParsing(t *testing.T) {
	data, err := Asset("janus-openstack-types.yml")
	assert.Nil(t, err, "Can't load janus openstack types")
	assert.NotNil(t, data, "Can't load janus openstack types")
	var topo Topology

	err = yaml.Unmarshal(data, &topo)
	assert.Nil(t, err, "Can't parse janus openstack types")
}
