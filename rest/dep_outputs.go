package rest

import (
	"net/http"
	"path"
	"strings"

	"github.com/julienschmidt/httprouter"
	"gopkg.in/yaml.v2"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

func (s *Server) getOutputHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	opt := params.ByName("opt")

	kv := s.consulClient.KV()

	status, err := deployments.GetDeploymentStatus(kv, id)
	if err != nil {
		if deployments.IsDeploymentNotFoundError(err) {
			WriteError(w, r, ErrNotFound)
		}
	}

	expression, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, id, "topology/outputs", opt, "value"), nil)
	if err != nil {
		log.Panic(err)
	}
	if expression == nil {
		WriteError(w, r, ErrNotFound)
		return
	}

	var output Output
	if len(expression.Value) > 0 {
		va := tosca.ValueAssignment{}
		err = yaml.Unmarshal(expression.Value, &va)
		if err != nil {
			log.Panicf("Unable to unmarshal value expression: %v", err)
		}
		result, err := deployments.NewResolver(kv, id).ResolveExpressionForNode(va.Expression, "", "")
		if err != nil {

			if status == deployments.DEPLOYMENT_IN_PROGRESS {
				// Things may not be resolvable yet
				output = Output{Name: opt, Value: ""}
			} else {
				log.Panicf("Unable to resolve value expression %q: %v", string(expression.Value), err)
			}
		} else {
			output = Output{Name: opt, Value: result}
		}
	} else {
		output = Output{Name: opt, Value: ""}
	}
	encodeJSONResponse(w, r, output)
}

func (s *Server) listOutputsHandler(w http.ResponseWriter, r *http.Request) {

	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
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
	outputsTopoPrefix := path.Join(consulutil.DeploymentKVPrefix, id, "/topology/outputs") + "/"
	optPaths, _, err := kv.Keys(outputsTopoPrefix, "/", nil)
	if err != nil {
		log.Panic(err)
	}
	links := make([]AtomLink, len(optPaths))
	for optIndex, optP := range optPaths {
		optName := strings.TrimRight(strings.TrimPrefix(optP, outputsTopoPrefix), "/ ")

		link := newAtomLink(LinkRelOutput, path.Join("/deployments", id, "outputs", optName))
		links[optIndex] = link
	}
	return links
}
