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

package upgradeschema

import (
	"path"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
)

// UpgradeTo112 allows to upgrade Consul schema from 1.1.1 to 1.1.2
func UpgradeTo112(kv *api.KV, leaderch <-chan struct{}) error {
	log.Print("Upgrading to database version 1.1.12..")
	return up112UpgradeHostsPoolStorage(kv)
}

func up112UpgradeHostsPoolStorage(kv *api.KV) error {
	log.Print("\tUpgrade hosts pool storage...")
	defaultLocationName := "hostsPool111"
	// Check the schema is the previous one by retrieving the host status
	keys, _, err := kv.Keys(consulutil.HostsPoolPrefix+"/", "/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, k := range keys {
		log.Debugf("check key=%q", k)
		kvp, _, err := kv.Get(path.Join(k, "status"), nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp == nil || len(kvp.Value) == 0 {
			log.Debugf("No host status retrieved for this key. We assume its a hosts pool location with up-to-date schema.")
			continue
		}

		log.Debugf("Found a host from legacy pool: it will be associated to location name:%q. This name can be changed with the CLI/REST API locations.", defaultLocationName)

		err = moveKeyToNewSchema(kv, k, defaultLocationName)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
	}
	return nil
}

func moveKeyToNewSchema(kv *api.KV, key, defaultLocationName string) error {
	kvps, _, err := kv.List(key+"/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, kvp := range kvps {
		if kvp != nil && len(kvp.Value) > 0 {
			newKey := path.Join(consulutil.HostsPoolPrefix, defaultLocationName, strings.TrimPrefix(kvp.Key, consulutil.HostsPoolPrefix))
			log.Debugf("Create new key: %q", newKey)
			err = consulutil.StoreConsulKey(newKey, kvp.Value)
			if err != nil {
				return errors.Wrapf(err, "failed to store hosts pool key with location:%q", defaultLocationName)
			}
		}
	}

	log.Debugf("Delete old key tree: %q", key)
	_, err = kv.DeleteTree(key, nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return nil
}
