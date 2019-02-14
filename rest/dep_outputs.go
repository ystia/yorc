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
	"path"

	"github.com/julienschmidt/httprouter"

	"github.com/ystia/yorc/v3/deployments"
	"github.com/ystia/yorc/v3/log"
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

	result, err := deployments.GetTopologyOutputValue(kv, id, opt)
	if err != nil {
		if status == deployments.DEPLOYMENT_IN_PROGRESS {
			// Things may not be resolvable yet
			encodeJSONResponse(w, r, Output{Name: opt, Value: ""})
			return
		}
		log.Panicf("Unable to resolve topology output %q: %v", opt, err)
	}
	if result == nil {
		writeError(w, r, errNotFound)
		return
	}

	encodeJSONResponse(w, r, Output{Name: opt, Value: result.String()})
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
