package tasks

import (
	"github.com/hashicorp/consul/api"
	"strings"
)

const tasksPrefix = "_janus/tasks"

type TaskType int

const (
	DEPLOY TaskType = iota
	UNDEPLOY
)

type Task struct {
	Id       string
	TargetId string
	status   string
	TaskType TaskType
	TaskLock *api.Lock
	kv       *api.KV
}

func NewTask(id, targetId, status string, taskLock *api.Lock, kv *api.KV, taskType TaskType) *Task {
	return &Task{Id: id, status: status, TargetId: targetId, TaskLock: taskLock, kv: kv, TaskType: taskType}
}

func (t *Task) Status() string {
	return t.status
}

func (t *Task) WithStatus(status string) error {
	p := &api.KVPair{Key: strings.Join([]string{tasksPrefix, t.Id, "status"}, "/"), Value: []byte(status)}
	_, err := t.kv.Put(p, nil)
	t.status = status
	return err
}
