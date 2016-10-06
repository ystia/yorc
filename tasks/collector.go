package tasks

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/satori/go.uuid"
	"novaforge.bull.com/starlings-janus/janus/log"
	"strconv"
)

type Collector struct {
	consulClient *api.Client
}

func NewCollector(consulClient *api.Client) *Collector {
	return &Collector{consulClient: consulClient}
}

func (c *Collector) RegisterTask(targetId string, taskType TaskType) error {
	taskId := fmt.Sprint(uuid.NewV4())
	kv := c.consulClient.KV()
	taskPrefix := tasksPrefix + "/" + taskId
	taskLock, err := c.consulClient.LockKey(taskPrefix + "/.createLock")
	if err != nil {
		return err
	}
	stopLockChan := make(chan struct{})
	leaderCh, err := taskLock.Lock(stopLockChan)
	if err != nil {
		log.Printf("Failed to acquire lock for task with id %s: %+v", taskId, err)
		return err
	}
	if leaderCh == nil {
		log.Printf("Failed to acquire lock for task with id %s: %+v", taskId, err)
		return fmt.Errorf("Failed to acquire lock for task with id %s", taskId)
	}
	defer func() {
		log.Debugf("Unlocking newly created task with id %s", taskId)
		if err := taskLock.Unlock(); err != nil {
			log.Printf("Can't unlock createLock for task %s: %+v", taskId, err)
		}
		if err := taskLock.Destroy(); err != nil {
			log.Printf("Can't destroy createLock for task %s: %+v", taskId, err)
		}
	}()

	key := &api.KVPair{Key: taskPrefix + "/targetId", Value: []byte(targetId)}
	if _, err := kv.Put(key, nil); err != nil {
		log.Print(err)
		return err
	}
	key = &api.KVPair{Key: taskPrefix + "/status", Value: []byte(strconv.Itoa(int(INITIAL)))}
	if _, err := kv.Put(key, nil); err != nil {
		log.Print(err)
		return err
	}
	key = &api.KVPair{Key: taskPrefix + "/type", Value: []byte(strconv.Itoa(int(taskType)))}
	if _, err := kv.Put(key, nil); err != nil {
		log.Print(err)
		return err
	}
	return nil
}
