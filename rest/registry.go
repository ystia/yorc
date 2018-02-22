package rest

import (
	"github.com/ystia/yorc/registry"
	"net/http"
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
	definitions := reg.ListToscaDefinitions()
	definitionsCollection := RegistryDefinitionsCollection{Definitions: definitions}
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
