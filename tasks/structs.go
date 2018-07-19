package tasks

import "fmt"

//go:generate go-enum -f=structs.go

// TaskType x ENUM(
// Deploy,
// UnDeploy,
// ScaleOut,
// ScaleIn,
// Purge,
// CustomCommand,
// CustomWorkflow,
// Query,
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
