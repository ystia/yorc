package rest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"

	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tasks"
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

	var inputMap CustomCommandRequest
	if err = json.Unmarshal(body, &inputMap); err != nil {
		log.Panic(err)
	}

	inputsName, err := s.getInputNameFromCustom(id, inputMap.NodeName, inputMap.CustomCommandName)
	if err != nil {
		log.Panic(err)
	}

	data := make(map[string]string)

	// For now custom commands are for all instances
	instances, err := deployments.GetNodeInstancesIds(s.consulClient.KV(), id, inputMap.NodeName)
	data[path.Join("nodes", inputMap.NodeName)] = strings.Join(instances, ",")
	data["commandName"] = inputMap.CustomCommandName

	for _, name := range inputsName {
		if err != nil {
			log.Panic(err)
		}
		data[path.Join("inputs", name)] = inputMap.Inputs[name]
	}

	taskID, err := s.tasksCollector.RegisterTaskWithData(id, tasks.CustomCommand, data)
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

func (s *Server) getInputNameFromCustom(deploymentID, nodeName, customCName string) ([]string, error) {
	nodeType, err := deployments.GetNodeType(s.consulClient.KV(), deploymentID, nodeName)

	if err != nil {
		return nil, err
	}

	kv := s.consulClient.KV()
	kvp, _, err := kv.Keys(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeType, "interfaces/custom", customCName, "inputs")+"/", "/", nil)
	if err != nil {
		log.Panic(err)
	}

	var result []string

	for _, key := range kvp {
		res, _, err := kv.Get(path.Join(key, "is_property_definition"), nil)

		if err != nil {
			return nil, err
		}

		if res == nil {
			continue
		}

		isPropDef, err := strconv.ParseBool(string(res.Value))
		if err != nil {
			return nil, err
		}

		if isPropDef {
			result = append(result, path.Base(key))
		}
	}

	return result, nil
}
