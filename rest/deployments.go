package rest

import (
	"archive/zip"
	"encoding/json"
	"fmt"
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
	"strconv"
	"strings"
	"sync"
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
			// Ensure that we have full rights on directory to be able to extract files into them
			if err = os.MkdirAll(path, f.Mode()|0700); err != nil {
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
	errCh := make(chan error, 30)
	wg := sync.WaitGroup{}
	s.storeConsulKey(errCh, &wg, deployments.DeploymentKVPrefix+"/"+uuid+"/status", fmt.Sprint(deployments.INITIAL))

	wg.Add(1)
	go s.storeDeploymentDefinition(topology, uuid, false, "", &wg, errCh)

	doneCh := make(chan []error, 1)

	go func() {
		errList := make([]error, 0)
		for err := range errCh {
			errList = append(errList, err)
		}
		doneCh <- errList
		close(doneCh)
	}()

	// Wait for all consul records to be stored
	wg.Wait()
	// Close errCh and retrieve errors from doneCh
	close(errCh)

	errors := <-doneCh
	if len(errors) > 0 {
		log.Panicf("Errors encountred during the YAML definition parsing and storage: %v", errors)
	}

	if err := s.tasksCollector.RegisterTask(uuid, tasks.DEPLOY); err != nil {
		log.Panic(err)
	}

	w.Header().Set("Location", fmt.Sprintf("/deployments/%s", uuid))
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) deleteDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	if err := s.tasksCollector.RegisterTask(id, tasks.UNDEPLOY); err != nil {
		log.Panic(err)
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) getDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
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

func (s *Server) getOutputHandler(w http.ResponseWriter, r *http.Request) {

	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	opt := params.ByName("opt")

	kv := s.consulClient.KV()
	expression, _, err := kv.Get(deployments.DeploymentKVPrefix+"/"+id+"/topology/outputs/"+opt, nil)
	if err != nil {
		log.Panic(err)
	}
	if expression == nil {
		WriteError(w, ErrNotFound)
		return
	}

	expressionOpt := string(expression.Value)
	expressionString := fmt.Sprintf("%s", expressionOpt)
	str := strings.Split(expressionString, ":")
	functionOpt, expressionCore := str[0], str[1]

	if functionOpt == "get_attribute" {
		functionCore := "attributes"

		expressionCore = strings.TrimPrefix(expressionCore, "[")
		expressionCore = strings.TrimSuffix(expressionCore, "]")
		express := strings.Split(expressionCore, ",")
		node, varOpt := express[0], express[1]

		result, _, err := kv.Get(deployments.DeploymentKVPrefix+"/"+id+"/topology/nodes/"+node+"/"+functionCore+"/"+varOpt, nil)
		if err != nil {
			log.Panic(err)
		}

		output := Output{Name: opt, Value: string(result.Value)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(output)
	}
	if functionOpt == "get_property" {
		functionCore := "properties"

		expressionCore = strings.TrimPrefix(expressionCore, "[")
		expressionCore = strings.TrimSuffix(expressionCore, "]")
		express := strings.Split(expressionCore, ",")
		node, varOpt := express[0], express[1]

		result, _, err := kv.Get(deployments.DeploymentKVPrefix+"/"+id+"/topology/nodes/"+node+"/"+functionCore+"/"+varOpt, nil)
		if err != nil {
			log.Panic(err)
		}

		output := Output{Name: opt, Value: string(result.Value)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(output)
	}

}

func (s *Server) listOutputsHandler(w http.ResponseWriter, r *http.Request) {

	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	kv := s.consulClient.KV()

	optPath := deployments.DeploymentKVPrefix + "/" + id + "/topology/outputs/"
	optPaths, _, err := kv.Keys(optPath, "/", nil)
	if err != nil {
		log.Panic(err)

	}
	if len(optPaths) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	optCol := OutputsCollection{Outputs: make([]AtomLink, len(optPaths))}
	for optIndex, optP := range optPaths {
		optName := strings.TrimRight(strings.TrimPrefix(optP, deployments.DeploymentKVPrefix), "/ ")
		pre := "/" + id + "/topology/"
		RoptName := strings.TrimPrefix(optName, pre)

		link := newAtomLink(LINK_REL_OUTPUT, "/deployments/"+id+"/"+RoptName)
		optCol.Outputs[optIndex] = link
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(optCol)

}

func (s *Server) storePropertyDefinition(errCh chan error, wg *sync.WaitGroup, propPrefix, propName string, propDefinition tosca.PropertyDefinition) {
	s.storeConsulKey(errCh, wg, propPrefix+"/name", propName)
	s.storeConsulKey(errCh, wg, propPrefix+"/description", propDefinition.Description)
	s.storeConsulKey(errCh, wg, propPrefix+"/type", propDefinition.Type)
	s.storeConsulKey(errCh, wg, propPrefix+"/default", propDefinition.Default)
	s.storeConsulKey(errCh, wg, propPrefix+"/required", fmt.Sprint(propDefinition.Required))
}

func (s *Server) storeAttributeDefinition(errCh chan error, wg *sync.WaitGroup, attrPrefix, attrName string, attrDefinition tosca.AttributeDefinition) {
	s.storeConsulKey(errCh, wg, attrPrefix+"/name", attrName)
	s.storeConsulKey(errCh, wg, attrPrefix+"/description", attrDefinition.Description)
	s.storeConsulKey(errCh, wg, attrPrefix+"/type", attrDefinition.Type.Expression.Value)
	s.storeConsulKey(errCh, wg, attrPrefix+"/default", attrDefinition.Default)
	s.storeConsulKey(errCh, wg, attrPrefix+"/status", attrDefinition.Status)
}

func (s *Server) storeDeploymentDefinition(topology tosca.Topology, id string, imports bool, pathImport string, wg *sync.WaitGroup, errCh chan error) {
	prefix := path.Join(deployments.DeploymentKVPrefix, id)
	topologyPrefix := path.Join(prefix, "topology")

	s.storeConsulKey(errCh, wg, topologyPrefix+"/tosca_version", topology.TOSCAVersion)
	s.storeConsulKey(errCh, wg, topologyPrefix+"/description", topology.Description)
	s.storeConsulKey(errCh, wg, topologyPrefix+"/name", topology.Name)
	s.storeConsulKey(errCh, wg, topologyPrefix+"/version", topology.Version)
	s.storeConsulKey(errCh, wg, topologyPrefix+"/author", topology.Author)
	// Imports
	log.Debug(topology.Imports)
	for _, element := range topology.Imports {
		for _, value := range element {
			topology := tosca.Topology{}
			importValue := strings.Trim(value.File, " \t")
			if strings.HasPrefix(importValue, "<") && strings.HasSuffix(importValue, ">") {
				// Internal import
				importValue = strings.Trim(importValue, "<>")
				var defBytes []byte
				var err error
				if defBytes, err = tosca.Asset(importValue); err != nil {
					panic(fmt.Errorf("Failed to import internal definition %s: %v", importValue, err))
				}
				if err = yaml.Unmarshal(defBytes, &topology); err != nil {
					panic(fmt.Errorf("Failed to parse internal definition %s: %v", importValue, err))
				}
				log.Debugf("%+v", topology)
				wg.Add(1)
				go s.storeDeploymentDefinition(topology, id, false, "", wg, errCh)
			} else {
				uploadFile := filepath.Join("work", "deployments", id, "overlay", value.File)

				definition, err := os.Open(uploadFile)
				if err != nil {
					panic(err)
				}

				defBytes, err := ioutil.ReadAll(definition)

				err = yaml.Unmarshal(defBytes, &topology)
				log.Debugf("%+v", topology)
				wg.Add(1)
				go s.storeDeploymentDefinition(topology, id, true, filepath.Dir(value.File), wg, errCh)
			}

		}
	}

	nodesPrefix := path.Join(topologyPrefix, "nodes")
	for nodeName, node := range topology.TopologyTemplate.NodeTemplates {
		nodePrefix := nodesPrefix + "/" + nodeName
		s.storeConsulKey(errCh, wg, nodePrefix+"/name", nodeName)
		s.storeConsulKey(errCh, wg, nodePrefix+"/type", node.Type)
		propertiesPrefix := nodePrefix + "/properties"
		for propName, propValue := range node.Properties {
			s.storeConsulKey(errCh, wg, propertiesPrefix+"/"+propName, fmt.Sprint(propValue))
		}
		attributesPrefix := nodePrefix + "/attributes"
		for attrName, attrValue := range node.Attributes {
			s.storeConsulKey(errCh, wg, attributesPrefix+"/"+attrName, fmt.Sprint(attrValue))
		}
		capabilitiesPrefix := nodePrefix + "/capabilities"
		for capName, capability := range node.Capabilities {
			capabilityPrefix := capabilitiesPrefix + "/" + capName
			capabilityPropsPrefix := capabilityPrefix + "/properties"
			for propName, propValue := range capability.Properties {
				s.storeConsulKey(errCh, wg, capabilityPropsPrefix+"/"+propName, fmt.Sprint(propValue))
			}
			capabilityAttrPrefix := capabilityPrefix + "/attributes"
			for attrName, attrValue := range capability.Attributes {
				s.storeConsulKey(errCh, wg, capabilityAttrPrefix+"/"+attrName, fmt.Sprint(attrValue))
			}
		}
		requirementsPrefix := nodePrefix + "/requirements"
		for _, reqValueMap := range node.Requirements {
			for reqName, reqValue := range reqValueMap {
				reqPrefix := requirementsPrefix + "/" + reqName
				s.storeConsulKey(errCh, wg, reqPrefix+"/name", reqName)
				s.storeConsulKey(errCh, wg, reqPrefix+"/node", reqValue.Node)
				s.storeConsulKey(errCh, wg, reqPrefix+"/relationship", reqValue.Relationship)
				s.storeConsulKey(errCh, wg, reqPrefix+"/capability", reqValue.Capability)
				for propName, propValue := range reqValue.RelationshipProps {
					s.storeConsulKey(errCh, wg, reqPrefix+"/properties/"+propName, fmt.Sprint(propValue))
				}
			}
		}
		artifactsPrefix := nodePrefix + "/artifacts"
		for artName, artDef := range node.Artifacts {
			artPrefix := artifactsPrefix + "/" + artName
			s.storeConsulKey(errCh, wg, artPrefix+"/name", artName)
			s.storeConsulKey(errCh, wg, artPrefix+"/metatype", "artifact")
			s.storeConsulKey(errCh, wg, artPrefix+"/description", artDef.Description)
			if imports {
				s.storeConsulKey(errCh, wg, artPrefix+"/file", filepath.Join(pathImport, artDef.File))
			} else {
				s.storeConsulKey(errCh, wg, artPrefix+"/file", artDef.File)
			}
			s.storeConsulKey(errCh, wg, artPrefix+"/type", artDef.Type)
			s.storeConsulKey(errCh, wg, artPrefix+"/repository", artDef.Repository)
			s.storeConsulKey(errCh, wg, artPrefix+"/deploy_path", artDef.DeployPath)
		}
	}

	typesPrefix := path.Join(topologyPrefix, "types")
	for nodeTypeName, nodeType := range topology.NodeTypes {
		nodeTypePrefix := typesPrefix + "/" + nodeTypeName
		s.storeConsulKey(errCh, wg, nodeTypePrefix+"/name", nodeTypeName)
		s.storeConsulKey(errCh, wg, nodeTypePrefix+"/derived_from", nodeType.DerivedFrom)
		s.storeConsulKey(errCh, wg, nodeTypePrefix+"/description", nodeType.Description)
		s.storeConsulKey(errCh, wg, nodeTypePrefix+"/metatype", "Node")
		s.storeConsulKey(errCh, wg, nodeTypePrefix+"/version", nodeType.Version)
		propertiesPrefix := nodeTypePrefix + "/properties"
		for propName, propDefinition := range nodeType.Properties {
			propPrefix := propertiesPrefix + "/" + propName
			s.storePropertyDefinition(errCh, wg, propPrefix, propName, propDefinition)
		}
		attributesPrefix := nodeTypePrefix + "/attributes"
		for attrName, attrDefinition := range nodeType.Attributes {
			attrPrefix := attributesPrefix + "/" + attrName
			if attrDefinition.Type.Expression.IsLiteral() {
				s.storeAttributeDefinition(errCh, wg, attrPrefix, attrName, attrDefinition)
			} else if attrDefinition.Type.Expression.Value == "get_operation_output" {
				s.storeConsulKey(errCh, wg, nodeTypePrefix+"/output/"+attrDefinition.Type.Expression.Children()[1].Value+"/"+attrDefinition.Type.Expression.Children()[2].Value+"/"+attrDefinition.Type.Expression.Children()[3].Value, attrDefinition.Type.Expression.Children()[3].Value)
			}
		}

		requirementsPrefix := nodeTypePrefix + "/requirements"
		for _, reqMap := range nodeType.Requirements {
			for reqName, reqDefinition := range reqMap {
				reqPrefix := requirementsPrefix + "/" + reqName
				s.storeConsulKey(errCh, wg, reqPrefix+"/name", reqName)
				s.storeConsulKey(errCh, wg, reqPrefix+"/node", reqDefinition.Node)
				s.storeConsulKey(errCh, wg, reqPrefix+"/occurences/lower_bound", strconv.FormatUint(reqDefinition.Occurrences.LowerBound, 10))
				s.storeConsulKey(errCh, wg, reqPrefix+"/occurences/upper_bound", strconv.FormatUint(reqDefinition.Occurrences.UpperBound, 10))
				s.storeConsulKey(errCh, wg, reqPrefix+"/relationship", reqDefinition.Relationship)
				s.storeConsulKey(errCh, wg, reqPrefix+"/capability", reqDefinition.Capability)
			}
		}
		capabilitiesPrefix := nodeTypePrefix + "/capabilities"
		for capName, capability := range nodeType.Capabilities {
			capabilityPrefix := capabilitiesPrefix + "/" + capName

			s.storeConsulKey(errCh, wg, capabilityPrefix+"/name", capName)
			s.storeConsulKey(errCh, wg, capabilityPrefix+"/type", capability.Type)
			s.storeConsulKey(errCh, wg, capabilityPrefix+"/description", capability.Description)
			s.storeConsulKey(errCh, wg, capabilityPrefix+"/occurences/lower_bound", strconv.FormatUint(capability.Occurrences.LowerBound, 10))
			s.storeConsulKey(errCh, wg, capabilityPrefix+"/occurences/upper_bound", strconv.FormatUint(capability.Occurrences.UpperBound, 10))
			s.storeConsulKey(errCh, wg, capabilityPrefix+"/valid_sources", strings.Join(capability.ValidSourceTypes, ","))
			capabilityPropsPrefix := capabilityPrefix + "/properties"
			for propName, propValue := range capability.Properties {
				propPrefix := capabilityPropsPrefix + "/" + propName
				s.storePropertyDefinition(errCh, wg, propPrefix, propName, propValue)
			}
			capabilityAttrPrefix := capabilityPrefix + "/attributes"
			for attrName, attrValue := range capability.Attributes {
				attrPrefix := capabilityAttrPrefix + "/" + attrName
				s.storeAttributeDefinition(errCh, wg, attrPrefix, attrName, attrValue)
			}
		}

		interfacesPrefix := nodeTypePrefix + "/interfaces"
		for intTypeName, intMap := range nodeType.Interfaces {
			for intName, intDef := range intMap {
				intPrefix := path.Join(interfacesPrefix, intTypeName, intName)
				s.storeConsulKey(errCh, wg, intPrefix+"/name", intName)
				s.storeConsulKey(errCh, wg, intPrefix+"/description", intDef.Description)

				for inputName, inputDef := range intDef.Inputs {
					inputPrefix := path.Join(intPrefix, "inputs", inputName)
					s.storeConsulKey(errCh, wg, inputPrefix+"/name", inputName)
					s.storeConsulKey(errCh, wg, inputPrefix+"/expression", inputDef.String())
				}
				if imports {
					s.storeConsulKey(errCh, wg, intPrefix+"/implementation/primary", filepath.Join(pathImport, intDef.Implementation.Primary))
				} else {
					s.storeConsulKey(errCh, wg, intPrefix+"/implementation/primary", intDef.Implementation.Primary)
				}
				s.storeConsulKey(errCh, wg, intPrefix+"/implementation/dependencies", strings.Join(intDef.Implementation.Dependencies, ","))
			}
		}

		artifactsPrefix := nodeTypePrefix + "/artifacts"
		for artName, artDef := range nodeType.Artifacts {
			artPrefix := artifactsPrefix + "/" + artName
			s.storeConsulKey(errCh, wg, artPrefix+"/name", artName)
			s.storeConsulKey(errCh, wg, artPrefix+"/description", artDef.Description)
			if imports {
				s.storeConsulKey(errCh, wg, artPrefix+"/file", filepath.Join(pathImport, artDef.File))
			} else {
				s.storeConsulKey(errCh, wg, artPrefix+"/file", artDef.File)
			}
			s.storeConsulKey(errCh, wg, artPrefix+"/type", artDef.Type)
			s.storeConsulKey(errCh, wg, artPrefix+"/repository", artDef.Repository)
			s.storeConsulKey(errCh, wg, artPrefix+"/deploy_path", artDef.DeployPath)
		}

	}

	for relationName, relationType := range topology.RelationshipTypes {
		relationTypePrefix := typesPrefix + "/" + relationName
		s.storeConsulKey(errCh, wg, relationTypePrefix+"/name", relationName)
		s.storeConsulKey(errCh, wg, relationTypePrefix+"/derived_from", relationType.DerivedFrom)
		s.storeConsulKey(errCh, wg, relationTypePrefix+"/description", relationType.Description)
		s.storeConsulKey(errCh, wg, relationTypePrefix+"/version", relationType.Version)
		s.storeConsulKey(errCh, wg, relationTypePrefix+"/metatype", "Relationship")
		propertiesPrefix := relationTypePrefix + "/properties"
		for propName, propDefinition := range relationType.Properties {
			propPrefix := propertiesPrefix + "/" + propName
			s.storePropertyDefinition(errCh, wg, propPrefix, propName, propDefinition)
		}
		attributesPrefix := relationTypePrefix + "/attributes"
		for attrName, attrDefinition := range relationType.Attributes {
			attrPrefix := attributesPrefix + "/" + attrName
			if attrDefinition.Type.Expression.IsLiteral() {
				s.storeAttributeDefinition(errCh, wg, attrPrefix, attrName, attrDefinition)
			} else if attrDefinition.Type.Expression.Value == "get_operation_output" {
				s.storeConsulKey(errCh, wg, relationTypePrefix+"/output/"+attrDefinition.Type.Expression.Children()[1].Value+"/"+attrDefinition.Type.Expression.Children()[2].Value+"/"+attrDefinition.Type.Expression.Children()[3].Value, attrDefinition.Type.Expression.Children()[3].Value)
			}
		}

		interfacesPrefix := relationTypePrefix + "/interfaces"
		for intTypeName, intMap := range relationType.Interfaces {
			for intName, intDef := range intMap {
				intPrefix := path.Join(interfacesPrefix, intTypeName, intName)
				s.storeConsulKey(errCh, wg, intPrefix+"/name", intName)
				s.storeConsulKey(errCh, wg, intPrefix+"/description", intDef.Description)

				for inputName, inputDef := range intDef.Inputs {
					inputPrefix := path.Join(intPrefix, "inputs", inputName)
					s.storeConsulKey(errCh, wg, inputPrefix+"/name", inputName)
					s.storeConsulKey(errCh, wg, inputPrefix+"/expression", inputDef.String())
				}
				if imports {
					s.storeConsulKey(errCh, wg, intPrefix+"/implementation/primary", filepath.Join(pathImport, intDef.Implementation.Primary))
				} else {
					s.storeConsulKey(errCh, wg, intPrefix+"/implementation/primary", intDef.Implementation.Primary)
				}
				s.storeConsulKey(errCh, wg, intPrefix+"/implementation/dependencies", strings.Join(intDef.Implementation.Dependencies, ","))
			}
		}

		artifactsPrefix := relationTypePrefix + "/artifacts"
		for artName, artDef := range relationType.Artifacts {
			artPrefix := artifactsPrefix + "/" + artName
			s.storeConsulKey(errCh, wg, artPrefix+"/name", artName)
			s.storeConsulKey(errCh, wg, artPrefix+"/metatype", "artifact")
			s.storeConsulKey(errCh, wg, artPrefix+"/description", artDef.Description)
			if imports {
				s.storeConsulKey(errCh, wg, artPrefix+"/file", filepath.Join(pathImport, artDef.File))
			} else {
				s.storeConsulKey(errCh, wg, artPrefix+"/file", artDef.File)
			}
			s.storeConsulKey(errCh, wg, artPrefix+"/type", artDef.Type)
			s.storeConsulKey(errCh, wg, artPrefix+"/repository", artDef.Repository)
			s.storeConsulKey(errCh, wg, artPrefix+"/deploy_path", artDef.DeployPath)
		}

		s.storeConsulKey(errCh, wg, relationTypePrefix+"/valid_target_type", strings.Join(relationType.ValidTargetTypes, ", "))

	}

	for capabilityTypeName, capabilityType := range topology.CapabilityTypes {
		capabilityTypePrefix := typesPrefix + "/" + capabilityTypeName
		s.storeConsulKey(errCh, wg, capabilityTypePrefix+"/name", capabilityTypeName)
		s.storeConsulKey(errCh, wg, capabilityTypePrefix+"/derived_from", capabilityType.DerivedFrom)
		s.storeConsulKey(errCh, wg, capabilityTypePrefix+"/description", capabilityType.Description)
		s.storeConsulKey(errCh, wg, capabilityTypePrefix+"/version", capabilityType.Version)
		propertiesPrefix := capabilityTypePrefix + "/properties"
		for propName, propDefinition := range capabilityType.Properties {
			propPrefix := propertiesPrefix + "/" + propName
			s.storePropertyDefinition(errCh, wg, propPrefix, propName, propDefinition)
		}
		attributesPrefix := capabilityTypePrefix + "/attributes"
		for attrName, attrDefinition := range capabilityType.Attributes {
			attrPrefix := attributesPrefix + "/" + attrName
			s.storeAttributeDefinition(errCh, wg, attrPrefix, attrName, attrDefinition)
		}
		s.storeConsulKey(errCh, wg, capabilityTypePrefix+"/valid_source_types", strings.Join(capabilityType.ValidSourceTypes, ","))
	}

	workflowsPrefix := path.Join(deployments.DeploymentKVPrefix, id, "workflows")
	for wfName, workflow := range topology.TopologyTemplate.Workflows {
		workflowPrefix := workflowsPrefix + "/" + wfName
		for stepName, step := range workflow.Steps {
			stepPrefix := workflowPrefix + "/steps/" + stepName
			s.storeConsulKey(errCh, wg, stepPrefix+"/node", step.Node)
			if step.Activity.CallOperation != "" {
				s.storeConsulKey(errCh, wg, stepPrefix+"/activity/operation", step.Activity.CallOperation)
			}
			if step.Activity.Delegate != "" {
				s.storeConsulKey(errCh, wg, stepPrefix+"/activity/delegate", step.Activity.Delegate)
			}
			if step.Activity.SetState != "" {
				s.storeConsulKey(errCh, wg, stepPrefix+"/activity/set-state", step.Activity.SetState)
			}
			for _, next := range step.OnSuccess {
				s.storeConsulKey(errCh, wg, fmt.Sprintf("%s/next/%s", stepPrefix, next), "")
			}
		}
	}
	wg.Done()
}
