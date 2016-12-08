package tasks

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"path"
	"strconv"
	"time"
)

type TaskLock interface {
	Lock(tries int, retryInterval time.Duration) (<-chan struct{}, error)
	Release() error
}

type consulTaskLock struct {
	lock     *api.Lock
	targetId string
}

func (l *consulTaskLock) Lock(tries int, retryInterval time.Duration) (<-chan struct{}, error) {
	for i := 0; i < tries; i++ {
		taskLeadCh, err := l.lock.Lock(nil)
		if err != nil || taskLeadCh == nil {
			log.Debugf("Failed to acquire task lock for target with id %q.", l.targetId)
			time.Sleep(retryInterval)
		} else {
			return taskLeadCh, nil
		}
	}
	return nil, fmt.Errorf("Failed to acquire task lock for target with id %q.", l.targetId)
}

func (l *consulTaskLock) Release() error {
	if err := l.lock.Unlock(); err != nil {
		return nil
	}
	return l.lock.Destroy()
}

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
	p := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, t.Id, "status"), Value: []byte(strconv.Itoa(int(status)))}
	_, err := t.kv.Put(p, nil)
	t.status = status
	return err
}

func GetTasksIdsForTarget(kv *api.KV, targetId string) ([]string, error) {
	tasksKeys, _, err := kv.Keys(consulutil.TasksPrefix+"/", "/", nil)
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
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskId, "status"), nil)
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
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskId, "type"), nil)
	if err != nil {
		return Deploy, err
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return Deploy, fmt.Errorf("Missing status for type with id %q", taskId)
	}
	typeInt, err := strconv.Atoi(string(kvp.Value))
	if err != nil {
		return Deploy, err
	}
	return TaskType(typeInt), nil
}

func GetTaskTarget(kv *api.KV, taskId string) (string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskId, "targetId"), nil)
	if err != nil {
		return "", nil
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return "", fmt.Errorf("Missing targetId for task with id %q", taskId)
	}
	return string(kvp.Value), nil
}

func TaskExists(kv *api.KV, taskId string) (bool, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskId, "targetId"), nil)
	if err != nil {
		return false, nil
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return false, nil
	}
	return true, nil
}

func CancelTask(kv *api.KV, taskId string) error {
	kvp := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, taskId, ".canceledFlag"), Value: []byte("true")}
	_, err := kv.Put(kvp, nil)
	return err
}

func TargetHasLivingTasks(kv *api.KV, targetId string) (bool, string, string, error) {
	tasksKeys, _, err := kv.Keys(consulutil.TasksPrefix+"/", "/", nil)
	if err != nil {
		return false, "", "", err
	}
	for _, taskKey := range tasksKeys {
		kvp, _, err := kv.Get(path.Join(taskKey, "targetId"), nil)
		if err != nil {
			return false, "", "", err
		}
		if kvp != nil && len(kvp.Value) > 0 && string(kvp.Value) == targetId {
			kvp, _, err := kv.Get(path.Join(taskKey, "status"), nil)
			taskId := path.Base(taskKey)
			if err != nil {
				return false, "", "", err
			}
			if kvp == nil || len(kvp.Value) == 0 {
				return false, "", "", fmt.Errorf("Missing status for task with id %q", taskId)
			}
			statusInt, err := strconv.Atoi(string(kvp.Value))
			if err != nil {
				return false, "", "", err
			}
			switch TaskStatus(statusInt) {
			case INITIAL, RUNNING:
				return true, taskId, TaskStatus(statusInt).String(), nil
			}
		}
	}
	return false, "", "", nil
}

func NewTaskLockForTarget(client *api.Client, targetId string) (TaskLock, error) {
	lock, err := client.LockKey(path.Join(consulutil.TasksLocksPrefix, targetId))
	if err != nil {
		return nil, err
	}
	return &consulTaskLock{lock: lock, targetId: targetId}, nil
}
