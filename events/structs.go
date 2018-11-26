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

package events

//go:generate go-enum -f=structs.go --lower

// LogLevel x ENUM(
// Instance,
// Deployment,
// CustomCommand,
// Scaling,
// Workflow,
// WorkflowStep
// )
type StatusChangeType int

// EventOptionalInfo is event's additional info
type EventOptionalInfo map[string]interface{}

// EventStatusChange represents status change event
type EventStatusChange struct {
	Timestamp      string `json:"timestamp"`
	Type           string `json:"type"`
	Node           string `json:"node,omitempty"`
	Instance       string `json:"instance,omitempty"`
	TaskID         string `json:"task_id,omitempty"`
	DeploymentID   string `json:"deployment_id"`
	Status         string `json:"status"`
	AdditionalInfo EventOptionalInfo
}
