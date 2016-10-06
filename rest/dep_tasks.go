package rest

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/tasks"
)

func (s *Server) getTaskHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	taskId := params.ByName("taskId")
	kv := s.consulClient.KV()

	tExists, err := tasks.TaskExists(kv, taskId)
	if err != nil {
		log.Panic(err)
	}
	if !tExists {
		WriteError(w, r, ErrNotFound)
		return
	}

	// First check that the targetId of the task is the deployment id
	ttid, err := tasks.GetTaskTarget(kv, taskId)
	if err != nil {
		log.Panic(err)
	}
	if ttid != id {
		WriteError(w, r, NewBadRequestError(fmt.Errorf("Task with id %q doesn't correspond to the deployment with id %q", taskId, id)))
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
