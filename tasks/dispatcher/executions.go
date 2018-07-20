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
	"context"
	"path"
	"strconv"

	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/tasks"
)

type TaskExecution struct {
	ID           string
	TaskID       string
	TargetID     string
	status       tasks.TaskStatus
	TaskType     tasks.TaskType
	creationDate time.Time
	lock         *api.Lock
	kv           *api.KV
	workflow     string
	step         string
	parentID     string
}

func (t *TaskExecution) releaseLock() {
	t.lock.Unlock()
	t.lock.Destroy()
}

func (t *TaskExecution) Status() tasks.TaskStatus {
	return t.status
}

//FIXME need to be done in concurrency context
func (t *TaskExecution) WithStatus(ctx context.Context, status tasks.TaskStatus) error {
	p := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, t.TaskID, "status"), Value: []byte(strconv.Itoa(int(status)))}
	_, err := t.kv.Put(p, nil)
	t.status = status
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	_, err = tasks.EmitTaskEventWithContextualLogs(ctx, t.kv, t.TargetID, t.TaskID, t.TaskType, t.status.String())

	return err
}
