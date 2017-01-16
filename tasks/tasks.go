package tasks

import (
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

type TaskLock interface {
	Lock(tries int, retryInterval time.Duration) (<-chan struct{}, error)
	Release() error
}

type consulTaskLock struct {
	lock     *api.Lock
	targetID string
}

func (l *consulTaskLock) Lock(tries int, retryInterval time.Duration) (<-chan struct{}, error) {
	for i := 0; i < tries; i++ {
		taskLeadCh, err := l.lock.Lock(nil)
		if err != nil || taskLeadCh == nil {
			log.Debugf("Failed to acquire task lock for target with id %q.", l.targetID)
			time.Sleep(retryInterval)
		} else {
			return taskLeadCh, nil
		}
	}
	return nil, fmt.Errorf("Failed to acquire task lock for target with id %q.", l.targetID)
}

func (l *consulTaskLock) Release() error {
	if err := l.lock.Unlock(); err != nil {
		return nil
	}
	return l.lock.Destroy()
}

type Task struct {
	ID       string
	TargetID string
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
	p := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, t.ID, "status"), Value: []byte(strconv.Itoa(int(status)))}
	_, err := t.kv.Put(p, nil)
	t.status = status
	return err
}

func GetTasksIdsForTarget(kv *api.KV, targetID string) ([]string, error) {
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
		if kvp != nil && len(kvp.Value) > 0 && string(kvp.Value) == targetID {
			tasks = append(tasks, path.Base(taskKey))
		}
	}
	return tasks, nil
}

func GetTaskStatus(kv *api.KV, taskID string) (TaskStatus, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "status"), nil)
	if err != nil {
		return FAILED, err
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return FAILED, fmt.Errorf("Missing status for task with id %q", taskID)
	}
	statusInt, err := strconv.Atoi(string(kvp.Value))
	if err != nil {
		return FAILED, err
	}
	return TaskStatus(statusInt), nil
}

func GetTaskType(kv *api.KV, taskID string) (TaskType, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "type"), nil)
	if err != nil {
		return Deploy, err
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return Deploy, fmt.Errorf("Missing status for type with id %q", taskID)
	}
	typeInt, err := strconv.Atoi(string(kvp.Value))
	if err != nil {
		return Deploy, err
	}
	return TaskType(typeInt), nil
}

func GetTaskTarget(kv *api.KV, taskID string) (string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "targetId"), nil)
	if err != nil {
		return "", nil
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return "", fmt.Errorf("Missing targetId for task with id %q", taskID)
	}
	return string(kvp.Value), nil
}

func TaskExists(kv *api.KV, taskID string) (bool, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "targetId"), nil)
	if err != nil {
		return false, nil
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return false, nil
	}
	return true, nil
}

func CancelTask(kv *api.KV, taskID string) error {
	kvp := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, taskID, ".canceledFlag"), Value: []byte("true")}
	_, err := kv.Put(kvp, nil)
	return err
}

func TargetHasLivingTasks(kv *api.KV, targetID string) (bool, string, string, error) {
	tasksKeys, _, err := kv.Keys(consulutil.TasksPrefix+"/", "/", nil)
	if err != nil {
		return false, "", "", err
	}
	for _, taskKey := range tasksKeys {
		kvp, _, err := kv.Get(path.Join(taskKey, "targetId"), nil)
		if err != nil {
			return false, "", "", err
		}
		if kvp != nil && len(kvp.Value) > 0 && string(kvp.Value) == targetID {
			kvp, _, err := kv.Get(path.Join(taskKey, "status"), nil)
			taskID := path.Base(taskKey)
			if err != nil {
				return false, "", "", err
			}
			if kvp == nil || len(kvp.Value) == 0 {
				return false, "", "", fmt.Errorf("Missing status for task with id %q", taskID)
			}
			statusInt, err := strconv.Atoi(string(kvp.Value))
			if err != nil {
				return false, "", "", err
			}
			switch TaskStatus(statusInt) {
			case INITIAL, RUNNING:
				return true, taskID, TaskStatus(statusInt).String(), nil
			}
		}
	}
	return false, "", "", nil
}

func NewTaskLockForTarget(client *api.Client, targetID string) (TaskLock, error) {
	lock, err := client.LockKey(path.Join(consulutil.TasksLocksPrefix, targetID))
	if err != nil {
		return nil, err
	}
	return &consulTaskLock{lock: lock, targetID: targetID}, nil
}
