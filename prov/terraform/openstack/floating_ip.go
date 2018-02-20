package openstack

import (
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
)

// An IP is...
// TODO consider refactoring this as it is not clear
type IP struct {
	Name string
	Pool string
	IsIP bool
}

func (g *osGenerator) generateFloatingIP(kv *api.KV, url, instanceName string) (IP, error) {
	var nodeType string
	result := IP{}
	result.IsIP = false
	var err error
	if nodeType, err = g.getStringFormConsul(kv, url, "type"); err != nil {
		return IP{}, err
	}
	if nodeType != "yorc.nodes.openstack.FloatingIP" {
		return IP{}, errors.Errorf("Unsupported node type for %s: %s", url, nodeType)
	}
	nodeName, err := g.getStringFormConsul(kv, url, "name")
	if err != nil {
		return IP{}, err
	}
	result.Name = nodeName + "-" + instanceName
	if ip, err := g.getStringFormConsul(kv, url, "properties/ip"); err != nil {
		return IP{}, err
	} else if ip != "" {
		result.Pool = ip
		result.IsIP = true
	} else if networkName, err := g.getStringFormConsul(kv, url, "properties/floating_network_name"); err != nil {
		return IP{}, err
	} else if networkName != "" {
		result.Pool = networkName
	} else {
		return IP{}, errors.Errorf("A network name or IP need to be provided")
	}

	return result, nil
}
