package tasks

//go:generate stringer -type=TaskStatus,TaskType -output=structs_string.go structs.go

const tasksPrefix = "_janus/tasks"

type TaskType int

const (
	DEPLOY TaskType = iota
	UNDEPLOY
)

type TaskStatus int

const (
	INITIAL TaskStatus = iota
	DONE
	FAILED
)
