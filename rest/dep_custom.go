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
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"github.com/ystia/yorc/v3/helper/collections"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/deployments"
	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/prov/operations"
	"github.com/ystia/yorc/v3/tasks"
)

func (s *Server) newCustomCommandHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")

	dExits, err := deployments.DoesDeploymentExists(s.consulClient.KV(), id)
	if err != nil {
		log.Panicf("%v", err)
	}
	if !dExits {
		writeError(w, r, errNotFound)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Panic(err)
	}

	var ccRequest CustomCommandRequest
	if err = json.Unmarshal(body, &ccRequest); err != nil {
		log.Panic(err)
	}
	// Check that provided node exists
	nodeName := ccRequest.NodeName
	nodeExists, err := deployments.DoesNodeExist(s.consulClient.KV(), id, nodeName)
	if err != nil {
		log.Panicf("%v", err)
	}
	if !nodeExists {
		writeError(w, r, newBadRequestParameter("node", errors.Errorf("Node %q must exist", nodeName)))
		return
	}

	// Get node instances on which the command is to be applied
	var instances []string
	if ccRequest.Instances == nil {
		// Apply command on all the instances
		instances, err = deployments.GetNodeInstancesIds(s.consulClient.KV(), id, nodeName)
		if err != nil {
			log.Panic(err)
		}
	} else {
		checked, inexistent := s.checkInstances(id, nodeName, ccRequest.Instances)
		if checked {
			instances = ccRequest.Instances
		} else {
			writeError(w, r, newBadRequestParameter("instance", errors.Errorf("Instance %q must exist", inexistent)))
			return
		}
	}

	ccRequest.InterfaceName = strings.ToLower(ccRequest.InterfaceName)
	inputsName, err := s.getInputNameFromCustom(id, nodeName, ccRequest.InterfaceName, ccRequest.CustomCommandName)
	if err != nil {
		log.Panic(err)
	}

	data := make(map[string]string)
	data[path.Join("nodes", nodeName)] = strings.Join(instances, ",")
	data["commandName"] = ccRequest.CustomCommandName
	data["interfaceName"] = ccRequest.InterfaceName

	for _, name := range inputsName {
		if err != nil {
			log.Panic(err)
		}
		data[path.Join("inputs", name)] = ccRequest.Inputs[name].String()
	}

	taskID, err := s.tasksCollector.RegisterTaskWithData(id, tasks.TaskTypeCustomCommand, data)
	if err != nil {
		if ok, _ := tasks.IsAnotherLivingTaskAlreadyExistsError(err); ok {
			writeError(w, r, newBadRequestError(err))
			return
		}
		log.Panic(err)
	}

	w.Header().Set("Location", fmt.Sprintf("/deployments/%s/tasks/%s", id, taskID))
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) getInputNameFromCustom(deploymentID, nodeName, interfaceName, customCName string) ([]string, error) {
	kv := s.consulClient.KV()
	op, err := operations.GetOperation(context.Background(), kv, deploymentID, nodeName, interfaceName+"."+customCName, "", "")
	if err != nil {
		return nil, err
	}
	inputs, err := deployments.GetOperationInputs(kv, deploymentID, op.ImplementedInNodeTemplate, op.ImplementedInType, op.Name)
	if err != nil {
		log.Panic(err)
	}

	result := inputs[:0]
	for _, inputName := range inputs {
		isPropDef, err := deployments.IsOperationInputAPropertyDefinition(kv, deploymentID, op.ImplementedInNodeTemplate, op.ImplementedInType, op.Name, inputName)
		if err != nil {
			return nil, err
		}
		if isPropDef {
			result = append(result, inputName)
		}
	}
	return result, nil
}

// checkInstances checks if provided instances exist and returns true in this case.
// If a provided instance does not exists, returns false and the inexistent instance name
func (s *Server) checkInstances(deploymentID, nodeName string, provInstances []string) (bool, string) {
	// Get known instances for the provided node
	allInstances, err := deployments.GetNodeInstancesIds(s.consulClient.KV(), deploymentID, nodeName)
	if err != nil {
		log.Panic(err)
	}
	// Check that provided instances exist
	for _, provInstance := range provInstances {
		if !collections.ContainsString(allInstances, provInstance) {
			return false, provInstance
		}
	}
	return true, ""
}
