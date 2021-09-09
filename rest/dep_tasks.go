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
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/tasks"
)

func (s *Server) tasksPreChecks(w http.ResponseWriter, r *http.Request, id, taskID string) bool {
	tExists, err := tasks.TaskExists(taskID)
	if err != nil {
		log.Panic(err)
	}
	if !tExists {
		writeError(w, r, errNotFound)
		return false
	}

	// First check that the targetId of the task is the deployment id
	ttid, err := tasks.GetTaskTarget(taskID)
	if err != nil {
		log.Panic(err)
	}
	if ttid != id {
		writeError(w, r, newBadRequestError(errors.Errorf("Task with id %q doesn't correspond to the deployment with id %q", taskID, id)))
		return false
	}
	return true
}

func (s *Server) cancelTaskHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")
	taskID := params.ByName("taskId")
	if !s.tasksPreChecks(w, r, id, taskID) {
		return
	}

	if taskStatus, err := tasks.GetTaskStatus(taskID); err != nil {
		log.Panic(err)
	} else if taskStatus != tasks.TaskStatusRUNNING && taskStatus != tasks.TaskStatusINITIAL {
		writeError(w, r, newBadRequestError(errors.Errorf("Cannot cancel a task with status %q", taskStatus.String())))
		return
	}

	if err := tasks.CancelTask(taskID); err != nil {
		log.Panic(err)
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) getTaskHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")
	taskID := params.ByName("taskId")

	if !s.tasksPreChecks(w, r, id, taskID) {
		return
	}

	task := Task{ID: taskID, TargetID: id}
	status, err := tasks.GetTaskStatus(taskID)
	if err != nil {
		log.Panic(err)
	}
	task.Status = status.String()

	taskType, err := tasks.GetTaskType(taskID)
	if err != nil {
		log.Panic(err)
	}
	task.Type = taskType.String()

	resultSet, err := tasks.GetTaskResultSet(taskID)
	if err != nil {
		log.Panic(err)
	}
	if resultSet != "" {
		task.ResultSet = []byte(resultSet)
	}

	task.Outputs, err = tasks.GetTaskOutputs(taskID)
	if err != nil {
		log.Panic(err)
	}

	taskErrorMessage, err := tasks.GetTaskErrorMessage(taskID)
	if err != nil {
		log.Panic(err)
	}
	task.ErrorMessage = taskErrorMessage
	encodeJSONResponse(w, r, task)
}

func (s *Server) getTaskStepsHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	deploymentID := params.ByName("id")
	taskID := params.ByName("taskId")

	if !s.tasksPreChecks(w, r, deploymentID, taskID) {
		return
	}

	steps, err := tasks.GetTaskRelatedSteps(taskID)
	if err != nil {
		log.Panic(err)
	}
	encodeJSONResponse(w, r, steps)
}

func (s *Server) updateTaskStepStatusHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	deploymentID := params.ByName("id")
	taskID := params.ByName("taskId")
	stepID := params.ByName("stepId")
	// Check Task/Deployment
	if !s.tasksPreChecks(w, r, deploymentID, taskID) {
		return
	}

	if !checkBlockingOperationOnDeployment(ctx, deploymentID, w, r) {
		return
	}

	// Check TaskStep/Task existence
	stExists, stepBefore, err := tasks.TaskStepExists(taskID, stepID)
	if err != nil {
		log.Panic(err)
	}
	if !stExists {
		writeError(w, r, errNotFound)
		log.Panic("Unknown step related to this task")
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Panic(err)
	}

	step := &tasks.TaskStep{}
	err = json.Unmarshal(body, step)
	if err != nil {
		log.Panic(err)
	}
	step.Name = stepID

	// Check taskStep status change
	allowed, err := tasks.CheckTaskStepStatusChange(stepBefore.Status, step.Status)
	if err != nil {
		log.Panic(err)
	}
	if !allowed {
		writeError(w, r, errForbidden)
		log.Panicf("The task step status update from %s to %s is forbidden", stepBefore.Status, step.Status)
	}

	err = tasks.UpdateTaskStepStatus(taskID, step)
	if err != nil {
		log.Panic(err)
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) resumeTaskHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")
	taskID := params.ByName("taskId")
	if !s.tasksPreChecks(w, r, id, taskID) {
		return
	}

	if !checkBlockingOperationOnDeployment(ctx, id, w, r) {
		return
	}
	if taskStatus, err := tasks.GetTaskStatus(taskID); err != nil {
		log.Panic(err)
	} else if taskStatus != tasks.TaskStatusFAILED {
		writeError(w, r, newBadRequestError(errors.Errorf("Cannot resume a task with status %q. Only task in %q status can be resumed.", taskStatus.String(), tasks.TaskStatusFAILED.String())))
		return
	}

	if err := s.tasksCollector.ResumeTask(ctx, taskID); err != nil {
		log.Panic(err)
	}
	w.WriteHeader(http.StatusAccepted)
}
