package tasks

import (
	"fmt"
	"path"
	"strconv"

	"github.com/hashicorp/consul/api"
	"github.com/satori/go.uuid"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

// A Collector is used to register new tasks in Janus
type Collector struct {
	consulClient *api.Client
}

// NewCollector creates a Collector
func NewCollector(consulClient *api.Client) *Collector {
	return &Collector{consulClient: consulClient}
}

// RegisterTaskWithData register an new Task of a given type with some data
//
// The task id is returned.
func (c *Collector) RegisterTaskWithData(targetID string, taskType TaskType, data map[string]string) (string, error) {
	destroy, lock, taskID, err := c.registerTaskWithoutDestroyLock(targetID, taskType, data)
	defer destroy(lock, taskID, targetID)
	if err != nil {
		return "", err
	}
	return taskID, nil
}

// RegisterTask register an new Task of a given type.
//
// The task id is returned.
// Basically this is a shorthand for RegisterTaskWithData(targetID, taskType, nil)
func (c *Collector) RegisterTask(targetID string, taskType TaskType) (string, error) {
	return c.RegisterTaskWithData(targetID, taskType, nil)
}

func (c *Collector) registerTaskWithoutDestroyLock(targetID string, taskType TaskType, data map[string]string) (func(taskLockCreate *api.Lock, taskId, targetId string), *api.Lock, string, error) { // First check if other tasks are running for this target before creating a new one
	hasLivingTask, livingTaskID, livingTaskStatus, err := TargetHasLivingTasks(c.consulClient.KV(), targetID)
	if err != nil {
		return nil, nil, "", err
	} else if hasLivingTask {
		return nil, nil, "", anotherLivingTaskAlreadyExistsError{taskID: livingTaskID, targetID: targetID, status: livingTaskStatus}
	}
	taskID := fmt.Sprint(uuid.NewV4())
	kv := c.consulClient.KV()
	taskPrefix := consulutil.TasksPrefix + "/" + taskID
	// Then use a lock in the task to prevent dispatcher to get the task before finishing task creation
	taskLockCreate, err := c.consulClient.LockKey(taskPrefix + "/.createLock")
	if err != nil {
		return nil, nil, taskID, err
	}
	stopLockChan := make(chan struct{})
	defer close(stopLockChan)
	leaderCh, err := taskLockCreate.Lock(stopLockChan)
	if err != nil {
		log.Printf("Failed to acquire create lock for task with id %q (target id %q): %+v", taskID, targetID, err)
		return nil, nil, taskID, err
	}
	if leaderCh == nil {
		log.Printf("Failed to acquire create lock for task with id %q (target id %q).", taskID, targetID)
		return nil, nil, taskID, fmt.Errorf("Failed to acquire lock for task with id %q (target id %q)", taskID, targetID)
	}

	key := &api.KVPair{Key: taskPrefix + "/targetId", Value: []byte(targetID)}
	if _, err := kv.Put(key, nil); err != nil {
		log.Print(err)
		return nil, nil, taskID, err
	}
	key = &api.KVPair{Key: taskPrefix + "/status", Value: []byte(strconv.Itoa(int(INITIAL)))}
	if _, err := kv.Put(key, nil); err != nil {
		log.Print(err)
		return nil, nil, taskID, err
	}
	key = &api.KVPair{Key: taskPrefix + "/type", Value: []byte(strconv.Itoa(int(taskType)))}
	if _, err := kv.Put(key, nil); err != nil {
		log.Print(err)
		return nil, nil, taskID, err
	}

	if data != nil {
		for keyM, valM := range data {
			key = &api.KVPair{Key: path.Join(taskPrefix, keyM), Value: []byte(valM)}
			if _, err := kv.Put(key, nil); err != nil {
				log.Print(err)
				return nil, nil, taskID, err
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

	return destroy, taskLockCreate, taskID, nil
}
