package rest

import (
	"fmt"
	"net/http"
	"path"

	"github.com/julienschmidt/httprouter"

	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/helper/collections"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
)

func (s *Server) getNodeHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
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
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
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
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
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
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")
	nodeName := params.ByName("nodeName")
	instanceID := params.ByName("instanceId")
	attributeName := params.ByName("attributeName")
	kv := s.consulClient.KV()
	// state should exists if instance exists
	instances, err := deployments.GetNodeInstancesIds(kv, id, nodeName)
	if err != nil {
		writeError(w, r, newInternalServerError(err))
		return
	}
	if !collections.ContainsString(instances, instanceID) {
		writeError(w, r, newContentNotFoundError(fmt.Sprintf("Instance %q for node %q", instanceID, nodeName)))
		return
	}
	found, instanceAttribute, err := deployments.GetInstanceAttribute(kv, id, nodeName, instanceID, attributeName)
	if err != nil {
		writeError(w, r, newInternalServerError(err))
		return
	}
	if !found {
		writeError(w, r, newContentNotFoundError(fmt.Sprintf("Attribute %q for node %q (instance %q)", attributeName, nodeName, instanceID)))
		return
	}

	attribute := Attribute{Name: attributeName, Value: instanceAttribute}
	encodeJSONResponse(w, r, attribute)
}
