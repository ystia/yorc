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
	links := []AtomLink{newAtomLink(LinkRelSelf, r.URL.Path), newAtomLink(LinkRelDeployment, path.Clean(r.URL.Path+"/../.."))}
	instanceIds, err := deployments.GetNodeInstancesIds(kv, id, nodeName)
	if err != nil {
		log.Panic(err)
	}
	for _, instanceID := range instanceIds {
		links = append(links, newAtomLink(LinkRelInstance, path.Join(r.URL.Path, "instances", instanceID)))
	}
	node.Links = links
	encodeJSONResponse(w, r, node)
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
		writeError(w, r, errNotFound)
		return
	}
	nodePath := path.Clean(r.URL.Path + "/../..")
	nodeInstance := NodeInstance{ID: instanceID, Status: string(kvp.Value)}
	nodeInstance.Links = []AtomLink{
		newAtomLink(LinkRelSelf, r.URL.Path),
		newAtomLink(LinkRelNode, nodePath),
		newAtomLink(LinkRelDeployment, path.Clean(r.URL.Path+"/../../../..")),
	}
	attributesNames, err := deployments.GetNodeAttributesNames(kv, id, nodeName)
	if err != nil {
		writeError(w, r, newInternalServerError(err))
		return
	}
	for _, attr := range attributesNames {
		nodeInstance.Links = append(nodeInstance.Links, newAtomLink(LinkRelAttribute, path.Join(r.URL.Path, "attributes", attr)))
	}
	encodeJSONResponse(w, r, nodeInstance)
}

func (s *Server) getNodeInstanceAttributesListHandler(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, r, errNotFound)
		return
	}

	attributesNames, err := deployments.GetNodeAttributesNames(kv, id, nodeName)
	if err != nil {
		writeError(w, r, newInternalServerError(err))
		return
	}
	attrList := AttributesCollection{Attributes: make([]AtomLink, len(attributesNames))}
	for i, attr := range attributesNames {
		attrList.Attributes[i] = newAtomLink(LinkRelAttribute, path.Join(r.URL.Path, attr))
	}
	encodeJSONResponse(w, r, attrList)
}

func (s *Server) getNodeInstanceAttributeHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	nodeName := params.ByName("nodeName")
	instanceID := params.ByName("instanceId")
	attributeName := params.ByName("attributeName")
	kv := s.consulClient.KV()
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, id, "topology/instances", nodeName, instanceID, "attributes/state"), nil)
	if err != nil {
		log.Panic(err)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		writeError(w, r, errNotFound)
		return
	}
	found, result, err := deployments.GetNodeAttributes(kv, id, nodeName, attributeName)
	if err != nil {
		writeError(w, r, newInternalServerError(err))
		return
	}

	if !found {
		writeError(w, r, errNotFound)
		return
	}
	instanceAttribute, ok := result[instanceID]
	if !ok {
		writeError(w, r, errNotFound)
		return
	}
	if instanceAttribute != "" {
		resultExpr := &tosca.ValueAssignment{}
		err = yaml.Unmarshal([]byte(instanceAttribute), resultExpr)
		if err != nil {
			log.Panic(err)
		}
		resolver := deployments.NewResolver(kv, id)
		instanceAttribute, err = resolver.ResolveValueAssignmentForNode(resultExpr, nodeName, instanceID)
		if err != nil {
			log.Panic(err)
		}
	}
	attribute := Attribute{Name: attributeName, Value: instanceAttribute}
	encodeJSONResponse(w, r, attribute)
}
