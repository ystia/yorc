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
	"path"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/helper/consulutil"

	"github.com/hashicorp/consul/api"

	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
	"github.com/ystia/yorc/tasks"
)

type scheduledAction struct {
	prov.Action
	kv                   *api.KV
	deploymentID         string
	timeInterval         time.Duration
	latestDataIndex      uint64
	asyncOperationString string

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
	log.Debugf("Scheduling action:%+v", sca)
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
				ticker.Stop()
			}
		}
	}
}

func (sca *scheduledAction) proceed() error {
	// To fit with Task Manager, pass the id/actionType in data
	err := sca.updateData()
	if err != nil {
		return err
	}
	sca.Data["actionType"] = sca.ActionType
	sca.Data["id"] = sca.ID
	sca.Data["asyncOperation"] = sca.asyncOperationString
	taskID, err := defaultScheduler.collector.RegisterTaskWithData(sca.deploymentID, tasks.TaskTypeAction, sca.Data)
	if err != nil {
		return err
	}
	log.Debugf("Proceed scheduled action:%+v with taskID:%q", sca, taskID)
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
