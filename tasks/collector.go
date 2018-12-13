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

package tasks

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
)

// A Collector is used to register new tasks in Yorc
//
// Deprecated: use *collector.Collector instead
type Collector struct {
	consulClient *api.Client
}

// NewCollector creates a Collector
//
// Deprecated: use collector.NewCollector instead
func NewCollector(consulClient *api.Client) *Collector {
	return &Collector{consulClient: consulClient}
}

// RegisterTaskWithData register a new Task of a given type with some data
//
// The task id is returned.
//
// Deprecated: use (c *collector.Collector) RegisterTaskWithData instead
func (c *Collector) RegisterTaskWithData(targetID string, taskType TaskType, data map[string]string) (string, error) {
	destroy, lock, taskID, err := c.registerTaskWithoutDestroyLock(targetID, taskType, data)
	if destroy != nil {
		defer destroy(lock, taskID, targetID)
	}
	if err != nil {
		return "", err
	}
	return taskID, nil
}

// RegisterTask register a new Task of a given type.
//
// The task id is returned.
// Basically this is a shorthand for RegisterTaskWithData(targetID, taskType, nil)
//
// Deprecated: use (c *collector.Collector) RegisterTask instead
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
		log.Debugf("Failed to acquire create lock for task with id %q (target id %q): %+v", taskID, targetID, err)
		return nil, nil, taskID, err
	}
	if leaderCh == nil {
		log.Debugf("Failed to acquire create lock for task with id %q (target id %q).", taskID, targetID)
		return nil, nil, taskID, errors.Errorf("Failed to acquire lock for task with id %q (target id %q)", taskID, targetID)
	}
	_, errGrp, store := consulutil.WithContext(context.Background())
	store.StoreConsulKeyAsString(taskPrefix+"/targetId", targetID)
	store.StoreConsulKeyAsString(taskPrefix+"/status", strconv.Itoa(int(TaskStatusINITIAL)))
	store.StoreConsulKeyAsString(taskPrefix+"/type", strconv.Itoa(int(taskType)))
	dateBin, err := time.Now().MarshalBinary()
	if err != nil {
		return nil, nil, taskID, errors.Wrap(err, "Failed to generate task creation date")
	}
	store.StoreConsulKey(taskPrefix+"/creationDate", dateBin)
	if data != nil {
		for keyM, valM := range data {
			store.StoreConsulKeyAsString(path.Join(taskPrefix, "data", keyM), valM)
		}
	}

	err = errGrp.Wait()
	if err != nil {
		return nil, nil, taskID, err
	}
	EmitTaskEventWithContextualLogs(nil, kv, targetID, taskID, taskType, "unknown", TaskStatusINITIAL.String())

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
