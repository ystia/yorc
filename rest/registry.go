package rest

import (
	"net/http"

	"novaforge.bull.com/starlings-janus/janus/registry"
)

var reg = registry.GetRegistry()

func (s *Server) listRegistryDelegatesHandler(w http.ResponseWriter, r *http.Request) {
	delegates := reg.ListDelegateExecutors()
	delegatesCollection := RegistryDelegatesCollection{Delegates: delegates}
	encodeJSONResponse(w, r, delegatesCollection)
}

func (s *Server) listRegistryDefinitionsHandler(w http.ResponseWriter, r *http.Request) {
	definitions := reg.ListToscaDefinitions()
	definitionsCollection := RegistryDefinitionsCollection{Definitions: definitions}
	encodeJSONResponse(w, r, definitionsCollection)
}
