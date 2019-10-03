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

package adapter

import (
	"encoding/json"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/prov/hostspool"
)

// AdaptedLocationType is the location target for adapter
const AdaptedLocationType = "hostspool"

type HostsPoolLocationAdapter struct {
	mgr hostspool.Manager
}

func NewHostsPoolLocationAdapter(client *api.Client) *HostsPoolLocationAdapter {
	return &HostsPoolLocationAdapter{hostspool.NewManager(client)}
}

func (a *HostsPoolLocationAdapter) SetLocationConfiguration(name string, props config.DynamicMap) error {
	hosts := make([]hostspool.Host, 0)
	var err error
	if props["hosts"] != nil {
		hosts, err = toHosts(props)
		if err != nil {
			return err
		}
	}

	// Check if location exists
	locations, err := a.mgr.ListLocations()
	var checkpoint uint64
	if collections.ContainsString(locations, name) {
		// Retrieve checkpoint to create or update hosts pool location
		_, _, checkpoint, err = a.mgr.List(name)
		if err != nil {
			return err
		}
	}

	// Apply config
	return a.mgr.Apply(name, hosts, &checkpoint)
}

func (a *HostsPoolLocationAdapter) GetLocationConfiguration(name string) (config.DynamicMap, error) {
	// Check if location exists
	locations, err := a.mgr.ListLocations()
	if err != nil {
		return nil, err
	}
	if !collections.ContainsString(locations, name) {
		return nil, errors.Errorf("No such location %q", name)
	}

	hostnames, _, _, err := a.mgr.List(name)
	if err != nil {
		return nil, err
	}

	hosts := make([]hostspool.HostConfig, 0)
	for _, hostname := range hostnames {
		host, err := a.mgr.GetHost(name, hostname)
		if err != nil {
			return nil, err
		}

		hosts = append(hosts, hostspool.HostConfig{Name: host.Name, Connection: host.Connection, Labels: host.Labels})
	}
	return toProperties(hostspool.PoolConfig{Hosts: hosts})
}

func toHosts(props config.DynamicMap) ([]hostspool.Host, error) {
	b, err := json.Marshal(props)
	if err != nil {
		return nil, err
	}

	var pool hostspool.Pool
	err = json.Unmarshal(b, &pool)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshall hostsPool from hostspool configuration with data:%q", string(b))
	}

	return pool.Hosts, nil
}

func toProperties(hostsPool hostspool.PoolConfig) (config.DynamicMap, error) {
	b, err := json.Marshal(hostsPool)
	if err != nil {
		return nil, err
	}

	var props config.DynamicMap
	err = json.Unmarshal(b, &props)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshall hostsPool from hostspool configuration with data:%q", string(b))
	}

	return props, nil
}
