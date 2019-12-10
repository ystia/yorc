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
	"github.com/ystia/yorc/v4/config"
	"path"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/deployments/store"
	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
)

func getCommonsTypesList(kv *api.KV) ([]string, error) {
	paths := store.GetCommonsTypesPaths()
	res := make([]string, 0)
	for _, p := range paths {
		keys, _, err := kv.Keys(path.Join(p, "types")+"/", "/", nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		for _, k := range keys {
			res = append(res, path.Base(k))
		}
	}
	return res, nil

}

func up111RemoveCommonsImports(kv *api.KV, deploymentPrefix string) error {
	iKeys, _, err := kv.Keys(path.Join(deploymentPrefix, "topology/imports")+"/", "/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, importPrefix := range iKeys {
		importName := path.Base(importPrefix)
		if strings.HasPrefix(importName, "<") && strings.HasSuffix(importName, ">") {
			_, err = kv.DeleteTree(importPrefix+"/", nil)
			if err != nil {
				return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
			}
		}
	}
	return nil
}

func up111RemoveCommonsTypes(kv *api.KV, commons []string, deploymentPrefix string) error {
	tKeys, _, err := kv.Keys(path.Join(deploymentPrefix, "topology/types")+"/", "/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, tPrefix := range tKeys {
		if collections.ContainsString(commons, path.Base(tPrefix)) {
			log.Debugf("\tRemoving type %q from deployment %q", path.Base(tPrefix), path.Base(deploymentPrefix))
			_, err := kv.DeleteTree(tPrefix+"/", nil)
			if err != nil {
				return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
			}
		} else {
			consulutil.StoreConsulKeyWithFlags(path.Join(tPrefix, ".existFlag"), nil, 0)
		}
	}
	return nil
}
func up111UpgradeCommonsTypes(kv *api.KV) error {
	log.Print("\tRemoving commons types...")
	commons, err := getCommonsTypesList(kv)
	if err != nil {
		return err
	}
	depKeys, _, err := kv.Keys(consulutil.DeploymentKVPrefix+"/", "/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, deploymentPrefix := range depKeys {
		err = up111RemoveCommonsTypes(kv, commons, deploymentPrefix)
		if err != nil {
			return err
		}
		err = up111RemoveCommonsImports(kv, deploymentPrefix)
		if err != nil {
			return err
		}

	}
	return nil
}

// UpgradeTo111 allows to upgrade Consul schema from 1.1.0 to 1.1.1
func UpgradeTo111(cfg config.Configuration, kv *api.KV, leaderch <-chan struct{}) error {
	log.Print("Upgrading to database version 1.1.1...")
	return up111UpgradeCommonsTypes(kv)
}
