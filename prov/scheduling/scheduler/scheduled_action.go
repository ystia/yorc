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

package scheduler

import (
	"context"
	"path"
	"sync"
	"time"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/events"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/helper/metricsutil"
	"github.com/ystia/yorc/v3/helper/stringutil"
	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/prov"
	"github.com/ystia/yorc/v3/tasks"
)

type scheduledAction struct {
	prov.Action
	kv                   *api.KV
	deploymentID         string
	timeInterval         time.Duration
	latestDataIndex      uint64
	asyncOperationString string

	latestTaskID       string
	stopScheduling     bool
	stopSchedulingLock sync.Mutex
	chStop             chan struct{}
}

func (sca *scheduledAction) start() {
	sca.stopSchedulingLock.Lock()
	defer sca.stopSchedulingLock.Unlock()

	sca.chStop = make(chan struct{})
	sca.stopScheduling = false
	go sca.schedule()
}

func (sca *scheduledAction) stop() {
	sca.stopSchedulingLock.Lock()
	defer sca.stopSchedulingLock.Unlock()

	if !sca.stopScheduling {
		sca.stopScheduling = true
		close(sca.chStop)
	}
}

func (sca *scheduledAction) schedule() {
	log.Debugf("Scheduling action with ID:%q", sca.ID)
	ticker := time.NewTicker(sca.timeInterval)
	for {
		select {
		case <-sca.chStop:
			log.Debugf("Stop scheduling action with id:%s", sca.ID)
			ticker.Stop()
			return
		case <-ticker.C:
			err := sca.proceed()
			if err != nil {
				log.Printf("Failed to schedule action:%+v due to err:%+v", sca, err)
				// TODO(loicalbertin) ok if we stop the ticker on error this action will never be rescheduled.
				// And the routine will be blocked on select until chStop is closed.
				// Is it really what we want?
				ticker.Stop()
			}
		}
	}
}

func (sca *scheduledAction) proceed() error {
	metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"scheduling", sca.ActionType, sca.ID, "ticks"}), 1)
	// To fit with Task Manager, pass the id/actionType in data
	err := sca.updateData()
	if err != nil {
		return err
	}
	sca.Data["actionType"] = sca.ActionType
	sca.Data["id"] = sca.ID
	sca.Data["asyncOperation"] = sca.asyncOperationString

	if sca.latestTaskID != "" {
		status, err := tasks.GetTaskStatus(sca.kv, sca.latestTaskID)
		// As action task is deleted after being executed asynchronously, we handle if task still exists
		if tasks.IsTaskNotFoundError(err) {
			return sca.registerNewTask()
		}
		if err != nil {
			return err
		}
		if status == tasks.TaskStatusINITIAL || status == tasks.TaskStatusRUNNING {
			ctx := context.Background()
			if sca.AsyncOperation.TaskID != "" {
				ctx = events.AddLogOptionalFields(ctx, events.LogOptionalFields{
					events.ExecutionID:   sca.AsyncOperation.TaskID,
					events.WorkFlowID:    sca.AsyncOperation.WorkflowName,
					events.NodeID:        sca.AsyncOperation.NodeName,
					events.InterfaceName: stringutil.GetAllExceptLastElement(sca.AsyncOperation.Operation.Name, "."),
					events.OperationName: stringutil.GetLastElement(sca.AsyncOperation.Operation.Name, "."),
				})
			}
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, sca.deploymentID).Registerf("Scheduled action: %+v miss trigger due to another execution (task ID: %q) already planned or running. Will try to be rescheduled on next trigger.", sca, sca.latestTaskID)
			metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"scheduling", sca.ActionType, sca.ID, "misses"}), 1)
			return nil
		}
	}

	return sca.registerNewTask()
}

func (sca *scheduledAction) registerNewTask() error {
	var err error
	sca.latestTaskID, err = defaultScheduler.collector.RegisterTaskWithData(sca.deploymentID, tasks.TaskTypeAction, sca.Data)
	if err != nil {
		return err
	}
	consulutil.StoreConsulKeyAsString(path.Join(consulutil.SchedulingKVPrefix, "actions", sca.ID, "latestTaskID"), sca.latestTaskID)
	log.Debugf("Proceed scheduled action with ID:%q with taskID:%q", sca.ID, sca.latestTaskID)
	return nil
}

func (sca *scheduledAction) updateData() error {
	dataPath := path.Join(consulutil.SchedulingKVPrefix, "actions", sca.ID, "data")
	_, meta, err := sca.kv.Get(dataPath, nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if meta.LastIndex > sca.latestDataIndex {
		// re-read data
		kvps, _, err := sca.kv.List(dataPath, nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		for _, kvp := range kvps {
			key := path.Base(kvp.Key)
			sca.Data[key] = string(kvp.Value)
		}
	}
	return nil
}
