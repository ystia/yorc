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

// LocationAdapter interface represents the behaviour of a specific adapter for locations management
type LocationAdapter interface {
	CreateLocationConfiguration(locationName string, props config.DynamicMap) error
	GetLocationConfiguration(locationName string) (config.DynamicMap, error)
	SetLocationConfiguration(locationName string, props config.DynamicMap) error
	RemoveLocation(locationName string) error
	GetLocations() (map[string]config.DynamicMap, error)
	ListLocations() ([]string, error)
}

type hostsPoolLocationAdapter struct {
	mgr hostspool.Manager
}

// NewHostsPoolLocationAdapter allows to create a new hostsPool location adapter
func NewHostsPoolLocationAdapter(client *api.Client) LocationAdapter {
	return &hostsPoolLocationAdapter{hostspool.NewManager(client)}
}

// NewHostsPoolLocationAdapterWithSSHFactory creates a Location Adapter with a given ssh factory
//
// Currently this is used for testing purpose to mock the ssh connection.
func NewHostsPoolLocationAdapterWithSSHFactory(client *api.Client, sshClientFactory hostspool.SSHClientFactory) LocationAdapter {
	return &hostsPoolLocationAdapter{hostspool.NewManagerWithSSHFactory(client, sshClientFactory)}
}

// CreateLocationConfiguration allows to create a location configuration of hosts pool type
func (a *hostsPoolLocationAdapter) CreateLocationConfiguration(locationName string, props config.DynamicMap) error {
	// Check if location exists
	locations, err := a.mgr.ListLocations()
	if err != nil {
		return err
	}
	if collections.ContainsString(locations, locationName) {
		return errors.Errorf("location %q already exists", locationName)
	}
	return a.setLocationConfiguration(locationName, props, true)
}

// SetLocationConfiguration allows to set (create or update) a location configuration of hosts pool type
func (a *hostsPoolLocationAdapter) SetLocationConfiguration(locationName string, props config.DynamicMap) error {
	return a.setLocationConfiguration(locationName, props, false)
}

func (a *hostsPoolLocationAdapter) setLocationConfiguration(locationName string, props config.DynamicMap, create bool) error {
	hosts := make([]hostspool.Host, 0)
	var err error
	if props["hosts"] != nil {
		hosts, err = toHosts(props)
		if err != nil {
			return err
		}
	} else {
		return errors.WithStack(badRequestError{})
	}

	var checkpoint uint64
	if !create {
		// Check if location exists
		locations, err := a.mgr.ListLocations()
		if collections.ContainsString(locations, locationName) {
			// Retrieve checkpoint to create or update hosts pool location
			_, _, checkpoint, err = a.mgr.List(locationName)
			if err != nil {
				return err
			}
		}
	}

	// Apply config
	return a.mgr.Apply(locationName, hosts, &checkpoint)
}

// GetLocationConfiguration allows to get a location configuration of hosts pool type
func (a *hostsPoolLocationAdapter) GetLocationConfiguration(locationName string) (config.DynamicMap, error) {
	// Check if location exists
	locations, err := a.mgr.ListLocations()
	if err != nil {
		return nil, err
	}
	if !collections.ContainsString(locations, locationName) {
		return nil, errors.Errorf("No such location %q", locationName)
	}

	hostnames, _, _, err := a.mgr.List(locationName)
	if err != nil {
		return nil, err
	}

	hosts := make([]hostspool.HostConfig, 0)
	for _, hostname := range hostnames {
		host, err := a.mgr.GetHost(locationName, hostname)
		if err != nil {
			return nil, err
		}

		hosts = append(hosts, hostspool.HostConfig{Name: host.Name, Connection: host.Connection, Labels: host.Labels})
	}
	return toProperties(hostspool.PoolConfig{Hosts: hosts})
}

// RemoveLocation allows to remove a location of hosts pool type
func (a *hostsPoolLocationAdapter) RemoveLocation(locationName string) error {
	return a.mgr.RemoveLocation(locationName)
}

// GetLocations allows to retrieve all hosts pool locations
func (a *hostsPoolLocationAdapter) GetLocations() (map[string]config.DynamicMap, error) {
	locations, err := a.mgr.ListLocations()
	if err != nil {
		return nil, err
	}

	results := make(map[string]config.DynamicMap)
	for _, location := range locations {
		lconfig, err := a.GetLocationConfiguration(location)
		if err != nil {
			return nil, err
		}
		results[location] = lconfig
	}

	return results, err
}

// ListLocations allows to list all locations
func (a *hostsPoolLocationAdapter) ListLocations() ([]string, error) {
	locations, err := a.mgr.ListLocations()
	if err != nil {
		return nil, err
	}
	return locations, nil
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
