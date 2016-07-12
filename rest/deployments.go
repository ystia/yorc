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
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"novaforge.bull.com/starlings-janus/janus/tosca"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func extractFile(f *zip.File, path string) {
	fileReader, err := f.Open()
	if err != nil {
		log.Panic(err)
	}
	defer fileReader.Close()

	targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
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
	uploadPath := filepath.Join("work", "deployments", uuid)
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
	destDir := filepath.Join(uploadPath, "overlay")
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
		path := filepath.Join(destDir, f.Name)
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

	log.Debugf("%+v", topology)

	storeConsulKey(s.consulClient.KV(), deployments.DeploymentKVPrefix+"/"+uuid+"/status", fmt.Sprint(deployments.INITIAL))

	s.storeDeploymentDefinition(topology, uuid, false, "")

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
	result, _, err := kv.Get(deployments.DeploymentKVPrefix+"/"+id+"/status", nil)
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
	depPaths, _, err := kv.Keys(deployments.DeploymentKVPrefix+"/", "/", nil)
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
		link := newAtomLink(LINK_REL_DEPLOYMENT, "/deployments"+depId)
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

func storePropertyDefinition(kv *api.KV, propPrefix, propName string, propDefinition tosca.PropertyDefinition) {
	storeConsulKey(kv, propPrefix+"/name", propName)
	storeConsulKey(kv, propPrefix+"/description", propDefinition.Description)
	storeConsulKey(kv, propPrefix+"/type", propDefinition.Type)
	storeConsulKey(kv, propPrefix+"/default", propDefinition.Default)
	storeConsulKey(kv, propPrefix+"/required", fmt.Sprint(propDefinition.Required))
}

func storeAttributeDefinition(kv *api.KV, attrPrefix, attrName string, attrDefinition tosca.AttributeDefinition) {
	storeConsulKey(kv, attrPrefix+"/name", attrName)
	storeConsulKey(kv, attrPrefix+"/description", attrDefinition.Description)
	storeConsulKey(kv, attrPrefix+"/type", attrDefinition.Type)
	storeConsulKey(kv, attrPrefix+"/default", attrDefinition.Default)
	storeConsulKey(kv, attrPrefix+"/status", attrDefinition.Status)
}

func (s *Server) storeDeploymentDefinition(topology tosca.Topology, id string, imports bool, pathImport string) {
	// Get a handle to the KV API
	kv := s.consulClient.KV()
	prefix := path.Join(deployments.DeploymentKVPrefix, id)
	topologyPrefix := path.Join(prefix, "topology")

	storeConsulKey(kv, topologyPrefix+"/tosca_version", topology.TOSCAVersion)
	storeConsulKey(kv, topologyPrefix+"/description", topology.Description)
	storeConsulKey(kv, topologyPrefix+"/name", topology.Name)
	storeConsulKey(kv, topologyPrefix+"/version", topology.Version)
	storeConsulKey(kv, topologyPrefix+"/author", topology.Author)
	// Imports
	log.Debug(topology.Imports)
	for _, element := range topology.Imports  {
		for _, value := range element {
			topology := tosca.Topology{}

			uploadFile := filepath.Join("work", "deployments", id, "overlay", value.File)

			definition, err := os.Open(uploadFile)
			if err != nil {
				panic(err)
			}

			defBytes, err := ioutil.ReadAll(definition)

			err = yaml.Unmarshal(defBytes, &topology)

			log.Debugf("%+v", topology)

			s.storeDeploymentDefinition(topology, id, true, filepath.Dir(value.File))
		}
	}

	nodesPrefix := path.Join(topologyPrefix, "nodes")
	for nodeName, node := range topology.TopologyTemplate.NodeTemplates {
		nodePrefix := nodesPrefix + "/" + nodeName
		storeConsulKey(kv, nodePrefix+"/name", nodeName)
		storeConsulKey(kv, nodePrefix+"/type", node.Type)
		propertiesPrefix := nodePrefix + "/properties"
		for propName, propValue := range node.Properties {
			storeConsulKey(kv, propertiesPrefix+"/"+propName, fmt.Sprint(propValue))
		}
		attributesPrefix := nodePrefix + "/attributes"
		for attrName, attrValue := range node.Attributes {
			storeConsulKey(kv, attributesPrefix+"/"+attrName, fmt.Sprint(attrValue))
		}
		capabilitiesPrefix := nodePrefix + "/capabilities"
		for capName, capability := range node.Capabilities {
			capabilityPrefix := capabilitiesPrefix + "/" + capName
			capabilityPropsPrefix := capabilityPrefix + "/properties"
			for propName, propValue := range capability.Properties {
				storeConsulKey(kv, capabilityPropsPrefix+"/"+propName, fmt.Sprint(propValue))
			}
			capabilityAttrPrefix := capabilityPrefix + "/attributes"
			for attrName, attrValue := range capability.Attributes {
				storeConsulKey(kv, capabilityAttrPrefix+"/"+attrName, fmt.Sprint(attrValue))
			}
		}
		requirementsPrefix := nodePrefix + "/requirements"
		for _, reqValueMap := range node.Requirements {
			for reqName, reqValue := range reqValueMap {
				reqPrefix := requirementsPrefix + "/" + reqName
				storeConsulKey(kv, reqPrefix+"/name", reqName)
				storeConsulKey(kv, reqPrefix+"/node", reqValue.Node)
				storeConsulKey(kv, reqPrefix+"/relationship", reqValue.Relationship)
				storeConsulKey(kv, reqPrefix+"/capability", reqValue.Capability)
				for propName, propValue := range reqValue.RelationshipProps {
					storeConsulKey(kv, reqPrefix + "/properties/" + propName, fmt.Sprint(propValue))
				}
			}
		}
		artifactsPrefix := nodePrefix + "/artifacts"
		for artName, artDef := range node.Artifacts {
			artPrefix := artifactsPrefix + "/" + artName
			storeConsulKey(kv, artPrefix+"/name", artName)
			storeConsulKey(kv, artPrefix+"/description", artDef.Description)
			if imports {
				storeConsulKey(kv, artPrefix+"/file", filepath.Join(pathImport,artDef.File))
			} else {
				storeConsulKey(kv, artPrefix+"/file", artDef.File)
			}
			storeConsulKey(kv, artPrefix+"/type", artDef.Type)
			storeConsulKey(kv, artPrefix+"/repository", artDef.Repository)
			storeConsulKey(kv, artPrefix+"/deploy_path", artDef.DeployPath)
		}
	}

	nodesTypesPrefix := path.Join(topologyPrefix, "types")
	for nodeTypeName, nodeType := range topology.NodeTypes {
		nodeTypePrefix := nodesTypesPrefix + "/" + nodeTypeName
		storeConsulKey(kv, nodeTypePrefix+"/name", nodeTypeName)
		storeConsulKey(kv, nodeTypePrefix+"/derived_from", nodeType.DerivedFrom)
		storeConsulKey(kv, nodeTypePrefix+"/description", nodeType.Description)
		storeConsulKey(kv, nodeTypePrefix+"/version", nodeType.Version)
		propertiesPrefix := nodeTypePrefix + "/properties"
		for propName, propDefinition := range nodeType.Properties {
			propPrefix := propertiesPrefix + "/" + propName
			storePropertyDefinition(kv, propPrefix, propName, propDefinition)
		}
		attributesPrefix := nodeTypePrefix + "/attributes"
		for attrName, attrDefinition := range nodeType.Attributes {
			attrPrefix := attributesPrefix + "/" + attrName
			storeAttributeDefinition(kv, attrPrefix, attrName, attrDefinition)
		}

		requirementsPrefix := nodeTypePrefix + "/requirements"
		for reqName, reqDefinition := range nodeType.Requirements {
			reqPrefix := requirementsPrefix + "/" + reqName
			storeConsulKey(kv, reqPrefix+"/name", reqName)
			storeConsulKey(kv, reqPrefix+"/node", reqDefinition.Node)
			storeConsulKey(kv, reqPrefix+"/occurences", reqDefinition.Occurrences)
			storeConsulKey(kv, reqPrefix+"/relationship", reqDefinition.Relationship)
			storeConsulKey(kv, reqPrefix+"/capability", reqDefinition.Capability)
		}

		capabilitiesPrefix := nodeTypePrefix + "/capabilities"
		for capName, capability := range nodeType.Capabilities {
			capabilityPrefix := capabilitiesPrefix + "/" + capName

			storeConsulKey(kv, capabilityPrefix+"/name", capName)
			storeConsulKey(kv, capabilityPrefix+"/type", capability.Type)
			storeConsulKey(kv, capabilityPrefix+"/description", capability.Description)
			storeConsulKey(kv, capabilityPrefix+"/occurences", capability.Occurrences)
			storeConsulKey(kv, capabilityPrefix+"/valid_sources", strings.Join(capability.ValidSourceTypes, ","))
			capabilityPropsPrefix := capabilityPrefix + "/properties"
			for propName, propValue := range capability.Properties {
				propPrefix := capabilityPropsPrefix + "/" + propName
				storePropertyDefinition(kv, propPrefix, propName, propValue)
			}
			capabilityAttrPrefix := capabilityPrefix + "/attributes"
			for attrName, attrValue := range capability.Attributes {
				attrPrefix := capabilityAttrPrefix + "/" + attrName
				storeAttributeDefinition(kv, attrPrefix, attrName, attrValue)
			}
		}

		interfacesPrefix := nodeTypePrefix + "/interfaces"
		for intTypeName, intMap := range nodeType.Interfaces {
			for intName, intDef := range intMap {
				intPrefix := path.Join(interfacesPrefix, intTypeName, intName)
				storeConsulKey(kv, intPrefix+"/name", intName)
				storeConsulKey(kv, intPrefix+"/description", intDef.Description)

				for inputName, inputDef := range intDef.Inputs {
					inputPrefix := path.Join(intPrefix, "inputs", inputName)
					storeConsulKey(kv, inputPrefix+"/name", inputName)
					storeConsulKey(kv, inputPrefix+"/expression", inputDef.String())
				}
				if imports {
					storeConsulKey(kv, intPrefix+"/implementation/primary", filepath.Join(pathImport,intDef.Implementation.Primary))
				} else {
					storeConsulKey(kv, intPrefix+"/implementation/primary", intDef.Implementation.Primary)
				}
				storeConsulKey(kv, intPrefix+"/implementation/dependencies", strings.Join(intDef.Implementation.Dependencies, ","))
			}
		}

		artifactsPrefix := nodeTypePrefix + "/artifacts"
		for artName, artDef := range nodeType.Artifacts {
			artPrefix := artifactsPrefix + "/" + artName
			storeConsulKey(kv, artPrefix+"/name", artName)
			storeConsulKey(kv, artPrefix+"/description", artDef.Description)
			if imports {
				storeConsulKey(kv, artPrefix+"/file", filepath.Join(pathImport,artDef.File))
			} else {
				storeConsulKey(kv, artPrefix+"/file", artDef.File)
			}
			storeConsulKey(kv, artPrefix+"/type", artDef.Type)
			storeConsulKey(kv, artPrefix+"/repository", artDef.Repository)
			storeConsulKey(kv, artPrefix+"/deploy_path", artDef.DeployPath)
		}

	}

	workflowsPrefix := path.Join(deployments.DeploymentKVPrefix, id, "workflows")
	for wfName, workflow := range topology.TopologyTemplate.Workflows {
		workflowPrefix := workflowsPrefix + "/" + wfName
		for stepName, step := range workflow.Steps {
			stepPrefix := workflowPrefix + "/steps/" + stepName
			storeConsulKey(kv, stepPrefix+"/node", step.Node)
			if step.Activity.CallOperation != "" {
				storeConsulKey(kv, stepPrefix+"/activity/operation", step.Activity.CallOperation)
			}
			if step.Activity.Delegate != "" {
				storeConsulKey(kv, stepPrefix+"/activity/delegate", step.Activity.Delegate)
			}
			if step.Activity.SetState != "" {
				storeConsulKey(kv, stepPrefix+"/activity/set-state", step.Activity.SetState)
			}
			for _, next := range step.OnSuccess {
				storeConsulKey(kv, fmt.Sprintf("%s/next/%s", stepPrefix, next), "")
			}
		}
	}

}
