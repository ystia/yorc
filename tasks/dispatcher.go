package tasks

import (
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/log"
	"strconv"
	"strings"
	"time"
)

type Dispatcher struct {
	client     *api.Client
	shutdownCh chan struct{}
	WorkerPool chan chan *Task
	maxWorkers int
	cfg        config.Configuration
}

func NewDispatcher(maxWorkers int, shutdownCh chan struct{}, client *api.Client, cfg config.Configuration) *Dispatcher {
	pool := make(chan chan *Task, maxWorkers)
	return &Dispatcher{WorkerPool: pool, client: client, shutdownCh: shutdownCh, maxWorkers: maxWorkers, cfg: cfg}
}

func (d *Dispatcher) Run() {

	for i := 0; i < d.maxWorkers; i++ {
		worker := NewWorker(d.WorkerPool, d.shutdownCh, d.client, d.cfg)
		worker.Start()
	}
	log.Printf("%d worker started", d.maxWorkers)
	var waitIndex uint64 = 0
	kv := d.client.KV()
	nodeName, err := d.client.Agent().NodeName()
	if err != nil {
		log.Panicf("Can't connect to Consul %+v... Aborting", err)
	}

	for {
		select {
		case <-d.shutdownCh:
			log.Printf("Dispatcher received shutdown signal. Exiting...")
			return
		default:
		}
		q := &api.QueryOptions{WaitIndex: waitIndex}
		log.Debugf("Long pooling task list")
		tasksKeys, rMeta, err := kv.Keys(tasksPrefix+"/", "/", q)
		if err != nil {
			log.Printf("Error getting tasks list: %+v", err)
			continue
		}
		if waitIndex == rMeta.LastIndex {
			// long pool ended due to a timeout
			// there is no new items go back to the pooling
			continue
		}
		waitIndex = rMeta.LastIndex
		log.Debugf("Got response new wait index is %d", waitIndex)
		for _, taskKey := range tasksKeys {
			log.Debugf("Check if createLock exists for task %s", taskKey)
			for {
				if createLock, _, _ := kv.Get(taskKey+".createLock", nil); createLock != nil {
					// Locked in creation let's it finish
					log.Debugf("CreateLock exists for task %s wait for few ms", taskKey)
					time.Sleep(100 * time.Millisecond)
				} else {
					break
				}
			}
			kvPairContent, _, err := kv.Get(taskKey+"status", nil)
			if err != nil {
				log.Printf("Failed to get status for key %s: %+v", taskKey, err)
				continue
			}
			if kvPairContent == nil {
				log.Printf("Failed to get status for key %s: nil value", taskKey)
				continue
			}
			statusInt, err := strconv.Atoi(string(kvPairContent.Value))
			if err != nil {
				log.Printf("Failed to get status for key %s: %+v", taskKey, err)
				continue
			}
			status := TaskStatus(statusInt)
			if status != INITIAL && status != RUNNING {
				log.Debugf("Skiping task %s with status %s", taskKey, status)
				continue
			}

			log.Debugf("Try to acquire processing lock for task %s", taskKey)
			opts := &api.LockOptions{
				Key:          taskKey + ".processingLock",
				Value:        []byte(nodeName),
				LockTryOnce:  true,
				LockWaitTime: 10 * time.Millisecond,
			}
			lock, err := d.client.LockOpts(opts)
			if err != nil {
				log.Printf("Can't create processing lock for key %s: %+v", taskKey, err)
				continue
			}
			leaderChan, err := lock.Lock(nil)
			if err != nil {
				log.Printf("Can't create acquire lock for key %s: %+v", taskKey, err)
				continue
			}
			if leaderChan == nil {
				log.Debugf("Another instance got the lock for key %s", taskKey)
				continue
			}
			log.Debugf("Got processing lock for task %s", taskKey)
			// Got lock
			kvPairContent, _, err = kv.Get(taskKey+"targetId", nil)
			if err != nil {
				log.Printf("Failed to get targetId for key %s: %+v", taskKey, err)
				continue
			}
			if kvPairContent == nil {
				log.Printf("Failed to get targetId for key %s: nil value", taskKey)
				continue
			}
			targetId := string(kvPairContent.Value)

			kvPairContent, _, err = kv.Get(taskKey+"type", nil)
			if err != nil {
				log.Printf("Failed to get task type for key %s: %+v", taskKey, err)
				continue
			}
			if kvPairContent == nil {
				log.Printf("Failed to get task type for key %s: nil value", taskKey)
				continue
			}
			taskType, err := strconv.Atoi(string(kvPairContent.Value))
			if err != nil {
				log.Printf("Failed to get task type for key %s: %+v", taskKey, err)
				continue
			}

			keyPath := strings.Split(taskKey, "/")
			log.Debugf("%+q", keyPath)
			var taskId string
			for i := len(keyPath) - 1; i >= 0; i-- {
				if keyPath[i] != "" {
					taskId = keyPath[i]
					break
				}
			}

			log.Printf("Processing task %q linked to deployment %q", taskId, targetId)
			task := &Task{Id: taskId, status: status, TargetId: targetId, taskLock: lock, kv: kv, TaskType: TaskType(taskType)}
			log.Debugf("New task created %+v: pushing it to a work channel", task)
			// try to obtain a worker task channel that is available.
			// this will block until a worker is idle
			select {
			case taskChannel := <-d.WorkerPool:
				taskChannel <- task
			case <-leaderChan:
				// lock lost
				continue
			case <-d.shutdownCh:
				lock.Unlock()
				lock.Destroy()
				log.Printf("Dispatcher received shutdown signal. Exiting...")
				return
			case <-time.After(5 * time.Second):
				// Timeout let another instance a chance to consume this
				// task
				lock.Unlock()
				lock.Destroy()
				time.Sleep(100 * time.Millisecond)
				break
			}

		}
	}

}
