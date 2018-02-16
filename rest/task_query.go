package rest

import (
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tasks"
)

func (s *Server) taskQueryPreChecks(w http.ResponseWriter, r *http.Request, taskID string) bool {
	kv := s.consulClient.KV()

	tExists, err := tasks.TaskExists(kv, taskID)
	if err != nil {
		log.Panic(err)
	}
	if !tExists {
		writeError(w, r, errNotFound)
		return false
	}
	return true
}

func (s *Server) getTaskQueryHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	taskID := params.ByName("taskId")
	kv := s.consulClient.KV()

	if !s.taskQueryPreChecks(w, r, taskID) {
		return
	}

	task := Task{ID: taskID}
	targetID, err := tasks.GetTaskTarget(kv, taskID)
	if err != nil {
		log.Panic(err)
	}
	task.TargetID = targetID
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

	resultSet, err := tasks.GetTaskResultSet(kv, taskID)
	if err != nil {
		log.Panic(err)
	}
	if resultSet != "" {
		task.ResultSet = []byte(resultSet)
	}
	encodeJSONResponse(w, r, task)
}

func (s *Server) deleteTaskQueryHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	taskID := params.ByName("taskId")
	kv := s.consulClient.KV()
	if !s.taskQueryPreChecks(w, r, taskID) {
		return
	}

	if taskType, err := tasks.GetTaskType(kv, taskID); err != nil {
		log.Panic(err)
	} else if taskType != tasks.Query {
		writeError(w, r, newBadRequestError(errors.Errorf("Cannot delete a non query-typed task (task type is: %q)", taskType.String())))
		return
	}

	if taskStatus, err := tasks.GetTaskStatus(kv, taskID); err != nil {
		log.Panic(err)
	} else if taskStatus != tasks.DONE && taskStatus != tasks.FAILED {
		writeError(w, r, newBadRequestError(errors.Errorf("Cannot delete a task with status %q", taskStatus.String())))
		return
	}

	if err := tasks.DeleteTask(kv, taskID); err != nil {
		log.Panic(err)
	}
	w.WriteHeader(http.StatusAccepted)
}
