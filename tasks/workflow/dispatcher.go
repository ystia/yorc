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

package workflow

import (
	"path"
	"strings"
	"sync"
	"time"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tasks"
)

const executionLockPrefix = ".processingLock-"

// Dispatcher concern is polling executions task and dispatch them across available workers
// It has to acquire a lock on the execution task as other distributed dispatchers can try to do the same
// If it gets the lock, it instantiates an execution task and push it to workers pool
// If it receives a message from shutdown channel, it has to spread the shutdown to workers
type Dispatcher struct {
	client           *api.Client
	shutdownCh       chan struct{}
	WorkerPool       chan chan *taskExecution
	maxWorkers       int
	cfg              config.Configuration
	wg               *sync.WaitGroup
	createWorkerFunc func(*Dispatcher)
}

// NewDispatcher create a new Dispatcher with a given number of workers
func NewDispatcher(cfg config.Configuration, shutdownCh chan struct{}, client *api.Client, wg *sync.WaitGroup) *Dispatcher {
	pool := make(chan chan *taskExecution, cfg.WorkersNumber)
	dispatcher := &Dispatcher{WorkerPool: pool, client: client, shutdownCh: shutdownCh, maxWorkers: cfg.WorkersNumber, cfg: cfg, wg: wg, createWorkerFunc: createWorker}
	dispatcher.emitMetrics(client)
	return dispatcher
}

// getTaskExecsNbWait calculates the number of task executions that wait
func (d *Dispatcher) getTaskExecsNbWait(client *api.Client) (float32, error) {
	var nb float32
	tasksKeys, err := consulutil.GetKeys(consulutil.TasksPrefix)
	if err != nil {
		return 0, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if tasksKeys == nil {
		return 0, nil
	}
	for _, taskKey := range tasksKeys {
		taskID := path.Base(taskKey)
		nbExec, err := numberOfWaitingExecutionsForTask(client, taskID)
		if err != nil {
			return 0, err
		}
		nb = nb + float32(nbExec)
	}
	return nb, nil
}

func (d *Dispatcher) emitMetrics(client *api.Client) {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		lastWarn := time.Now().Add(-6 * time.Minute)
		for {
			select {
			case <-time.After(d.cfg.Tasks.Dispatcher.MetricsRefreshTime):
				d.emitWorkersMetrics()
				d.emitTaskExecutionsMetrics(client, &lastWarn)
			case <-d.shutdownCh:
				return
			}
		}
	}()
}

func (d *Dispatcher) emitWorkersMetrics() {
	metrics.SetGauge([]string{"workers", "free"}, float32(len(d.WorkerPool)))
}

func (d *Dispatcher) emitTaskExecutionsMetrics(client *api.Client, lastWarn *time.Time) {
	nbWaiting, err := d.getTaskExecsNbWait(client)
	if err != nil {
		now := time.Now()
		if now.Sub(*lastWarn) > 5*time.Minute {
			// Do not print each time
			*lastWarn = now
			log.Printf("Warning: Failed to get metrics for taskExecutions: %+v", err)
		}
		return
	}
	metrics.SetGauge([]string{"taskExecutions", "nbWaiting"}, nbWaiting)
}

func getExecutionKeyValue(execID, execKey string) (string, error) {
	execPath := path.Join(consulutil.ExecutionsTaskPrefix, execID)
	exist, value, err := consulutil.GetStringValue(path.Join(execPath, execKey))
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get value for key %q due to error: %+v", path.Join(execPath, execKey), err)
	}
	if !exist {
		return "", errors.Errorf("No key:%q", path.Join(execPath, execKey))
	}
	return value, nil
}

func (d *Dispatcher) deleteExecutionTree(execID string) {
	// remove execution kv tree
	log.Debugf("Remove task execution tree with ID:%q", execID)
	_, err := d.client.KV().DeleteTree(path.Join(consulutil.ExecutionsTaskPrefix, execID)+"/", nil)
	if err != nil {
		log.Printf("Failed to remove execution KV tree with ID:%q due to error:%+v", execID, err)
		return
	}
}

func createWorker(d *Dispatcher) {
	worker := newWorker(d.WorkerPool, d.shutdownCh, d.client, d.cfg)
	worker.Start()
}

// Run creates workers and polls new task executions
func (d *Dispatcher) Run() {

	for i := 0; i < d.maxWorkers; i++ {
		d.createWorkerFunc(d)
	}
	log.Printf("%d workers started", d.maxWorkers)
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
		q := &api.QueryOptions{
			WaitIndex: waitIndex,
			WaitTime:  d.cfg.Tasks.Dispatcher.LongPollWaitTime,
		}
		log.Debugf("Long polling Task Executions")
		execKeys, rMeta, err := kv.Keys(consulutil.ExecutionsTaskPrefix+"/", "/", q)
		if err != nil {
			err = errors.Wrap(err, "Error getting task executions")
			log.Print(err)
			log.Debugf("%+v", err)
			continue
		}
		waitIndex = rMeta.LastIndex
		log.Debugf("Got response new wait index is %d", waitIndex)
		for _, execKey := range execKeys {
			execID := path.Base(execKey)
			// Ignore locks
			if strings.HasPrefix(execID, executionLockPrefix) {
				continue
			}

			log.Debugf("Try to acquire processing lock for task execution %s", execKey)
			opts := &api.LockOptions{
				Key:   path.Join(consulutil.ExecutionsTaskPrefix, executionLockPrefix+execID),
				Value: []byte(nodeName),
				SessionOpts: &api.SessionEntry{
					Name:     "DispatcherLock-" + nodeName,
					Behavior: api.SessionBehaviorRelease,
					Checks:   []string{"service:yorc", "serfHealth"},
					TTL:      "10s",
				},
				SessionTTL:   "10s",
				LockTryOnce:  true,
				LockWaitTime: d.cfg.Tasks.Dispatcher.LockWaitTime,
			}
			lock, err := d.client.LockOpts(opts)
			if err != nil {
				log.Printf("Can't create processing lock for key %s: %+v", execKey, err)
				continue
			}
			leaderChan, err := lock.Lock(d.shutdownCh)
			if err != nil {
				log.Printf("Can't create acquire lock for key %s: %+v", execKey, err)
				continue
			}
			if leaderChan == nil {
				log.Debugf("Can not acquire lock for execution key %s", execKey)
				continue
			}

			log.Debugf("Got processing lock for Task Execution %s", execKey)

			taskID, err := getExecutionKeyValue(execID, "taskID")
			if err != nil {
				log.Debugf("Ignore execution with ID:%q due to error:%v", execID, err)
				d.deleteExecutionTree(execID)
				lock.Unlock()
				lock.Destroy()
				continue
			}
			// Check TaskExecution status
			status, err := tasks.GetTaskStatus(taskID)
			if err != nil {
				log.Debugf("Seems the task with id:%q is no longer relevant due to error:%s so related execution will be removed", taskID, err)
				d.deleteExecutionTree(execID)
				lock.Unlock()
				lock.Destroy()
				continue
			}
			if status != tasks.TaskStatusINITIAL && status != tasks.TaskStatusRUNNING {
				log.Debugf("Skipping Task Execution with status %q", status)
				// Delete useless execution
				d.deleteExecutionTree(execID)
				lock.Unlock()
				lock.Destroy()
				continue
			}

			inProgress, err := tasks.IsStepRegistrationInProgress(taskID)
			if err != nil {
				log.Printf("Can't check if task %s registration is in progress %+v", taskID, err)
				continue
			}
			if inProgress {
				log.Debugf("Task %s registration is still in progress", taskID, err)
				lock.Unlock()
				lock.Destroy()
				continue
			}

			t, err := buildTaskExecution(d.client, execID)
			if err != nil {
				log.Print(err)
				log.Debugf("%+v", err)
				lock.Unlock()
				lock.Destroy()
				continue
			}
			log.Printf("Processing Task Execution %q linked to deployment %q", execID, t.targetID)
			t.lock = lock
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
			case <-time.After(2 * time.Second):
				log.Debugf("Release the lock for execID:%q to let another yorc instance worker take the execution", execID)
				lock.Unlock()
				lock.Destroy()
				time.Sleep(100 * time.Millisecond)
				break
			}
		}
	}
}
