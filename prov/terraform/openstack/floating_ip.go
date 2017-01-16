package openstack

import "fmt"

type IP struct {
	Name string
	Pool string
	IsIP bool
}

func (g *Generator) generateFloatingIP(url, instanceName string) (IP, error) {
	var nodeType string
	result := IP{}
	result.IsIP = false
	var err error
	if nodeType, err = g.getStringFormConsul(url, "type"); err != nil {
		return IP{}, err
	}
	if nodeType != "janus.nodes.openstack.FloatingIP" {
		return IP{}, fmt.Errorf("Unsupported node type for %s: %s", url, nodeType)
	}
	nodeName, err := g.getStringFormConsul(url, "name")
	if err != nil {
		return IP{}, err
	}
	result.Name = nodeName + "-" + instanceName
	if ip, err := g.getStringFormConsul(url, "properties/ip"); err != nil {
		return IP{}, err
	} else if ip != "" {
		result.Pool = ip
		result.IsIP = true
	} else if networkName, err := g.getStringFormConsul(url, "properties/floating_network_name"); err != nil {
		return IP{}, err
	} else if networkName != "" {
		result.Pool = networkName
	} else {
		return IP{}, fmt.Errorf("A network name or IP need to be provided")
	}

	return result, nil
}
