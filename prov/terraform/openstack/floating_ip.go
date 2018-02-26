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
