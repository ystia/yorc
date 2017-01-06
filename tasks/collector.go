package tasks

import (
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/satori/go.uuid"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

type Collector struct {
	consulClient *api.Client
}

func NewCollector(consulClient *api.Client) *Collector {
	return &Collector{consulClient: consulClient}
}

func (c *Collector) RegisterTaskWithData(targetId string, taskType TaskType, data map[string]string) (string, error) {
	destroy, lock, taskId, err := c.RegisterTaskWithoutDestroyLock(targetId, taskType, data)
	defer destroy(lock, taskId, targetId)
	if err != nil {
		return "", err
	}
	return taskId, nil
}

func (c *Collector) RegisterTask(targetId string, taskType TaskType) (string, error) {
	destroy, lock, taskId, err := c.RegisterTaskWithoutDestroyLock(targetId, taskType, nil)
	defer destroy(lock, taskId, targetId)
	if err != nil {
		return "", err
	}
	return taskId, nil
}

func (c *Collector) RegisterTaskWithoutDestroyLock(targetId string, taskType TaskType, data map[string]string) (func(taskLockCreate *api.Lock, taskId, targetId string), *api.Lock, string, error) { // First check if other tasks are running for this target before creating a new one
	taskLock, err := NewTaskLockForTarget(c.consulClient, targetId)
	_, err = taskLock.Lock(10, 500*time.Millisecond)
	if err != nil {
		return nil, nil, "", err
	}
	defer taskLock.Release()
	if hasLivingTask, livingTaskId, livingTaskStatus, err := TargetHasLivingTasks(c.consulClient.KV(), targetId); err != nil {
		return nil, nil, "", err
	} else if hasLivingTask {
		return nil, nil, "", anotherLivingTaskAlreadyExistsError{taskId: livingTaskId, targetId: targetId, status: livingTaskStatus}
	}
	taskId := fmt.Sprint(uuid.NewV4())
	kv := c.consulClient.KV()
	taskPrefix := consulutil.TasksPrefix + "/" + taskId
	// Then use a lock in the task to prevent dispatcher to get the task before finishing task creation
	taskLockCreate, err := c.consulClient.LockKey(taskPrefix + "/.createLock")
	if err != nil {
		return nil, nil, taskId, err
	}
	stopLockChan := make(chan struct{})
	defer close(stopLockChan)
	leaderCh, err := taskLockCreate.Lock(stopLockChan)
	if err != nil {
		log.Printf("Failed to acquire create lock for task with id %q (target id %q): %+v", taskId, targetId, err)
		return nil, nil, taskId, err
	}
	if leaderCh == nil {
		log.Printf("Failed to acquire create lock for task with id %q (target id %q).", taskId, targetId)
		return nil, nil, taskId, fmt.Errorf("Failed to acquire lock for task with id %q (target id %q)", taskId, targetId)
	}

	key := &api.KVPair{Key: taskPrefix + "/targetId", Value: []byte(targetId)}
	if _, err := kv.Put(key, nil); err != nil {
		log.Print(err)
		return nil, nil, taskId, err
	}
	key = &api.KVPair{Key: taskPrefix + "/status", Value: []byte(strconv.Itoa(int(INITIAL)))}
	if _, err := kv.Put(key, nil); err != nil {
		log.Print(err)
		return nil, nil, taskId, err
	}
	key = &api.KVPair{Key: taskPrefix + "/type", Value: []byte(strconv.Itoa(int(taskType)))}
	if _, err := kv.Put(key, nil); err != nil {
		log.Print(err)
		return nil, nil, taskId, err
	}

	if data != nil {
		for keyM, valM := range data {
			key = &api.KVPair{Key: path.Join(taskPrefix, keyM), Value: []byte(valM)}
			if _, err := kv.Put(key, nil); err != nil {
				log.Print(err)
				return nil, nil, taskId, err
			}
		}
	}

	destroy := func(taskLockCreate *api.Lock, taskId, targetId string) {
		log.Debugf("Unlocking newly created task with id %q (target id %q)", taskId, targetId)
		if err := taskLockCreate.Unlock(); err != nil {
			log.Printf("Can't unlock createLock for task %q (target id %q): %+v", taskId, targetId, err)
		}
		if err := taskLockCreate.Destroy(); err != nil {
			log.Printf("Can't destroy createLock for task %q (target id %q): %+v", taskId, targetId, err)
		}
	}

	return destroy, taskLockCreate, taskId, nil
}
