// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tasks_refactor

import (
	metrics "github.com/armon/go-metrics"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tasks"
	"math"
	"path"
	"strconv"
	"sync"
	"time"
)

// NEEDS
// poll executions instead of tasks ? to dispatch acroos workers

// A Dispatcher is in charge to look for new tasks and dispatch them across available workers
type Dispatcher struct {
	client     *api.Client
	shutdownCh chan struct{}
	WorkerPool chan chan *task
	maxWorkers int
	cfg        config.Configuration
	wg         *sync.WaitGroup
}

// NewDispatcher create a new Dispatcher with a given number of workers
func NewDispatcher(cfg config.Configuration, shutdownCh chan struct{}, client *api.Client, wg *sync.WaitGroup) *Dispatcher {
	pool := make(chan chan *task, cfg.WorkersNumber)
	dispatcher := &Dispatcher{WorkerPool: pool, client: client, shutdownCh: shutdownCh, maxWorkers: cfg.WorkersNumber, cfg: cfg, wg: wg}
	dispatcher.emitTasksMetrics()
	return dispatcher
}

func getNbAndMaxTasksWaitTimeMs(kv *api.KV) (float32, float64, error) {
	now := time.Now()
	var max float64
	var nb float32
	tasksKeys, _, err := kv.Keys(consulutil.TasksPrefix+"/", "/", nil)
	if err != nil {
		return nb, max, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, taskKey := range tasksKeys {
		taskID := path.Base(taskKey)
		status, err := tasks.GetTaskStatus(kv, taskID)
		if err != nil {
			return nb, max, err
		}
		if status == tasks.TaskStatusINITIAL {
			nb++
			createDate, err := tasks.GetTaskCreationDate(kv, path.Base(taskKey))
			if err != nil {
				return nb, max, err
			}
			max = math.Max(max, float64(now.Sub(createDate)/time.Millisecond))
		}
	}
	return nb, max, nil
}

func (d *Dispatcher) emitTasksMetrics() {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		kv := d.client.KV()
		lastWarn := time.Now().Add(-6 * time.Minute)
		for {
			select {
			case <-time.After(time.Second):
				metrics.SetGauge([]string{"workers", "free"}, float32(len(d.WorkerPool)))
				nb, maxWait, err := getNbAndMaxTasksWaitTimeMs(kv)
				if err != nil {
					now := time.Now()
					if now.Sub(lastWarn) > 5*time.Minute {
						// Do not print each time
						lastWarn = now
						log.Printf("Warning: Failed to get Max Blocked duration for tasks: %v", err)
					}
					continue
				}
				metrics.AddSample([]string{"tasks", "maxBlockTimeMs"}, float32(maxWait))
				metrics.SetGauge([]string{"tasks", "nbWaiting"}, nb)
			case <-d.shutdownCh:
				return
			}
		}
	}()
}

// Run creates workers and polls new tasks
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
		log.Debugf("Long polling task list")
		tasksKeys, rMeta, err := kv.Keys(consulutil.TasksPrefix+"/", "/", q)
		if err != nil {
			err = errors.Wrap(err, "Error getting tasks list")
			log.Print(err)
			log.Debugf("%+v", err)
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
			taskID := path.Base(taskKey)
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
			status, err := GetTaskStatus(kv, taskID)

			if err != nil {
				log.Print(err)
				log.Debugf("%+v", err)
				continue
			}

			if status != TaskStatusINITIAL && status != TaskStatusRUNNING {
				log.Debugf("Skipping task with status %q", status)
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

			status, err = GetTaskStatus(kv, taskID)

			if err != nil {
				log.Print(err)
				log.Debugf("%+v", err)
				lock.Unlock()
				lock.Destroy()
				continue
			}

			if status != TaskStatusINITIAL && status != TaskStatusRUNNING {
				log.Debugf("Skipping task with status %q", status)
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
			kvPairContent, _, err = kv.Get(taskKey+"creationDate", nil)
			if err != nil {
				log.Printf("Failed to get task creationDate for key %s: %+v", taskKey, err)
				continue
			}
			if kvPairContent == nil {
				log.Printf("Failed to get task creationDate for key %s: nil value", taskKey)
				continue
			}
			creationDate := time.Time{}
			err = creationDate.UnmarshalBinary(kvPairContent.Value)
			if err != nil {
				log.Printf("Failed to get task creationDate for key %s: %+v", taskKey, err)
				continue
			}

			log.Printf("Processing task %q linked to deployment %q", taskID, targetID)
			t := &task{
				ID:           taskID,
				status:       status,
				TargetID:     targetID,
				taskLock:     lock,
				kv:           kv,
				creationDate: creationDate,
				TaskType:     TaskType(taskType),
			}
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
