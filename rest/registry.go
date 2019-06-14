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

package rest

import (
	"net/http"

	"github.com/ystia/yorc/v4/deployments/store"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/registry"
)

var reg = registry.GetRegistry()

func (s *Server) listRegistryDelegatesHandler(w http.ResponseWriter, r *http.Request) {
	delegates := reg.ListDelegateExecutors()
	delegatesCollection := RegistryDelegatesCollection{Delegates: delegates}
	encodeJSONResponse(w, r, delegatesCollection)
}

func (s *Server) listRegistryImplementationsHandler(w http.ResponseWriter, r *http.Request) {
	implementations := reg.ListOperationExecutors()
	implementationsCollection := RegistryImplementationsCollection{Implementations: implementations}
	encodeJSONResponse(w, r, implementationsCollection)
}

func (s *Server) listRegistryDefinitionsHandler(w http.ResponseWriter, r *http.Request) {
	defs, err := store.GetCommonsDefinitionsList()
	if err != nil {
		log.Panic(err)
	}

	definitionsCollection := RegistryDefinitionsCollection{Definitions: defs}
	encodeJSONResponse(w, r, definitionsCollection)
}

func (s *Server) listVaultsBuilderHandler(w http.ResponseWriter, r *http.Request) {
	vaults := reg.ListVaultClientBuilders()
	vaultsCollection := RegistryVaultsCollection{VaultClientBuilders: vaults}
	encodeJSONResponse(w, r, vaultsCollection)
}

func (s *Server) listInfraHandler(w http.ResponseWriter, r *http.Request) {
	infras := reg.ListInfraUsageCollectors()
	infraCollection := RegistryInfraUsageCollectorsCollection{InfraUsageCollectors: infras}
	encodeJSONResponse(w, r, infraCollection)
}
