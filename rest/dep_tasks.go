package rest

import (
	"fmt"
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
		writeError(w, r, errNotFound)
		return false
	}

	// First check that the targetId of the task is the deployment id
	ttid, err := tasks.GetTaskTarget(kv, taskID)
	if err != nil {
		log.Panic(err)
	}
	if ttid != id {
		writeError(w, r, newBadRequestError(fmt.Errorf("Task with id %q doesn't correspond to the deployment with id %q", taskID, id)))
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
		writeError(w, r, newBadRequestError(fmt.Errorf("Cannot cancel a task with status %q", taskStatus.String())))
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
