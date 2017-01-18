package rest

import (
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tasks"
)

func (s *Server) scaleHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	nodeName := params.ByName("nodeName")

	kv := s.consulClient.KV()

	if len(nodeName) == 0 {
		log.Panic("You must provide a nodename")
	}

	var instancesDelta int
	var err error
	if value, ok := r.URL.Query()["delta"]; ok {
		if instancesDelta, err = strconv.Atoi(value[0]); err != nil {
			writeError(w, r, newBadRequestError(err))
			return
		} else if instancesDelta == 0 {
			writeError(w, r, newBadRequestError(errors.New("You need to provide a non zero value as 'delta' parameter")))
			return
		}
	} else {
		writeError(w, r, newBadRequestError(errors.New("You need to provide a 'delta' parameter")))
		return
	}

	exists, err := deployments.DoesNodeExist(kv, id, nodeName)
	if err != nil {
		log.Panic(err)
	}
	if !exists {
		writeError(w, r, errNotFound)
		return
	}
	var ok bool
	if ok, err = deployments.HasScalableCapability(kv, id, nodeName); err != nil {
		log.Panic(err)
	} else if !ok {
		writeError(w, r, newBadRequestParameter("node", errors.Errorf("Node %q must be scalable", nodeName)))
		return
	}

	log.Debugf("Scaling %d instances of node %q", instancesDelta, nodeName)
	var taskID string
	if instancesDelta > 0 {
		taskID, err = s.scaleUp(id, nodeName, uint32(instancesDelta))
	} else {
		taskID, err = s.scaleDown(id, nodeName, uint32(-instancesDelta))
	}
	if err != nil {
		if tasks.IsAnotherLivingTaskAlreadyExistsError(err) {
			writeError(w, r, newBadRequestError(err))
			return
		}
		log.Panic(err)
	}
	w.Header().Set("Location", fmt.Sprintf("/deployments/%s/tasks/%s", id, taskID))
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) scaleUp(id, nodeName string, instancesDelta uint32) (string, error) {
	kv := s.consulClient.KV()
	maxInstances, err := deployments.GetMaxNbInstancesForNode(kv, id, nodeName)
	if err != nil {
		log.Panic(err)
	}
	currentNbInstance, err := deployments.GetNbInstancesForNode(kv, id, nodeName)
	if err != nil {
		log.Panic(err)
	}

	if currentNbInstance+instancesDelta > maxInstances {
		log.Debug("The delta is too high, the max instances number is choosen")
		instancesDelta = maxInstances - currentNbInstance
	}

	// NOTE: all those stuff on requirements should probably go into deployments.CreateNewNodeStackInstances
	var req []string

	req, err = deployments.GetRequirementsKeysByNameForNode(kv, id, nodeName, "network")
	if err != nil {
		log.Panic(err)
	}

	storageReq, err := deployments.GetRequirementsKeysByNameForNode(kv, id, nodeName, "local_storage")
	if err != nil {
		log.Panic(err)
	}
	req = append(req, storageReq...)

	var reqNameArr []string
	for _, reqPath := range req {
		var reqName *api.KVPair
		reqName, _, err = kv.Get(path.Join(reqPath, "node"), nil)
		if err != nil {
			log.Panic(err)
		}
		reqNameArr = append(reqNameArr, string(reqName.Value))
		// TODO: for now the link between the requirement instance ID and the node instance ID is a kind of black magic. We should found a way to make it rational...
		_, err = deployments.CreateNewNodeStackInstances(kv, id, string(reqName.Value), int(instancesDelta))
		if err != nil {
			log.Panic(err)
		}
	}

	newInstanceID, err := deployments.CreateNewNodeStackInstances(kv, id, nodeName, int(instancesDelta))
	if err != nil {
		log.Panic(err)
	}

	data := make(map[string]string)

	data["node"] = nodeName
	data["new_instances_ids"] = strings.Join(newInstanceID, ",")
	data["req"] = strings.Join(reqNameArr, ",")

	return s.tasksCollector.RegisterTaskWithData(id, tasks.ScaleUp, data)
}

func (s *Server) scaleDown(id, nodeName string, instancesDelta uint32) (string, error) {
	kv := s.consulClient.KV()

	minInstances, err := deployments.GetMinNbInstancesForNode(kv, id, nodeName)
	if err != nil {
		log.Panic(err)
	}
	currentNbInstance, err := deployments.GetNbInstancesForNode(kv, id, nodeName)
	if err != nil {
		log.Panic(err)
	}

	if currentNbInstance-instancesDelta < minInstances {
		log.Debug("The delta is too low, the min instances number is choosen")
		instancesDelta = currentNbInstance - minInstances
	}

	var req []string

	req, err = deployments.GetRequirementsKeysByNameForNode(kv, id, nodeName, "network")
	if err != nil {
		log.Panic(err)
	}

	storageReq, err := deployments.GetRequirementsKeysByNameForNode(kv, id, nodeName, "local_storage")
	if err != nil {
		log.Panic(err)
	}
	req = append(req, storageReq...)

	var reqNameArr []string
	for _, reqPath := range req {
		var reqName *api.KVPair
		reqName, _, err = kv.Get(path.Join(reqPath, "node"), nil)
		if err != nil {
			log.Panic(err)
		}
		reqNameArr = append(reqNameArr, string(reqName.Value))
	}
	// TODO: we should not make assertions on instance IDs type (should not consider them as int) and should be delegated to the deployments package
	newInstanceID := []string{}
	for i := currentNbInstance - 1; i > currentNbInstance-1-instancesDelta; i-- {
		newInstanceID = append(newInstanceID, strconv.Itoa(int(i)))
	}

	data := make(map[string]string)

	data["node"] = nodeName
	data["new_instances_ids"] = strings.Join(newInstanceID, ",")
	data["req"] = strings.Join(reqNameArr, ",")

	return s.tasksCollector.RegisterTaskWithData(id, tasks.ScaleDown, data)

}
