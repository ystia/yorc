package tasks

import (
	"fmt"
	"strings"
)

//go:generate stringer -type=TaskStatus,TaskType -output=structs_string.go structs.go

// A TaskType determines the type of a Task
type TaskType int

const (
	// Deploy defines a Task of type "deploy"
	Deploy TaskType = iota
	// UnDeploy defines a Task of type "undeploy"
	UnDeploy
	// ScaleUp defines a Task of type "scale-up"
	ScaleUp
	// ScaleDown defines a Task of type "scale-down"
	ScaleDown
	// Purge defines a Task of type "purge"
	Purge
	// CustomCommand defines a Task of type "custom-command"
	CustomCommand
)

// TaskTypeForName converts a textual representation of a task into a TaskType
func TaskTypeForName(taskType string) (TaskType, error) {
	switch strings.ToLower(taskType) {
	case "deploy":
		return Deploy, nil
	case "undeploy":
		return UnDeploy, nil
	case "purge":
		return Purge, nil
	case "custom":
		return CustomCommand, nil
	case "scale-up":
		return ScaleUp, nil
	case "scale-down":
		return ScaleDown, nil
	default:
		return Deploy, fmt.Errorf("Unsupported task type %q", taskType)
	}
}

// TaskStatus represents the status of a Task
type TaskStatus int

const (
	// INITIAL is the initial status of a that haven't run yet
	INITIAL TaskStatus = iota
	// RUNNING is the status of a task that is currently processed
	RUNNING
	// DONE is the status of a task successful task
	DONE
	// FAILED is the status of a failed task
	FAILED
	// CANCELED is the status of a canceled task
	CANCELED
)

type anotherLivingTaskAlreadyExistsError struct {
	taskID   string
	targetID string
	status   string
}

func (e anotherLivingTaskAlreadyExistsError) Error() string {
	return fmt.Sprintf("Task with id %q and status %q already exists for target %q", e.taskID, e.status, e.targetID)
}

// IsAnotherLivingTaskAlreadyExistsError checks if an error is due to the fact that another task is currently running
func IsAnotherLivingTaskAlreadyExistsError(err error) bool {
	_, ok := err.(anotherLivingTaskAlreadyExistsError)
	return ok
}
