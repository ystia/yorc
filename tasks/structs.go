package tasks

import (
	"fmt"
	"strings"
)

//go:generate stringer -type=TaskStatus,TaskType -output=structs_string.go structs.go

const tasksPrefix = "_janus/tasks"

type TaskType int

const (
	DEPLOY TaskType = iota
	UNDEPLOY
)

func TaskTypeForName(taskType string) (TaskType, error) {
	switch strings.ToLower(taskType) {
	case "deploy":
		return DEPLOY, nil
	case "undeploy":
		return UNDEPLOY, nil
	default:
		return DEPLOY, fmt.Errorf("Unsupported task type %q", taskType)
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
