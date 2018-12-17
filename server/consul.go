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

package server

import (
	"github.com/blang/semver"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/server/upgradeschema"
)

var upgradeToMap = map[string]func(*api.KV, <-chan struct{}) error{
	"1.0.0": upgradeschema.UpgradeFromPre31,
}

var orderedUpgradesVersions []semver.Version

func init() {
	for v := range upgradeToMap {
		orderedUpgradesVersions = append(orderedUpgradesVersions, semver.MustParse(v))
	}
	semver.Sort(orderedUpgradesVersions)
}

func synchronizeDBUpdate(client *api.Client) (*api.Lock, <-chan struct{}, error) {

	lock, err := client.LockOpts(&api.LockOptions{
		Key: consulutil.YorcSchemaVersionPath + ".lock",
		// Value:        []byte("check"),
		LockTryOnce: true,
		// LockWaitTime: 10 * time.Millisecond,
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	var leaderCh <-chan struct{}
	for leaderCh == nil {
		log.Debug("Try to acquire lock for DB schema update check")
		leaderCh, err = lock.Lock(nil)
		if err != nil {
			return nil, nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}

	}
	log.Debug("Lock for DB schema update check acquired")
	return lock, leaderCh, nil
}

func setupConsulDBSchema(client *api.Client) error {
	lock, leaderCh, err := synchronizeDBUpdate(client)
	if err != nil {
		return err
	}
	defer func() {
		lock.Unlock()
		// will fail if another instance take it but will cleanup if it is the last one
		lock.Destroy()
	}()
	kv := client.KV()
	kvp, _, err := kv.Get(consulutil.YorcSchemaVersionPath, nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	// If there is no version it could mean either that:
	// 1/ we are updating from a pre-3.1 release
	// 2/ we are the first server to boot for the first time
	if kvp == nil {
		kvps, _, err := kv.Keys(consulutil.DeploymentKVPrefix+"/", "/", nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if len(kvps) > 0 {
			return upgradeFromVersion(client, leaderCh, "0.0.0")
		}

		return setNewVersion(kv)
	}

	return upgradeFromVersion(client, leaderCh, string(kvp.Value))
}

func setNewVersion(kv *api.KV) error {
	err := consulutil.StoreConsulKeyAsString(consulutil.YorcSchemaVersionPath, consulutil.YorcSchemaVersion)
	if err != nil {
		return err
	}
	return nil
}

func upgradeFromVersion(client *api.Client, leaderCh <-chan struct{}, fromVersion string) error {
	vCurrent, err := semver.Make(fromVersion)
	if err != nil {
		return errors.Wrapf(err, "failed to parse current version of consul db schema")
	}
	vNew, err := semver.Make(consulutil.YorcSchemaVersion)
	if err != nil {
		return errors.Wrapf(err, "failed to parse current version of consul db schema")
	}

	switch vNew.Compare(vCurrent) {
	case 0:
		// Same version nothing to do
		return nil
	case 1:
		// Make a Consul snapshot and restore it if any error occurs
		snap := client.Snapshot()
		snapReader, _, err := snap.Save(nil)
		if err != nil {
			return errors.Wrapf(err, "failed to upgrade consul db schema to %q", err)
		}
		defer snapReader.Close()
		for _, vUp := range orderedUpgradesVersions {
			if vUp.GE(vCurrent) {
				err = upgradeToMap[vUp.String()](client.KV(), leaderCh)
				if err != nil {
					// Restore Consul snapshot
					restoreErr := snap.Restore(nil, snapReader)
					if restoreErr != nil {
						log.Printf("failed to restore consul db schema to %q due to error:%+v", fromVersion, restoreErr)
					} else {
						log.Printf("As any error occurred, schema has been successfully restored to version %q", fromVersion)
					}
					return errors.Wrapf(err, "failed to upgrade consul db schema to %q.", vUp)
				}
			}
		}
	case -1:
		return errors.Errorf("this version of Yorc is too old compared to the current DB schema (%s), an upgrade is needed.", vCurrent)

	}

	return setNewVersion(client.KV())
}
