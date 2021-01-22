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

//go:generate go-enum -f=structs.go

// TaskType is an enumerated type for tasks
/*
ENUM(
Deploy
UnDeploy
ScaleOut
ScaleIn
Purge
CustomCommand
CustomWorkflow
Query
Action
ForcePurge // ForcePurge is deprecated and should not be used anymore this stay here to prevent task renumbering colision
AddNodes
RemoveNodes
)
*/
type TaskType int

// TaskStatus is an enumerated type for tasks statuses
/*
ENUM(
INITIAL
RUNNING
DONE
FAILED
CANCELED
)
*/
type TaskStatus int

// IsDeploymentRelatedTask returns true if the task is related to a deployment
//
// Typically query and action tasks are not necessary related to a deployment.
func IsDeploymentRelatedTask(tt TaskType) bool {
	return !(tt == TaskTypeQuery || tt == TaskTypeAction)
}
