// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rest

import (
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tasks"
	"net/http"
	"path"
	"strings"
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
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
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

func (s *Server) listTaskQueryHandler(w http.ResponseWriter, r *http.Request) {
	var target, query string
	query = strings.TrimPrefix(strings.TrimSuffix(r.URL.String(), "?"+r.URL.RawQuery), "/")

	queryValues := r.URL.Query()
	if queryValues != nil {
		target = queryValues.Get("target")
	}
	log.Debugf("Retrieving query tasks with query:%q and target:%q", query, target)
	kv := s.consulClient.KV()
	ids, err := tasks.GetQueryTaskIDs(kv, tasks.Query, query, target)
	if err != nil {
		log.Panic(err)
	}

	tasksCol := TasksCollection{Tasks: make([]AtomLink, len(ids))}
	for ind, taskID := range ids {
		targetID, err := tasks.GetTaskTarget(kv, taskID)
		if err != nil {
			log.Panic(err)
		}

		split := strings.Split(targetID, ":")
		if len(split) != 2 {
			log.Printf("Query Task (id: %q): unexpected format for targetID: %q. This task will be ignored", taskID, targetID)
			continue
		}
		target := split[1]
		link := newAtomLink(LinkRelTask, path.Join("/", query, target, "tasks", taskID))
		tasksCol.Tasks[ind] = link
	}
	encodeJSONResponse(w, r, tasksCol)
}
