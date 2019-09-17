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
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
)

var locationMgr *locationManager

type locationManager struct {
	cc *api.Client
}

// Initialize stores configured locations in consul, if not already done
func Initialize(cfg config.Configuration, cc *api.Client) error {
	locationMgr = &locationManager{cc: cc}

	err := locationMgr.storeLocations(cfg)

	return err
}

// GetLocations returns all locations configured
func GetLocations() (map[string]config.Location, error) {

	locations := make(map[string]config.Location)
	kvps, _, err := locationMgr.cc.KV().List(consulutil.LocationsPrefix, nil)
	if err != nil {
		return locations, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	for _, kvp := range kvps {
		locationName := path.Base(kvp.Key)
		var location config.Location
		err = json.Unmarshal(kvp.Value, &location)
		if err != nil {
			return locations, errors.Wrapf(err, "failed to unmarshal configuration for location %s", locationName)
		}

		locations[locationName] = location
	}

	return locations, err
}

// GetLocation returns configuration details for a given location
func GetLocation(locationName string) (config.Location, error) {

	var location config.Location
	kvp, _, err := locationMgr.cc.KV().Get(path.Join(consulutil.LocationsPrefix, locationName), nil)
	if err != nil {
		return location, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return location, errors.Errorf("No such location %q", locationName)
	}

	err = json.Unmarshal(kvp.Value, &location)
	if err != nil {
		return location, errors.Wrapf(err, "failed to unmarshal configuration for location %s", locationName)
	}

	return location, err
}

// GetFirstLocationOfType returns the first location of a given infrastructure type
// Used for backward compability while specifying infrastructures instead of locations
// is still supported
func GetFirstLocationOfType(locationType string) (config.Location, error) {

	var location config.Location
	locations, err := GetLocations()
	if err == nil {
		for _, loc := range locations {
			if loc.Type == locationType {
				location = loc
			}
			break
		}
	}

	return location, err
}

// Store locations in Consul if not already done
func (mgr *locationManager) storeLocations(cfg config.Configuration) error {

	lock, lockCh, err := mgr.lockLocations()
	if err != nil {
		return err
	}
	if lockCh == nil {
		log.Debugf("Another instance got the lock for locations")
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

	locations := cfg.Locations
	if len(cfg.Locations) == 0 && len(cfg.Infrastructures) > 0 {
		log.Println("Yorc configuration field 'Infrastructures' has been deprecated, use 'Locations' instead")
		// Converting each infrastructure as a location, using the infrastructure
		// type as a location name
		locations = make(map[string]config.Location)
		for k, v := range cfg.Infrastructures {
			locations[k] = config.Location{
				Type:          k,
				Configuration: v,
			}
		}
	}

	for locationName, infraConfig := range locations {

		b, err := json.Marshal(infraConfig)
		if err != nil {
			log.Printf("Failed to marshal infrastructure config [%+v]: due to error:%+v", infraConfig, err)
			return err
		}

		err = consulutil.StoreConsulKey(path.Join(consulutil.LocationsPrefix, locationName), b)
		if err != nil {
			return errors.Wrapf(err, "failed to store location %s in consul", locationName)
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
