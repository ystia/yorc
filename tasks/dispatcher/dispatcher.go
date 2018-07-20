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

package dispatcher

import (
	"time"

	"sync"

	"math"

	"path"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tasks"
)

// Dispatcher concern is polling executions task and dispatch them across available workers
// It has to acquire a lock on the execution task as distributed dispatchers can try to do the same
// If it get the lock, it instantiate an execution task and push it to workers pool
// If it receives a message from shutdown channel, it has to spread the shutdown to workers
type Dispatcher struct {
	client     *api.Client
	shutdownCh chan struct{}
	WorkerPool chan chan *TaskExecution
	maxWorkers int
	cfg        config.Configuration
	wg         *sync.WaitGroup
}

// NewDispatcher create a new Dispatcher with a given number of workers
func NewDispatcher(cfg config.Configuration, shutdownCh chan struct{}, client *api.Client, wg *sync.WaitGroup) *Dispatcher {
	pool := make(chan chan *TaskExecution, cfg.WorkersNumber)
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

func getExecutionKeyValue(kv *api.KV, execID, execKey string) (string, error) {
	execPath := path.Join(consulutil.TasksPrefix, "executions", execID)
	kvPairContent, _, err := kv.Get(path.Join(execPath, execKey), nil)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get value for key due to error: %+v", path.Join(execPath, execKey), err)
	}
	if kvPairContent == nil {
		return "", errors.Errorf("No key:%q", path.Join(execPath, execKey))
	}
	return string(kvPairContent.Value), nil
}

// Run creates workers and polls new task executions
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
		log.Debugf("Long polling TaskExecution")
		execKeys, rMeta, err := kv.Keys(path.Join(consulutil.TasksPrefix, "executions", "/"), "/", q)
		if err != nil {
			err = errors.Wrap(err, "Error getting task executions")
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
		for _, execKey := range execKeys {
			execID := path.Base(execKey)
			taskID, err := getExecutionKeyValue(kv, execID, "TaskID")
			if err != nil {
				log.Printf("%+v", err)
				continue
			}

			log.Debugf("Try to acquire processing lock for task execution %s", execKey)
			opts := &api.LockOptions{
				Key:          execKey + ".processingLock",
				Value:        []byte(nodeName),
				LockTryOnce:  true,
				LockWaitTime: 10 * time.Millisecond,
			}
			lock, err := d.client.LockOpts(opts)
			if err != nil {
				log.Printf("Can't create processing lock for key %s: %+v", execKey, err)
				continue
			}
			leaderChan, err := lock.Lock(nil)
			if err != nil {
				log.Printf("Can't create acquire lock for key %s: %+v", execKey, err)
				continue
			}
			if leaderChan == nil {
				log.Debugf("Another instance got the lock for key %s", execKey)
				continue
			}

			// Check TaskExecution status
			status, err := tasks.GetTaskStatus(kv, taskID)
			if err != nil {
				log.Print(err)
				log.Debugf("%+v", err)
				lock.Unlock()
				lock.Destroy()
				continue
			}
			if status != tasks.TaskStatusINITIAL && status != tasks.TaskStatusRUNNING {
				log.Debugf("Skipping Task Execution with status %q", status)
				lock.Unlock()
				lock.Destroy()
				continue
			}

			log.Debugf("Got processing lock for Task Execution %s", execKey)

			targetID, err := tasks.GetTaskTarget(kv, taskID)
			if err != nil {
				log.Print(err)
				log.Debugf("%+v", err)
				continue
			}
			taskType, err := tasks.GetTaskType(kv, taskID)
			if err != nil {
				log.Print(err)
				log.Debugf("%+v", err)
				continue
			}
			creationDate, err := tasks.GetTaskCreationDate(kv, taskID)
			if err != nil {
				log.Print(err)
				log.Debugf("%+v", err)
				continue
			}

			// Retrieve workflow, Step information in case of workflow Step TaskExecution
			var workflow, step string
			if tasks.IsWorkflowTask(taskType) {
				workflow, err = getExecutionKeyValue(kv, execID, "workflow")
				if err != nil {
					log.Print(err)
					log.Debugf("%+v", err)
				}
				step, err = getExecutionKeyValue(kv, execID, "step")
				if err != nil {
					log.Print(err)
					log.Debugf("%+v", err)
				}
			}

			log.Printf("Processing Task Execution %q linked to deployment %q", taskID, targetID)
			t := &TaskExecution{
				ID:           execID,
				TaskID:       taskID,
				status:       status,
				TargetID:     targetID,
				lock:         lock,
				kv:           kv,
				creationDate: creationDate,
				TaskType:     tasks.TaskType(taskType),
				workflow:     workflow,
				step:         step,
			}
			log.Debugf("New Task Execution created %+v: pushing it to workers channel", t)
			// try to obtain a worker TaskExecution channel until timeout
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
				// Timeout to let another instance a chance to consume this Task Execution
				lock.Unlock()
				lock.Destroy()
				time.Sleep(100 * time.Millisecond)
				break
			}
		}
	}
}
