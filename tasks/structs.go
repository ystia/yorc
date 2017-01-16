package tasks

import (
	"fmt"
	"strings"
)

//go:generate stringer -type=TaskStatus,TaskType -output=structs_string.go structs.go

type TaskType int

const (
	Deploy TaskType = iota
	UnDeploy
	ScaleUp
	ScaleDown
	Purge
	CustomCommand
)

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

type TaskStatus int

const (
	INITIAL TaskStatus = iota
	RUNNING
	DONE
	FAILED
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

func IsAnotherLivingTaskAlreadyExistsError(err error) bool {
	_, ok := err.(anotherLivingTaskAlreadyExistsError)
	return ok
}
