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

package tasks

import (
	"fmt"
)

//go:generate go-enum -f=structs.go

// TaskType x ENUM(
// Deploy,
// UnDeploy,
// ScaleOut,
// ScaleIn,
// Purge,
// CustomCommand,
// CustomWorkflow,
// Query
// )
type TaskType int

// TaskStatus x ENUM(
// INITIAL,
// RUNNING,
// DONE,
// FAILED,
// CANCELED
// )
type TaskStatus int

type anotherLivingTaskAlreadyExistsError struct {
	taskID   string
	targetID string
	status   string
}

func (e anotherLivingTaskAlreadyExistsError) Error() string {
	return fmt.Sprintf("Task with id %q and status %q already exists for target %q", e.taskID, e.status, e.targetID)
}

// IsAnotherLivingTaskAlreadyExistsError checks if an error is due to the fact that another task is currently running
// If true, it returns the taskID of the currently running task
func IsAnotherLivingTaskAlreadyExistsError(err error) (bool, string) {
	e, ok := err.(anotherLivingTaskAlreadyExistsError)
	if ok {
		return ok, e.taskID
	}
	return ok, ""
}
