package tasks

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/satori/go.uuid"
	"novaforge.bull.com/starlings-janus/janus/log"
	"strconv"
	"time"
)

type Collector struct {
	consulClient *api.Client
}

func NewCollector(consulClient *api.Client) *Collector {
	return &Collector{consulClient: consulClient}
}

func (c *Collector) RegisterTask(targetId string, taskType TaskType) (string, error) {
	// First check if other tasks are running for this target before creating a new one
	taskLock, err := NewTaskLockForTarget(c.consulClient, targetId)
	_, err = taskLock.Lock(10, 500*time.Millisecond)
	if err != nil {
		return "", err
	}
	defer taskLock.Release()
	if hasLivingTask, livingTaskId, livingTaskStatus, err := TargetHasLivingTasks(c.consulClient.KV(), targetId); err != nil {
		return "", err
	} else if hasLivingTask {
		return "", anotherLivingTaskAlreadyExistsError{taskId: livingTaskId, targetId: targetId, status: livingTaskStatus}
	}
	taskId := fmt.Sprint(uuid.NewV4())
	kv := c.consulClient.KV()
	taskPrefix := tasksPrefix + "/" + taskId
	// Then use a lock in the task to prevent dispatcher to get the task before finishing task creation
	taskLockCreate, err := c.consulClient.LockKey(taskPrefix + "/.createLock")
	if err != nil {
		return taskId, err
	}
	stopLockChan := make(chan struct{})
	defer close(stopLockChan)
	leaderCh, err := taskLockCreate.Lock(stopLockChan)
	if err != nil {
		log.Printf("Failed to acquire create lock for task with id %q (target id %q): %+v", taskId, targetId, err)
		return taskId, err
	}
	if leaderCh == nil {
		log.Printf("Failed to acquire create lock for task with id %q (target id %q).", taskId, targetId)
		return taskId, fmt.Errorf("Failed to acquire lock for task with id %q (target id %q)", taskId, targetId)
	}
	defer func() {
		log.Debugf("Unlocking newly created task with id %q (target id %q)", taskId, targetId)
		if err := taskLockCreate.Unlock(); err != nil {
			log.Printf("Can't unlock createLock for task %q (target id %q): %+v", taskId, targetId, err)
		}
		if err := taskLockCreate.Destroy(); err != nil {
			log.Printf("Can't destroy createLock for task %q (target id %q): %+v", taskId, targetId, err)
		}
	}()

	key := &api.KVPair{Key: taskPrefix + "/targetId", Value: []byte(targetId)}
	if _, err := kv.Put(key, nil); err != nil {
		log.Print(err)
		return taskId, err
	}
	key = &api.KVPair{Key: taskPrefix + "/status", Value: []byte(strconv.Itoa(int(INITIAL)))}
	if _, err := kv.Put(key, nil); err != nil {
		log.Print(err)
		return taskId, err
	}
	key = &api.KVPair{Key: taskPrefix + "/type", Value: []byte(strconv.Itoa(int(taskType)))}
	if _, err := kv.Put(key, nil); err != nil {
		log.Print(err)
		return taskId, err
	}
	return taskId, nil
}
