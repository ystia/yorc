package prov

import (
	"context"

	"novaforge.bull.com/starlings-janus/janus/config"
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
	// Artifact type of the operation implementation
	ImplementationArtifact string
	// Additional information for relationship operation
	RelOp RelationshipOperation
}

// RelationshipOperation provides additional information for relationship operation
type RelationshipOperation struct {
	// If this is set to true then other struct fields could be considered.
	IsRelationshipOperation bool
	// Requirement index of the relationship in the source node
	RequirementIndex string
	// Name of the target node of the relationship
	TargetNodeName string
}

// OperationExecutor is the interface that wraps the ExecOperation method
//
// ExecOperation executes the given TOSCA operation for given nodeName on the given deploymentID.
// The taskID identifies the task that requested to execute this operation.
// The given ctx may be used to check for cancellation, conf is the server Configuration.
type OperationExecutor interface {
	ExecOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation Operation) error
}
