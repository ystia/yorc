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

// Package locations is responsible for handling locations where deployments
// can take place
package locations

import (
	"encoding/json"
	"github.com/ystia/yorc/v4/locations/adapter"
	"path"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tosca"
)

// LocationConfiguration holds a location configuration
type LocationConfiguration struct {
	Name       string            `yaml:"name" mapstructure:"name"`
	Type       string            `yaml:"type,omitempty" mapstructure:"type"` // not an enum as it could be extended by plugins
	Properties config.DynamicMap `yaml:"properties,omitempty" mapstructure:"properties"`
}

// A Manager is in charge of creating/updating/deleting locations from the pool
type Manager interface {
	InitializeLocations(locationFilePath string) (bool, error)
	CreateLocation(lConfig LocationConfiguration) error
	RemoveLocation(locationName, locationType string) error
	SetLocationConfiguration(lConfig LocationConfiguration) error
	GetLocations() ([]LocationConfiguration, error)
	GetLocationProperties(locationName, locationType string) (config.DynamicMap, error)
	GetLocationPropertiesForNode(deploymentID, nodeName, locationType string) (config.DynamicMap, error)
	GetPropertiesForFirstLocationOfType(locationType string) (config.DynamicMap, error)
	Cleanup() error
}

type locationManager struct {
	cc        *api.Client
	hpAdapter adapter.LocationAdapter
}

// LocationsDefinition represents the structure of an initialization file defining locations
type LocationsDefinition struct {
	Locations []LocationConfiguration `yaml:"locations" mapstructure:"locations" json:"locations"`
}

// NewManager creates a Location Manager
func NewManager(cfg config.Configuration) (Manager, error) {

	var locationMgr *locationManager
	if locationMgr == nil {
		client, err := cfg.GetConsulClient()
		if err != nil {
			return locationMgr, err
		}
		locationMgr = &locationManager{cc: client}
		locationMgr.hpAdapter = adapter.NewHostsPoolLocationAdapter(client)
	}

	return locationMgr, nil
}

// CreateLocation creates a location. Its name must be unique.
// If a location already exists with this name, an error will be returned.
func (mgr *locationManager) CreateLocation(lConfig LocationConfiguration) error {

	return mgr.createLocation(lConfig)
}

// RemoveLocation removes a given location
func (mgr *locationManager) RemoveLocation(locationName, locationType string) error {

	return mgr.removeLocation(locationName, locationType)
}

// SetLocationConfiguration sets the configuration of a location.
// If this location doesn't exist, it will be created.
func (mgr *locationManager) SetLocationConfiguration(lConfig LocationConfiguration) error {

	return mgr.setLocationConfiguration(lConfig)
}

// Cleanup deletes all locations configured
func (mgr *locationManager) Cleanup() error {
	return mgr.cleanupLocations()
}

// GetLocations returns all locations configured
func (mgr *locationManager) GetLocations() ([]LocationConfiguration, error) {

	var locations []LocationConfiguration
	kvps, _, err := mgr.cc.KV().List(consulutil.LocationsPrefix, nil)
	if err != nil {
		return locations, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	for _, kvp := range kvps {
		locationName := path.Base(kvp.Key)
		var lConfig LocationConfiguration
		err = json.Unmarshal(kvp.Value, &lConfig)
		if err != nil {
			return locations, errors.Wrapf(err, "failed to unmarshal configuration for location %s", locationName)
		}
		locations = append(locations, lConfig)
	}

	return locations, err
}

// GetLocationProperties returns properties configured for a given location
func (mgr *locationManager) GetLocationProperties(locationName, locationType string) (config.DynamicMap, error) {

	if locationType == adapter.AdaptedLocationType {
		return mgr.hpAdapter.GetLocationConfiguration(locationName)
	}
	var props config.DynamicMap
	kvp, _, err := mgr.cc.KV().Get(path.Join(consulutil.LocationsPrefix, locationName), nil)
	if err != nil {
		return props, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return props, errors.Errorf("No such location %q", locationName)
	}

	var lConfig LocationConfiguration
	err = json.Unmarshal(kvp.Value, &lConfig)
	props = lConfig.Properties
	if err != nil {
		return props, errors.Wrapf(err, "failed to unmarshal configuration for location %s", locationName)
	}

	return props, err
}

// GetLocationPropertiesForNode returns the properties of the location
// on which the node template in argument is or will be created.
// The corresponding location name should be provided in the node template metadata.
// If no location name is provided in the node template metadata, the configuration
// of the first location of the expected type in returned
func (mgr *locationManager) GetLocationPropertiesForNode(deploymentID, nodeName, locationType string) (config.DynamicMap, error) {

	// Get the location name in node template metadata
	found, locationName, err := deployments.GetNodeMetadata(
		mgr.cc.KV(), deploymentID, nodeName, tosca.MetadataLocationNameKey)
	if err != nil {
		return nil, err
	}

	if found {
		return mgr.GetLocationProperties(locationName, locationType)
	}

	// No location specified, get the first location matching the expected type
	return mgr.GetPropertiesForFirstLocationOfType(locationType)

}

// GetPropertiesForFirstLocationOfType returns properties for the first location
// of a given infrastructure type.
// Returns an error if there is no location of such type
func (mgr *locationManager) GetPropertiesForFirstLocationOfType(locationType string) (config.DynamicMap, error) {

	var props config.DynamicMap
	locations, err := mgr.GetLocations()
	if err == nil {
		// Set the error in case no location of such type is found
		err = errors.Errorf("Found no location of type %q", locationType)
		for _, loc := range locations {
			if loc.Type == locationType {
				props = loc.Properties
				err = nil
				break
			}
		}
	}

	return props, err
}

// InitializeLocations initialize locations from a file. This initialization
// wil be performed only once. If called a second time, no operation will be perforned
// and this call will return false to notify the caller that locations have already been
// done
func (mgr *locationManager) InitializeLocations(locationFilePath string) (bool, error) {

	initDone := false

	lock, _, err := mgr.lockLocations()
	if err != nil {
		return initDone, err
	}

	defer lock.Unlock()

	kv := mgr.cc.KV()
	// Appending a final "/" here is not necessary as there is no other keys starting
	// with consulutil.LocationsPrefix prefix
	kvpList, _, err := kv.List(consulutil.LocationsPrefix, nil)
	if err != nil {
		return initDone, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	if len(kvpList) > 0 {
		// locations are already stored in consul, nothing more to do here
		return initDone, err
	}

	v := viper.New()
	v.SetConfigFile(locationFilePath)
	err = v.ReadInConfig()
	if err != nil {
		return initDone, err
	}

	var locationsDefined LocationsDefinition
	err = v.Unmarshal(&locationsDefined)
	if err != nil {
		return initDone, err
	}

	log.Debugf("Configuring locations in consul")
	initDone = true
	for _, lConfig := range locationsDefined.Locations {

		err := mgr.setLocationConfiguration(lConfig)
		if err != nil {
			return initDone, err
		}
	}

	return initDone, err
}

func (mgr *locationManager) lockLocations() (*api.Lock, <-chan struct{}, error) {

	lock, err := mgr.cc.LockOpts(&api.LockOptions{
		Key:          ".lock_" + consulutil.LocationsPrefix,
		LockTryOnce:  true,
		LockWaitTime: 30 * time.Second,
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	var lockCh <-chan struct{}
	for lockCh == nil {
		log.Debug("Try to acquire lock for locations in consul")
		lockCh, err = lock.Lock(nil)
		if err != nil {
			return nil, nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}

	}
	log.Debug("Lock for locations acquired")
	return lock, lockCh, nil
}

// Deletes locations stored in Consul
func (mgr *locationManager) cleanupLocations() error {

	lock, _, err := mgr.lockLocations()
	if err != nil {
		return err
	}

	defer lock.Unlock()

	kv := mgr.cc.KV()
	// Appending a final "/" here is not necessary as there is no other keys starting
	// with consulutil.LocationsPrefix prefix
	_, err = kv.DeleteTree(consulutil.LocationsPrefix, nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	return err
}

func (mgr *locationManager) createLocation(configuration LocationConfiguration) error {

	lock, _, err := mgr.lockLocations()
	if err != nil {
		return err
	}

	defer lock.Unlock()

	// Check if a location with this name already exists
	kvp, _, err := mgr.cc.KV().Get(path.Join(consulutil.LocationsPrefix, configuration.Name), nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil && len(kvp.Value) != 0 {
		return errors.Errorf("location %q already exists", configuration.Name)
	}

	return mgr.setLocationConfiguration(configuration)
}

func (mgr *locationManager) setLocationConfiguration(configuration LocationConfiguration) error {
	if configuration.Type == adapter.AdaptedLocationType {
		return mgr.hpAdapter.SetLocationConfiguration(configuration.Name, configuration.Properties)
	}
	b, err := json.Marshal(configuration)
	if err != nil {
		log.Printf("Failed to marshal infrastructure config [%+v]: due to error:%+v", configuration, err)
		return err
	}

	err = consulutil.StoreConsulKey(path.Join(consulutil.LocationsPrefix, configuration.Name), b)
	if err != nil {
		return errors.Wrapf(err, "failed to store location %s in consul", configuration.Name)
	}
	return err
}

func (mgr *locationManager) removeLocation(locationName, locationType string) error {
	if locationType == adapter.AdaptedLocationType {
		return mgr.hpAdapter.RemoveLocation(locationName)
	}

	_, err := mgr.cc.KV().Delete(path.Join(consulutil.LocationsPrefix, locationName), nil)
	if err != nil {
		return errors.Wrapf(err, "failed to remove location %s", locationName)
	}
	return err
}
