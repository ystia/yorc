package rest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"novaforge.bull.com/starlings-janus/janus/tasks"
)

func (s *Server) tasksPreChecks(w http.ResponseWriter, r *http.Request, id, taskID string) bool {
	kv := s.consulClient.KV()

	tExists, err := tasks.TaskExists(kv, taskID)
	if err != nil {
		log.Panic(err)
	}
	if !tExists {
		WriteError(w, r, ErrNotFound)
		return false
	}

	// First check that the targetId of the task is the deployment id
	ttid, err := tasks.GetTaskTarget(kv, taskID)
	if err != nil {
		log.Panic(err)
	}
	if ttid != id {
		WriteError(w, r, NewBadRequestError(fmt.Errorf("Task with id %q doesn't correspond to the deployment with id %q", taskID, id)))
		return false
	}
	return true
}

func (s *Server) cancelTaskHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	taskID := params.ByName("taskId")
	kv := s.consulClient.KV()
	if !s.tasksPreChecks(w, r, id, taskID) {
		return
	}

	if taskStatus, err := tasks.GetTaskStatus(kv, taskID); err != nil {
		log.Panic(err)
	} else if taskStatus != tasks.RUNNING && taskStatus != tasks.INITIAL {
		WriteError(w, r, NewBadRequestError(fmt.Errorf("Cannot cancel a task with status %q", taskStatus.String())))
		return
	}

	if err := tasks.CancelTask(kv, taskID); err != nil {
		log.Panic(err)
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) getTaskHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	taskID := params.ByName("taskId")
	kv := s.consulClient.KV()

	if !s.tasksPreChecks(w, r, id, taskID) {
		return
	}

	task := Task{ID: taskID, TargetID: id}
	status, err := tasks.GetTaskStatus(kv, taskID)
	if err != nil {
		log.Panic(err)
	}
	task.Status = status.String()

	taskType, err := tasks.GetTaskType(kv, taskID)
	if err != nil {
		log.Panic(err)
	}
	task.Type = taskType.String()
	encodeJSONResponse(w, r, task)
}

func (s *Server) newTaskHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")

	var tr TaskRequest
	if body, err := ioutil.ReadAll(r.Body); err != nil {
		log.Panic(err)
	} else {
		if err = json.Unmarshal(body, &tr); err != nil {
			WriteError(w, r, NewBadRequestError(fmt.Errorf("Can't unmarshal task request %v", err)))
			return
		}
	}

	taskType, err := tasks.TaskTypeForName(tr.Type)
	if err != nil {
		log.Panic(err)
	}

	taskID, err := s.tasksCollector.RegisterTask(id, taskType)
	if err != nil {
		if tasks.IsAnotherLivingTaskAlreadyExistsError(err) {
			WriteError(w, r, NewBadRequestError(err))
			return
		}
		log.Panic(err)
	}

	w.Header().Set("Location", fmt.Sprintf("/deployments/%s/tasks/%s", id, taskID))
	w.WriteHeader(http.StatusCreated)
}
