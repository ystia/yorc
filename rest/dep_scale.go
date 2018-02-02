package rest

import (
	"fmt"
	"net/http"
	"path"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tasks"
)

func (s *Server) scaleHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")
	nodeName := params.ByName("nodeName")

	kv := s.consulClient.KV()

	dExits, err := deployments.DoesDeploymentExists(s.consulClient.KV(), id)
	if err != nil {
		log.Panicf("%v", err)
	}
	if !dExits {
		writeError(w, r, errNotFound)
		return
	}

	if len(nodeName) == 0 {
		log.Panic("You must provide a nodename")
	}

	var instancesDelta int
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
		if restError, ok := err.(*Error); ok {
			writeError(w, r, restError)
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
		return "", err
	}
	currentNbInstance, err := deployments.GetNbInstancesForNode(kv, id, nodeName)
	if err != nil {
		return "", err
	}

	if currentNbInstance+instancesDelta > maxInstances {
		log.Debug("The delta is too high, the max instances number is chosen")
		instancesDelta = maxInstances - currentNbInstance
		if instancesDelta == 0 {
			return "", newBadRequestMessage("Maximum number of instances reached")
		}
	}

	instancesByNodes, err := deployments.CreateNewNodeStackInstances(kv, id, nodeName, int(instancesDelta))
	if err != nil {
		return "", err
	}

	data := make(map[string]string)
	for scalableNode, nodeInstances := range instancesByNodes {
		data[path.Join("nodes", scalableNode)] = nodeInstances
	}
	return s.tasksCollector.RegisterTaskWithData(id, tasks.ScaleUp, data)
}

func (s *Server) scaleDown(id, nodeName string, instancesDelta uint32) (string, error) {
	kv := s.consulClient.KV()

	minInstances, err := deployments.GetMinNbInstancesForNode(kv, id, nodeName)
	if err != nil {
		return "", err
	}
	currentNbInstance, err := deployments.GetNbInstancesForNode(kv, id, nodeName)
	if err != nil {
		return "", err
	}

	if currentNbInstance-instancesDelta < minInstances {
		log.Debug("The delta is too low, the min instances number is chosen")
		instancesDelta = currentNbInstance - minInstances
		if instancesDelta == 0 {
			return "", newBadRequestMessage("Minimum number of instances reached")
		}
	}

	instancesByNodes, err := deployments.SelectNodeStackInstances(kv, id, nodeName, int(instancesDelta))
	if err != nil {
		return "", err
	}

	data := make(map[string]string)
	for scalableNode, nodeInstances := range instancesByNodes {
		data[path.Join("nodes", scalableNode)] = nodeInstances
	}

	return s.tasksCollector.RegisterTaskWithData(id, tasks.ScaleDown, data)

}
