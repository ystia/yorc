package tasks

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"path"
	"strconv"
)

type Task struct {
	Id       string
	TargetId string
	status   TaskStatus
	TaskType TaskType
	taskLock *api.Lock
	kv       *api.KV
}

func (t *Task) releaseLock() {
	t.taskLock.Unlock()
	t.taskLock.Destroy()
}

func (t *Task) Status() TaskStatus {
	return t.status
}

func (t *Task) WithStatus(status TaskStatus) error {
	p := &api.KVPair{Key: path.Join(tasksPrefix, t.Id, "status"), Value: []byte(strconv.Itoa(int(status)))}
	_, err := t.kv.Put(p, nil)
	t.status = status
	return err
}

func GetTasksIdsForTarget(kv *api.KV, targetId string) ([]string, error) {
	tasksKeys, _, err := kv.Keys(tasksPrefix+"/", "/", nil)
	if err != nil {
		return nil, err
	}
	tasks := make([]string, 0)
	for _, taskKey := range tasksKeys {
		kvp, _, err := kv.Get(path.Join(taskKey, "targetId"), nil)
		if err != nil {
			return nil, err
		}
		if kvp != nil && len(kvp.Value) > 0 && string(kvp.Value) == targetId {
			tasks = append(tasks, path.Base(taskKey))
		}
	}
	return tasks, nil
}

func GetTaskStatus(kv *api.KV, taskId string) (TaskStatus, error) {
	kvp, _, err := kv.Get(path.Join(tasksPrefix, taskId, "status"), nil)
	if err != nil {
		return FAILED, err
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return FAILED, fmt.Errorf("Missing status for task with id %q", taskId)
	}
	statusInt, err := strconv.Atoi(string(kvp.Value))
	if err != nil {
		return FAILED, err
	}
	return TaskStatus(statusInt), nil
}

func GetTaskType(kv *api.KV, taskId string) (TaskType, error) {
	kvp, _, err := kv.Get(path.Join(tasksPrefix, taskId, "type"), nil)
	if err != nil {
		return DEPLOY, err
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return DEPLOY, fmt.Errorf("Missing status for type with id %q", taskId)
	}
	typeInt, err := strconv.Atoi(string(kvp.Value))
	if err != nil {
		return DEPLOY, err
	}
	return TaskType(typeInt), nil
}

func GetTaskTarget(kv *api.KV, taskId string) (string, error) {
	kvp, _, err := kv.Get(path.Join(tasksPrefix, taskId, "targetId"), nil)
	if err != nil {
		return "", nil
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return "", fmt.Errorf("Missing targetId for task with id %q", taskId)
	}
	return string(kvp.Value), nil
}

func TaskExists(kv *api.KV, taskId string) (bool, error) {
	kvp, _, err := kv.Get(path.Join(tasksPrefix, taskId, "targetId"), nil)
	if err != nil {
		return false, nil
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return false, nil
	}
	return true, nil
}

func CancelTask(kv *api.KV, taskId string) error {
	kvp := &api.KVPair{Key: path.Join(tasksPrefix, taskId, ".canceledFlag"), Value: []byte("true")}
	_, err := kv.Put(kvp, nil)
	return err
}
