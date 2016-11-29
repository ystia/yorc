package rest

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/antonholmquist/jason"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"path"
	"strconv"
)

func (s *Server) newCustomCommandHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	ctx, errGrp, consulStore := consulutil.WithContext(ctx)

	v, err := jason.NewObjectFromReader(r.Body)

	if err != nil {
		log.Panic(err)
	}

	node, err := v.GetString("node")
	customCommandName, err := v.GetString("name")
	if err != nil {
		log.Panic(err)
	}

	inputsName, err := s.getInputNameFromCustom(id, node, customCommandName)
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

	consulStore.StoreConsulKey(path.Join(tasks.GetTaskPrefix(), taskId, "node"), []byte(node))
	consulStore.StoreConsulKey(path.Join(tasks.GetTaskPrefix(), taskId, "name"), []byte(customCommandName))

	inputsArg, err := v.GetValueArray("inputs")
	if err != nil {
		log.Panic(err)
	}

	for i, name := range inputsName {
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		err := enc.Encode((inputsArg[i]).Interface())

		if err != nil {
			log.Panic(err)
		}

		consulStore.StoreConsulKey(path.Join(tasks.GetTaskPrefix(), taskId, "inputs", name), buf.Bytes())
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
	kvp, _, err := kv.Keys(path.Join(deployments.DeploymentKVPrefix, depId, "topology/types", nodeType, "interfaces/custom", customCName, "inputs")+"/", "/", nil)
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
