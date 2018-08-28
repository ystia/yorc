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
	"context"
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tasks"
)

// A taskExecution is the unit task of execution. If task is a workflow, it contains as many TaskExecutions as workflow steps
type taskExecution struct {
	id           string
	taskID       string
	targetID     string
	taskType     tasks.TaskType
	creationDate time.Time
	lock         *api.Lock
	kv           *api.KV
	step         string
}

func (t *taskExecution) releaseLock() {
	t.lock.Unlock()
	t.lock.Destroy()
}

func (t *taskExecution) getTaskStatus() (tasks.TaskStatus, error) {
	return tasks.GetTaskStatus(t.kv, t.taskID)
}

// checkAndSetTaskStatus allows to check the task status before updating it
func (t *taskExecution) checkAndSetTaskStatus(ctx context.Context, finalStatus tasks.TaskStatus) error {
	kvp, meta, err := t.kv.Get(path.Join(consulutil.TasksPrefix, t.taskID, "status"), nil)
	if err != nil {
		return errors.Wrapf(err, "Failed to get task status for taskID:%q", t.taskID)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return errors.Wrapf(err, "Missing task status for taskID:%q", t.taskID)
	}
	st, err := strconv.Atoi(string(kvp.Value))
	if err != nil {
		return errors.Wrapf(err, "Invalid task status for taskID:%q", t.taskID)
	}

	status := tasks.TaskStatus(st)
	if finalStatus != status {
		if status == tasks.TaskStatusFAILED {
			mess := fmt.Sprintf("Can't set task status with taskID:%q to:%q because task status is FAILED", t.taskID, finalStatus.String())
			log.Printf(mess)
			return errors.Errorf(mess)
		}
		return t.setTaskStatus(ctx, finalStatus, meta.LastIndex)
	}
	return nil
}

func (t *taskExecution) setTaskStatus(ctx context.Context, status tasks.TaskStatus, lastIndex uint64) error {
	p := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, t.taskID, "status"), Value: []byte(strconv.Itoa(int(status)))}
	p.ModifyIndex = lastIndex
	set, _, err := t.kv.CAS(p, nil)
	if err != nil {
		log.Printf("Failed to set status to %q for taskID:%q due to error:%+v", status.String(), t.taskID, err)
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !set {
		log.Debugf("[WARNING] Failed to set task status to:%q for taskID:%q as last index has been changed before. Retry it", status.String(), t.taskID)
		return t.checkAndSetTaskStatus(ctx, status)
	}
	if status == tasks.TaskStatusFAILED {
		return t.addTaskErrorFlag(ctx)
	}
	return nil
}

// Add task error flag for monitoring failures in case of workflow task execution
func (t *taskExecution) addTaskErrorFlag(ctx context.Context) error {
	if t.step != "" {
		log.Debugf("Create error flag key for taskID:%q", t.taskID)
		p := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, t.taskID, ".errorFlag")}
		_, err := t.kv.Put(p, nil)
		if err != nil {
			log.Printf("Failed to set error flag for taskID:%q due to error:%+v", t.taskID, err)
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
	}
	return nil
}

func (t *taskExecution) deleteTaskErrorFlag(ctx context.Context, kvp *api.KVPair, lastIndex uint64) error {
	log.Debugf("Try to delete error flag key for taskID:%q", t.taskID)
	kvp.ModifyIndex = lastIndex
	del, _, err := t.kv.DeleteCAS(kvp, nil)
	if err != nil {
		log.Printf("Failed to delete error flag for taskID:%q due to error:%+v", t.taskID, err)
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !del {
		return errors.Errorf("Failed to delete task error flag for taskID:%q as last index has been changed before", t.taskID)
	}
	return nil
}
