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

package prov

import (
	"context"
	"fmt"

	"github.com/ystia/yorc/config"
)

// DelegateExecutor is the interface that wraps the ExecDelegate method
//
// ExecDelegate executes the given delegateOperation for given nodeName on the given deploymentID.
// The taskID identifies the task that requested to execute this delegate operation.
// The given ctx may be used to check for cancellation, conf is the server Configuration.
type DelegateExecutor interface {
	ExecDelegate(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error
}

// Operation represent a provisioning operation
type Operation struct {
	// The operation name
	Name string
	// Name of the type implementing this operation
	ImplementedInType string
	// Artifact type of the operation implementation
	ImplementationArtifact string
	// Additional information for relationship operation
	RelOp RelationshipOperation
	// Node on which operation should be executed
	OperationHost string
}

// String implements the fmt.Stringer interface
func (o Operation) String() string {
	s := fmt.Sprintf("{ Name: %q, Implemented in type: %q, Implementation Artifact: %q", o.Name, o.ImplementedInType, o.ImplementationArtifact)
	if o.RelOp.IsRelationshipOperation {
		s += ", " + o.RelOp.String()
	}
	s += " }"
	return s
}

// RelationshipOperation provides additional information for relationship operation
type RelationshipOperation struct {
	// If this is set to true then other struct fields could be considered.
	IsRelationshipOperation bool
	// Requirement index of the relationship in the source node
	RequirementIndex string
	// Name of the target node of the relationship
	TargetNodeName string
	// Requirement name in case of relationship
	TargetRelationship string
}

// String implements the fmt.Stringer interface
func (ro RelationshipOperation) String() string {
	return fmt.Sprintf("Relationship target node node: %q (Requirement index %q)", ro.TargetNodeName, ro.RequirementIndex)
}

// OperationExecutor is the interface that wraps the ExecOperation method
//
// ExecOperation executes the given TOSCA operation for given nodeName on the given deploymentID.
// The taskID identifies the task that requested to execute this operation.
// The given ctx may be used to check for cancellation, conf is the server Configuration.
type OperationExecutor interface {
	ExecOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation Operation) error
}

// InfraUsageCollector is the interface for collecting information about infrastructure usage
//
// GetUsageInfo returns data about infrastructure usage for defined infrastructure
type InfraUsageCollector interface {
	GetUsageInfo(ctx context.Context, cfg config.Configuration, taskID, infraName string) (map[string]interface{}, error)
}
