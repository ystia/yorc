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
	"path"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tosca"
)

var locationMgr *locationManager

type locationManager struct {
	cc *api.Client
}

// Initialize stores configured locations in consul, if not already done
func Initialize(cfg config.Configuration, cc *api.Client) error {
	locationMgr = &locationManager{cc: cc}
	return locationMgr.storeLocations(cfg)
}

// CreateLocation creates a location. Its name must be unique.
// If a location already exists with this name, an error will be returned.
func CreateLocation(lConfig config.LocationConfiguration) error {

	return locationMgr.createLocation(lConfig)
}

// RemoveLocation removes a given location
func RemoveLocation(locationName string) error {

	return locationMgr.removeLocation(locationName)
}

// SetLocationConfiguration sets the configuration of a location.
// If this location doesn't exist, it will be created.
func SetLocationConfiguration(lConfig config.LocationConfiguration) error {

	return locationMgr.setLocationConfiguration(lConfig)
}

// Cleanup deletes all locations configured
func Cleanup() error {
	return locationMgr.cleanupLocations()
}

// GetLocations returns all locations configured
func GetLocations() ([]config.LocationConfiguration, error) {

	var locations []config.LocationConfiguration
	kvps, _, err := locationMgr.cc.KV().List(consulutil.LocationsPrefix, nil)
	if err != nil {
		return locations, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	for _, kvp := range kvps {
		locationName := path.Base(kvp.Key)
		var lConfig config.LocationConfiguration
		err = json.Unmarshal(kvp.Value, &lConfig)
		if err != nil {
			return locations, errors.Wrapf(err, "failed to unmarshal configuration for location %s", locationName)
		}
		locations = append(locations, lConfig)
	}

	return locations, err
}

// GetLocationProperties returns properties configured for a given location
func GetLocationProperties(locationName string) (config.DynamicMap, error) {

	var props config.DynamicMap
	kvp, _, err := locationMgr.cc.KV().Get(path.Join(consulutil.LocationsPrefix, locationName), nil)
	if err != nil {
		return props, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return props, errors.Errorf("No such location %q", locationName)
	}

	var lConfig config.LocationConfiguration
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
func GetLocationPropertiesForNode(deploymentID, nodeName, locationType string) (config.DynamicMap, error) {

	// Get the location name in node template metadata
	found, locationName, err := deployments.GetNodeMetadata(
		locationMgr.cc.KV(), deploymentID, nodeName, tosca.MetadataLocationNameKey)
	if err != nil {
		return nil, err
	}

	if found {
		return GetLocationProperties(locationName)
	}

	// No location specified, get the first location matching the expected type
	return GetPropertiesForFirstLocationOfType(locationType)

}

// GetPropertiesForFirstLocationOfType returns properties for the first location
// of a given infrastructure type.
// Returns an error if there is no location of such type
func GetPropertiesForFirstLocationOfType(locationType string) (config.DynamicMap, error) {

	var props config.DynamicMap
	locations, err := GetLocations()
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

// Store locations in Consul if not already done
func (mgr *locationManager) storeLocations(cfg config.Configuration) error {

	lock, _, err := mgr.lockLocations()
	if err != nil {
		return err
	}

	defer lock.Unlock()

	kv := mgr.cc.KV()
	// Appending a final "/" here is not necessary as there is no other keys starting
	// with consulutil.LocationsPrefix prefix
	kvpList, _, err := kv.List(consulutil.LocationsPrefix, nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	if len(kvpList) > 0 {
		// locations are already stored in consul, nothing more to do here
		return nil
	}

	log.Debugf("Configuring locations in consul")

	for _, lConfig := range cfg.Locations {

		err := mgr.setLocationConfiguration(lConfig)
		if err != nil {
			return err
		}
	}

	return err
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

func (mgr *locationManager) createLocation(configuration config.LocationConfiguration) error {

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

func (mgr *locationManager) setLocationConfiguration(configuration config.LocationConfiguration) error {

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

func (mgr *locationManager) removeLocation(locationName string) error {
	_, err := mgr.cc.KV().Delete(path.Join(consulutil.LocationsPrefix, locationName), nil)
	if err != nil {
		return errors.Wrapf(err, "failed to remove location %s", locationName)
	}
	return err
}
