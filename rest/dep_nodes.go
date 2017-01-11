package rest

import (
	"net/http"
	"path"

	yaml "gopkg.in/yaml.v2"

	"github.com/julienschmidt/httprouter"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

func (s *Server) getNodeHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	nodeName := params.ByName("nodeName")

	kv := s.consulClient.KV()
	node := Node{Name: nodeName}
	links := []AtomLink{newAtomLink(LINK_REL_SELF, r.URL.Path)}
	instanceIds, err := deployments.GetNodeInstancesIds(kv, id, nodeName)
	if err != nil {
		log.Panic(err)
	}
	for _, instanceID := range instanceIds {
		links = append(links, newAtomLink(LINK_REL_INSTANCE, path.Join(r.URL.Path, "instances", instanceID)))
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
	instanceID := params.ByName("instanceId")
	kv := s.consulClient.KV()
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, id, "topology/instances", nodeName, instanceID, "attributes/state"), nil)
	if err != nil {
		log.Panic(err)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		WriteError(w, r, ErrNotFound)
		return
	}
	nodePath := path.Clean(r.URL.Path + "/../..")
	nodeInstance := NodeInstance{Id: instanceID, Status: string(kvp.Value)}
	nodeInstance.Links = []AtomLink{
		newAtomLink(LINK_REL_SELF, r.URL.Path),
		newAtomLink(LINK_REL_NODE, nodePath),
	}
	attributesNames, err := deployments.GetNodeAttributesNames(kv, id, nodeName)
	if err != nil {
		WriteError(w, r, NewInternalServerError(err))
		return
	}
	for _, attr := range attributesNames {
		nodeInstance.Links = append(nodeInstance.Links, newAtomLink(LINK_REL_ATTRIBUTE, path.Join(r.URL.Path, "attributes", attr)))
	}
	encodeJsonResponse(w, r, nodeInstance)
}

func (s *Server) getNodeInstanceAttributesListHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	nodeName := params.ByName("nodeName")
	instanceId := params.ByName("instanceId")
	kv := s.consulClient.KV()
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, id, "topology/instances", nodeName, instanceId, "attributes/state"), nil)
	if err != nil {
		log.Panic(err)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		WriteError(w, r, ErrNotFound)
		return
	}

	attributesNames, err := deployments.GetNodeAttributesNames(kv, id, nodeName)
	if err != nil {
		WriteError(w, r, NewInternalServerError(err))
		return
	}
	attrList := AttributesCollection{Attributes: make([]AtomLink, len(attributesNames))}
	for i, attr := range attributesNames {
		attrList.Attributes[i] = newAtomLink(LINK_REL_ATTRIBUTE, path.Join(r.URL.Path, attr))
	}
	encodeJsonResponse(w, r, attrList)
}

func (s *Server) getNodeInstanceAttributeHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	nodeName := params.ByName("nodeName")
	instanceId := params.ByName("instanceId")
	attributeName := params.ByName("attributeName")
	kv := s.consulClient.KV()
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, id, "topology/instances", nodeName, instanceId, "attributes/state"), nil)
	if err != nil {
		log.Panic(err)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		WriteError(w, r, ErrNotFound)
		return
	}
	found, result, err := deployments.GetNodeAttributes(kv, id, nodeName, attributeName)
	if err != nil {
		WriteError(w, r, NewInternalServerError(err))
		return
	}

	if !found {
		WriteError(w, r, ErrNotFound)
		return
	}
	instanceAttribute, ok := result[instanceId]
	if !ok {
		WriteError(w, r, ErrNotFound)
		return
	}
	if instanceAttribute != "" {
		resultExpr := &tosca.ValueAssignment{}
		err = yaml.Unmarshal([]byte(instanceAttribute), resultExpr)
		if err != nil {
			log.Panic(err)
		}
		resolver := deployments.NewResolver(kv, id)
		instanceAttribute, err = resolver.ResolveExpressionForNode(resultExpr.Expression, nodeName, instanceId)
		if err != nil {
			log.Panic(err)
		}
	}
	attribute := Attribute{Name: attributeName, Value: instanceAttribute}
	encodeJsonResponse(w, r, attribute)
}
