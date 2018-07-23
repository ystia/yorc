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
	"path"
	"strconv"

	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tasks"
)

// A TaskExecution is the unit task of execution. If task is a workflow, it contains as many TaskExecutions as workflow steps
type TaskExecution struct {
	ID           string
	TaskID       string
	TargetID     string
	status       tasks.StepStatus
	TaskType     tasks.TaskType
	creationDate time.Time
	lock         *api.Lock
	kv           *api.KV
	step         string
}

func (t *TaskExecution) releaseLock() {
	t.lock.Unlock()
	t.lock.Destroy()
}

func (t *TaskExecution) getTaskStatus() (tasks.TaskStatus, error) {
	return tasks.GetTaskStatus(t.kv, t.TaskID)
}

func (t *TaskExecution) checkAndSetTaskStatus(ctx context.Context, requiredStatus, status tasks.TaskStatus) error {
	status, err := t.getTaskStatus()
	if err != nil {
		return err
	}

	if status == requiredStatus {
		return t.setTaskStatus(ctx, status)
	}
	return errors.Errorf("Required status is:%q but actual status is:%s", requiredStatus.String(), status.String())
}

func (t *TaskExecution) setTaskStatus(ctx context.Context, status tasks.TaskStatus) error {
	p := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, t.TaskID, "status"), Value: []byte(strconv.Itoa(int(status)))}
	_, err := t.kv.Put(p, nil)
	if err != nil {
		log.Printf("Failed to set status to %q for taskID:%q due to error:%v", status.String(), t.TaskID, err)
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	_, err = tasks.EmitTaskEventWithContextualLogs(ctx, t.kv, t.TargetID, t.TaskID, t.TaskType, status.String())

	return err
}
