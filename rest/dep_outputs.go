package rest

import (
	"net/http"
	"path"

	"github.com/julienschmidt/httprouter"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func (s *Server) getOutputHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")
	opt := params.ByName("opt")

	kv := s.consulClient.KV()

	status, err := deployments.GetDeploymentStatus(kv, id)
	if err != nil {
		if deployments.IsDeploymentNotFoundError(err) {
			writeError(w, r, errNotFound)
		}
	}

	found, result, err := deployments.GetTopologyOutput(kv, id, opt)
	if err != nil {
		if status == deployments.DEPLOYMENT_IN_PROGRESS {
			// Things may not be resolvable yet
			encodeJSONResponse(w, r, Output{Name: opt, Value: ""})
			return
		}
		log.Panicf("Unable to resolve topology output %q: %v", opt, err)
	}
	if !found {
		writeError(w, r, errNotFound)
		return
	}

	encodeJSONResponse(w, r, Output{Name: opt, Value: result})
}

func (s *Server) listOutputsHandler(w http.ResponseWriter, r *http.Request) {

	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")
	links := s.listOutputsLinks(id)
	if len(links) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	optCol := OutputsCollection{Outputs: links}
	encodeJSONResponse(w, r, optCol)
}

func (s *Server) listOutputsLinks(id string) []AtomLink {
	kv := s.consulClient.KV()
	outNames, err := deployments.GetTopologyOutputsNames(kv, id)
	if err != nil {
		log.Panic(err)
	}
	links := make([]AtomLink, len(outNames))
	for optIndex, optName := range outNames {
		link := newAtomLink(LinkRelOutput, path.Join("/deployments", id, "outputs", optName))
		links[optIndex] = link
	}
	return links
}
