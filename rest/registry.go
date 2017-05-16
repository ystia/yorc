package rest

import (
	"net/http"

	"novaforge.bull.com/starlings-janus/janus/prov/registry"
)

var reg = registry.GetRegistry()

func (s *Server) listRegistryDelegatesHandler(w http.ResponseWriter, r *http.Request) {
	delegates := reg.ListDelegateExecutors()
	delegatesCollection := DelegatesCollection{Delegates: make([]Delegate, 0)}
	for match, origin := range delegates {
		delegatesCollection.Delegates = append(delegatesCollection.Delegates, Delegate{NodeType: match, Origin: origin})
	}
	encodeJSONResponse(w, r, delegatesCollection)
}
