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

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/log"
)

// UpgradeTo130 allows to upgrade Consul schema from 1.2.0 to 1.3.0
func UpgradeTo130(cfg config.Configuration, kv *api.KV, leaderch <-chan struct{}) error {
	log.Print("Upgrading to database version 1.3.0")
	return upgradeDeploymentsRefactoring(cfg, "1.3.0")
}
