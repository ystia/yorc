package openstack

import "fmt"

func (g *Generator) generateFloatingIP(url string) (string, error, bool) {
	var ret string
	var nodeType string
	var err error
	isIp := false
	if nodeType, err = g.getStringFormConsul(url, "type"); err != nil {
		return "", err, isIp
	}
	if nodeType != "janus.nodes.openstack.FloatingIP" {
		return "", fmt.Errorf("Unsupported node type for %s: %s", url, nodeType), isIp
	}
	if networkName, err := g.getStringFormConsul(url, "properties/floating_network_name"); err != nil {
		return "", err, isIp
	} else if networkName != "" {
		ret = networkName
	} else if ip, err :=  g.getStringFormConsul(url, "properties/ip"); err != nil {
		return "", err, isIp
	} else if ip != "" {
		ret = ip
		isIp = true
	} else {
		return "", fmt.Errorf("A network name or IP need to be provided"), isIp
	}

	return ret, nil, isIp
}