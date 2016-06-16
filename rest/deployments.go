package rest

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"github.com/gorilla/context"
	"github.com/hashicorp/consul/api"
	"github.com/julienschmidt/httprouter"
	"github.com/satori/go.uuid"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"novaforge.bull.com/starlings-janus/janus/tosca"
	"os"
	"path"
	"path/filepath"
	"strings"
	"novaforge.bull.com/starlings-janus/janus/deployments"
)

func extractFile(f *zip.File, path string) {
	fileReader, err := f.Open()
	if err != nil {
		log.Panic(err)
	}
	defer fileReader.Close()

	targetFile, err := os.OpenFile(path, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, f.Mode())
	if err != nil {
		log.Panic(err)
	}
	defer targetFile.Close()

	if _, err := io.Copy(targetFile, fileReader); err != nil {
		log.Panic(err)
	}
}

func (s *Server) newDeploymentHandler(w http.ResponseWriter, r *http.Request) {

	uuid := fmt.Sprint(uuid.NewV4())
	log.Printf("Analyzing deployment %s\n", uuid)

	var err error
	var file *os.File
	uploadPath := path.Join("work", "deployments", uuid)
	if err = os.MkdirAll(uploadPath, 0775); err != nil {
		log.Panicf("%+v", err)
	}

	file, err = os.Create(fmt.Sprintf("%s/deployment.zip", uploadPath))
	// check err
	if err != nil {
		log.Panicf("%+v", err)
	}

	_, err = io.Copy(file, r.Body)
	if err != nil {
		log.Panicf("%+v", err)
	}
	destDir := path.Join(uploadPath, "overlay")
	if err = os.MkdirAll(destDir, 0775); err != nil {
		log.Panicf("%+v", err)
	}
	zipReader, err := zip.OpenReader(file.Name())
	if err != nil {
		log.Fatal(err)
	}
	defer zipReader.Close()

	// Iterate through the files in the archive,
	// and extract them.
	// TODO: USe go routines to process files concurrently
	for _, f := range zipReader.File {
		path := path.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			if err = os.MkdirAll(path, f.Mode()); err != nil {
				log.Panicf("%+v", err)
			}
			continue
		}
		extractFile(f, path)
	}

	patterns := []struct {
		pattern string
	}{
		{"*.yml"},
		{"*.yaml"},
	}
	var yamlList []string
	for _, pattern := range patterns {
		if yamls, err := filepath.Glob(filepath.Join(destDir, pattern.pattern)); err != nil {
			log.Panicf("%+v", err)
		} else {
			for _, yaml := range yamls {
				yamlList = append(yamlList, yaml)
			}
		}
	}
	if len(yamlList) != 1 {
		log.Panic("One and only one YAML (.yml or .yaml) file should be present at the root of deployment archive")
	}

	// TODO to be improved parsing should be done in separate go routines
	topology := tosca.Topology{}
	//topology := make(map[interface{}]interface{})
	definition, err := os.Open(yamlList[0])
	defBytes, err := ioutil.ReadAll(definition)

	err = yaml.Unmarshal(defBytes, &topology)

	log.Printf("%+v", topology)

	storeConsulKey(s.consulClient.KV(), deployments.DeploymentKVPrefix + uuid + "/status", fmt.Sprint(deployments.INITIAL))

	s.storeDeploymentDefinition(topology, uuid)

	if err := s.tasksCollector.RegisterTask(uuid, tasks.DEPLOY); err != nil {
		log.Panic(err)
	}

	w.Header().Set("Location", fmt.Sprintf("/deployments/%s", uuid))
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) deleteDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	params = context.Get(r, "params").(httprouter.Params)
	id := params.ByName("id")
	if err := s.tasksCollector.RegisterTask(id, tasks.UNDEPLOY); err != nil {
		log.Panic(err)
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) getDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	params = context.Get(r, "params").(httprouter.Params)
	id := params.ByName("id")

	kv := s.consulClient.KV()
	result, _, err := kv.Get(deployments.DeploymentKVPrefix + "/" + id + "/status", nil)
	if err != nil {
		log.Panic(err)
	}
	if result == nil {
		WriteError(w, ErrNotFound)
		return
	}

	deployment := Deployment{Id: id, Status: string(result.Value)}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deployment)
}

func (s *Server) listDeploymentsHandler(w http.ResponseWriter, r *http.Request) {
	kv := s.consulClient.KV()
	depPaths, _, err := kv.Keys(deployments.DeploymentKVPrefix + "/", "/", nil)
	if err != nil {
		log.Panic(err)
	}
	if len(depPaths) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	depCol := DeploymentsCollection{Deployments: make([]AtomLink, len(depPaths))}
	for depIndex, depPath := range depPaths {
		depId := strings.TrimRight(strings.TrimPrefix(depPath, deployments.DeploymentKVPrefix), "/ ")
		link := newAtomLink(LINK_REL_DEPLOYMENT, "/deployments" + depId)
		depCol.Deployments[depIndex] = link
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(depCol)
}

func storeConsulKey(kv *api.KV, key, value string) {
	// PUT a new KV pair
	p := &api.KVPair{Key: key, Value: []byte(value)}
	if _, err := kv.Put(p, nil); err != nil {
		log.Panic(err)
	}
}

func (s *Server) storeDeploymentDefinition(topology tosca.Topology, id string) {
	// Get a handle to the KV API
	kv := s.consulClient.KV()
	prefix := strings.Join([]string{deployments.DeploymentKVPrefix, id}, "/")
	topologyPrefix := strings.Join([]string{prefix, "topology"}, "/")

	storeConsulKey(kv, topologyPrefix + "/tosca_version", topology.TOSCAVersion)
	storeConsulKey(kv, topologyPrefix + "/description", topology.Description)
	storeConsulKey(kv, topologyPrefix + "/name", topology.Name)
	storeConsulKey(kv, topologyPrefix + "/version", topology.Version)
	storeConsulKey(kv, topologyPrefix + "/author", topology.Author)
	// TODO deal with imports

	nodesPrefix := strings.Join([]string{topologyPrefix, "nodes"}, "/")
	for nodeName, node := range topology.TopologyTemplate.NodeTemplates {
		nodePrefix := nodesPrefix + "/" + nodeName
		storeConsulKey(kv, nodePrefix + "/name", nodeName)
		storeConsulKey(kv, nodePrefix + "/type", node.Type)
		propertiesPrefix := nodePrefix + "/properties"
		for propName, propValue := range node.Properties {
			storeConsulKey(kv, propertiesPrefix + "/" + propName, propValue)
		}
		capabilitiesPrefix := nodePrefix + "/capabilities"
		for capName, capability := range node.Capabilities {
			capabilityPrefix := capabilitiesPrefix + "/" + capName
			capabilityPropsPrefix := capabilityPrefix + "/properties"
			for propName, propValue := range capability.Properties {
				storeConsulKey(kv, capabilityPropsPrefix + "/" + propName, propValue)
			}
			capabilityAttrPrefix := capabilityPrefix + "/attributes"
			for attrName, attrValue := range capability.Attributes {
				storeConsulKey(kv, capabilityAttrPrefix + "/" + attrName, attrValue)
			}
		}
	}

	workflowsPrefix := strings.Join([]string{deployments.DeploymentKVPrefix, id, "workflows"}, "/")
	for wfName, workflow := range topology.TopologyTemplate.Workflows {
		workflowPrefix := workflowsPrefix + "/" + wfName
		for stepName, step := range workflow.Steps {
			stepPrefix := workflowPrefix + "/steps/" + stepName
			storeConsulKey(kv, stepPrefix + "/node", step.Node)
			if step.Activity.CallOperation != "" {
				storeConsulKey(kv, stepPrefix + "/activity/operation", step.Activity.CallOperation)
			}
			if step.Activity.Delegate != "" {
				storeConsulKey(kv, stepPrefix + "/activity/delegate", step.Activity.Delegate)
			}
			if step.Activity.CallOperation != "" {
				storeConsulKey(kv, stepPrefix + "/activity/set-state", step.Activity.SetState)
			}
			for nextId, next := range step.OnSuccess {
				storeConsulKey(kv, fmt.Sprintf("%s/next/%s-%s", stepPrefix, nextId, next), "")
			}
		}
	}

}
