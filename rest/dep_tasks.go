package rest

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/tasks"
)

func (s *Server) tasksPreChecks(w http.ResponseWriter, r *http.Request, id, taskId string) bool {
	kv := s.consulClient.KV()

	tExists, err := tasks.TaskExists(kv, taskId)
	if err != nil {
		log.Panic(err)
	}
	if !tExists {
		WriteError(w, r, ErrNotFound)
		return false
	}

	// First check that the targetId of the task is the deployment id
	ttid, err := tasks.GetTaskTarget(kv, taskId)
	if err != nil {
		log.Panic(err)
	}
	if ttid != id {
		WriteError(w, r, NewBadRequestError(fmt.Errorf("Task with id %q doesn't correspond to the deployment with id %q", taskId, id)))
		return false
	}
	return true
}

func (s *Server) cancelTaskHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	taskId := params.ByName("taskId")
	kv := s.consulClient.KV()
	if !s.tasksPreChecks(w, r, id, taskId) {
		return
	}

	if taskStatus, err := tasks.GetTaskStatus(kv, taskId); err != nil {
		log.Panic(err)
	} else if taskStatus != tasks.RUNNING && taskStatus != tasks.INITIAL {
		WriteError(w, r, NewBadRequestError(fmt.Errorf("Cannot cancel a task with status %q", taskStatus.String())))
		return
	}

	if err := tasks.CancelTask(kv, taskId); err != nil {
		log.Panic(err)
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) getTaskHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	taskId := params.ByName("taskId")
	kv := s.consulClient.KV()

	if !s.tasksPreChecks(w, r, id, taskId) {
		return
	}

	task := Task{Id: taskId, TargetId: id}
	status, err := tasks.GetTaskStatus(kv, taskId)
	if err != nil {
		log.Panic(err)
	}
	task.Status = status.String()

	taskType, err := tasks.GetTaskType(kv, taskId)
	if err != nil {
		log.Panic(err)
	}
	task.Type = taskType.String()
	encodeJsonResponse(w, r, task)
}
