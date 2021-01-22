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
	"context"
	"path"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
)

// UpgradeTo131 allows to upgrade Consul schema from 1.3.0 to 1.3.1
func UpgradeTo131(cfg config.Configuration, kv *api.KV, leaderch <-chan struct{}) error {
	log.Print("Upgrading to database version 1.3.1")

	tasks, _, err := kv.Keys(consulutil.TasksPrefix+"/", "/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()
	_, errGrp, store := consulutil.WithContext(ctx)
	for _, taskPath := range tasks {
		taskID := path.Base(taskPath)
		kvp, _, err := kv.Get(path.Join(taskPath, "targetId"), nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp != nil && len(kvp.Value) != 0 {
			deploymentID := string(kvp.Value)
			kvp, _, err = kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "status"), nil)
			if err != nil {
				return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
			}
			if kvp != nil && len(kvp.Value) != 0 {
				store.StoreConsulKeyWithFlags(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "tasks", taskID), nil, 1)
			}
		}
	}
	return errors.Wrap(errGrp.Wait(), "failed to store task key under deployment")
}
