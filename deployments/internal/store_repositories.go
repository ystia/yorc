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

package internal

import (
	"context"
	"path"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/tosca"
)

// StoreRepositories store repositories
func StoreRepositories(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix string) error {
	repositoriesPrefix := path.Join(topologyPrefix, "repositories")
	for repositoryName, repo := range topology.Repositories {
		repoPrefix := path.Join(repositoriesPrefix, repositoryName)
		consulStore.StoreConsulKeyAsString(path.Join(repoPrefix, "url"), repo.URL)
		consulStore.StoreConsulKeyAsString(path.Join(repoPrefix, "type"), repo.Type)
		consulStore.StoreConsulKeyAsString(path.Join(repoPrefix, "credentials", "user"), repo.Credit.User)
		consulStore.StoreConsulKeyAsString(path.Join(repoPrefix, "credentials", "token"), repo.Credit.Token)
		if repo.Credit.TokenType == "" {
			repo.Credit.TokenType = "password"
		}
		consulStore.StoreConsulKeyAsString(path.Join(repoPrefix, "credentials", "token_type"), repo.Credit.TokenType)
	}

	return nil
}
