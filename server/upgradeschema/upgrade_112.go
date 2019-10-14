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
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"path"
	"strings"
)

// UpgradeTo112 allows to upgrade Consul schema from 1.1.1 to 1.1.2
func UpgradeTo112(kv *api.KV, leaderch <-chan struct{}) error {
	log.Print("Upgrading to database version 1.1.12..")
	return up112UpgradeHostsPoolStorage(kv)
}

func up112UpgradeHostsPoolStorage(kv *api.KV) error {
	log.Print("\tUpgrade hosts pool storage...")

	// Check the schema is the previous one by retrieving the host status
	keys, _, err := kv.Keys(consulutil.HostsPoolPrefix+"/", "/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, k := range keys {
		kvp, _, err := kv.Get(path.Join(k, "status"), nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp == nil || len(kvp.Value) == 0 {
			log.Debugf("No host status retrieved. We assume the schema is up to date")
			return nil
		}
	}

	locationDefaultName := "hostsPool111"
	kvps, _, err := kv.List(consulutil.HostsPoolPrefix+"/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	if len(kvps) > 0 {
		log.Debugf("Found a hosts pool: it will be associated to location name:%q. This name can be changed with the CLI/REST API locations.", locationDefaultName)
	}
	for _, kvp := range kvps {
		if kvp != nil && len(kvp.Value) > 0 {
			newKey := path.Join(consulutil.HostsPoolPrefix, locationDefaultName, strings.TrimPrefix(kvp.Key, consulutil.HostsPoolPrefix))
			log.Debugf("Create new key: %q", newKey)
			err = consulutil.StoreConsulKey(newKey, kvp.Value)
			if err != nil {
				return errors.Wrapf(err, "failed to store hosts pool key with location:%q", locationDefaultName)
			}
		}
		log.Debugf("Delete old key: %q", kvp.Key)
		_, err = kv.Delete(kvp.Key, nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
	}
	return nil
}
