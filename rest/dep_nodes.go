package rest

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"path"
)

func (s *Server) getNodeHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	nodeName := params.ByName("nodeName")

	kv := s.consulClient.KV()

	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, id, "topology/nodes", nodeName, "status"), nil)
	if err != nil {
		log.Panic(err)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		WriteError(w, r, ErrNotFound)
		return
	}
	node := Node{Name: nodeName, Status: string(kvp.Value)}
	links := []AtomLink{newAtomLink(LINK_REL_SELF, r.URL.Path)}
	instanceIds, err := deployments.GetNodeInstancesIds(kv, id, nodeName)
	if err != nil {
		log.Panic(err)
	}
	for _, instanceId := range instanceIds {
		links = append(links, newAtomLink(LINK_REL_INSTANCE, path.Join(r.URL.Path, "instances", instanceId)))
	}
	node.Links = links
	encodeJsonResponse(w, r, node)
}

func (s *Server) getNodeInstanceHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	nodeName := params.ByName("nodeName")
	instanceId := params.ByName("instanceId")
	kv := s.consulClient.KV()
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, id, "topology/instances", nodeName, instanceId, "status"), nil)
	if err != nil {
		log.Panic(err)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		WriteError(w, r, ErrNotFound)
		return
	}
	nodePath := path.Clean(r.URL.Path + "/../..")
	nodeInstance := NodeInstance{Id: instanceId, Status: string(kvp.Value)}
	nodeInstance.Links = []AtomLink{
		newAtomLink(LINK_REL_SELF, r.URL.Path),
		newAtomLink(LINK_REL_NODE, nodePath),
	}

	encodeJsonResponse(w, r, nodeInstance)
}
