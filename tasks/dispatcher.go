package tasks

import (
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

// A Dispatcher is in charge to look for new tasks and dispatch them accross availables workers
type Dispatcher struct {
	client     *api.Client
	shutdownCh chan struct{}
	WorkerPool chan chan *task
	maxWorkers int
	cfg        config.Configuration
}

// NewDispatcher create a new Dispatcher with a given number of workers
func NewDispatcher(maxWorkers int, shutdownCh chan struct{}, client *api.Client, cfg config.Configuration) *Dispatcher {
	pool := make(chan chan *task, maxWorkers)
	return &Dispatcher{WorkerPool: pool, client: client, shutdownCh: shutdownCh, maxWorkers: maxWorkers, cfg: cfg}
}

// Run creates workers and waits for new tasks
func (d *Dispatcher) Run() {

	for i := 0; i < d.maxWorkers; i++ {
		worker := newWorker(d.WorkerPool, d.shutdownCh, d.client, d.cfg)
		worker.Start()
	}
	log.Printf("%d worker started", d.maxWorkers)
	var waitIndex uint64
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
		tasksKeys, rMeta, err := kv.Keys(consulutil.TasksPrefix+"/", "/", q)
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
			status, err := checkTaskStatus(kv, taskKey)

			if err != nil {
				log.Print(err)
				log.Debugf("%+v", err)
				continue
			}

			if status != INITIAL && status != RUNNING {
				log.Printf("Skipping task with status %q", status)
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

			status, err = checkTaskStatus(kv, taskKey)

			if err != nil {
				log.Print(err)
				log.Debugf("%+v", err)
				lock.Unlock()
				lock.Destroy()
				continue
			}

			if status != INITIAL && status != RUNNING {
				log.Printf("Skipping task with status %q", status)
				lock.Unlock()
				lock.Destroy()
				continue
			}

			log.Debugf("Got processing lock for task %s", taskKey)
			// Got lock
			kvPairContent, _, err := kv.Get(taskKey+"targetId", nil)
			if err != nil {
				log.Printf("Failed to get targetId for key %s: %+v", taskKey, err)
				continue
			}
			if kvPairContent == nil {
				log.Printf("Failed to get targetId for key %s: nil value", taskKey)
				continue
			}
			targetID := string(kvPairContent.Value)

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
			var taskID string
			for i := len(keyPath) - 1; i >= 0; i-- {
				if keyPath[i] != "" {
					taskID = keyPath[i]
					break
				}
			}

			log.Printf("Processing task %q linked to deployment %q", taskID, targetID)
			t := &task{ID: taskID, status: status, TargetID: targetID, taskLock: lock, kv: kv, TaskType: TaskType(taskType)}
			log.Debugf("New task created %+v: pushing it to a work channel", t)
			// try to obtain a worker task channel that is available.
			// this will block until a worker is idle
			select {
			case taskChannel := <-d.WorkerPool:
				taskChannel <- t
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

func checkTaskStatus(kv *api.KV, taskKey string) (TaskStatus, error) {
	kvPairContent, _, err := kv.Get(taskKey+"status", nil)
	if err != nil {
		return FAILED, errors.Wrapf(err, "Failed to get status for key %s: %+v", taskKey, err)
	}
	if kvPairContent == nil {
		return FAILED, errors.Errorf("Failed to get status for key %s: nil value", taskKey)
	}

	statusInt, err := strconv.Atoi(string(kvPairContent.Value))
	if err != nil {
		return FAILED, errors.Wrapf(err, "Failed to get status for key %s", taskKey)
	}
	return TaskStatus(statusInt), nil

}
