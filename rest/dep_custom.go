package rest

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"path"
	"strconv"
	"encoding/json"
	"io/ioutil"
)

func (s *Server) newCustomCommandHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	ctx, errGrp, consulStore := consulutil.WithContext(ctx)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Panic(err)
	}

	var inputMap deployments.InputsPropertyDef
	if err := json.Unmarshal(body, &inputMap);err != nil {
		log.Panic(err)
	}

	inputsName, err := s.getInputNameFromCustom(id, inputMap.NodeName, inputMap.CustomCommandName)
	if err != nil {
		log.Panic(err)
	}

	destroy, lock, taskId, err := s.tasksCollector.RegisterTaskWithoutDestroyLock(id, tasks.CustomCommand)
	if err != nil {
		if tasks.IsAnotherLivingTaskAlreadyExistsError(err) {
			WriteError(w, r, NewBadRequestError(err))
			return
		}
		log.Panic(err)
	}

	consulStore.StoreConsulKey(path.Join(consulutil.TasksPrefix, taskId, "node"), []byte(inputMap.NodeName))
	consulStore.StoreConsulKey(path.Join(consulutil.TasksPrefix, taskId, "name"), []byte(inputMap.CustomCommandName))

	for i, name := range inputsName {

		if err != nil {
			log.Panic(err)
		}

		consulStore.StoreConsulKey(path.Join(consulutil.TasksPrefix, taskId, "inputs", name), []byte((inputMap.Inputs[name])))
	}

	if errGrp.Wait() != nil {
		log.Panic(err)
	}

	destroy(lock, taskId, id)

	w.Header().Set("Location", fmt.Sprintf("/deployments/%s/tasks/%s", id, taskId))
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) getInputNameFromCustom(depId, nodeName, customCName string) ([]string, error) {
	nodeType, err := deployments.GetNodeType(s.consulClient.KV(), depId, nodeName)

	if err != nil {
		return nil, err
	}

	kv := s.consulClient.KV()
	kvp, _, err := kv.Keys(path.Join(consulutil.DeploymentKVPrefix, depId, "topology/types", nodeType, "interfaces/custom", customCName, "inputs")+"/", "/", nil)
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
