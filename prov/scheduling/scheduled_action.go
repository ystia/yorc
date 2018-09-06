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

package scheduling

import (
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
	"sync"
	"time"
)

type scheduledAction struct {
	id           string
	deploymentID string
	timeInterval time.Duration
	action       prov.Action

	stopScheduling     bool
	stopSchedulingLock sync.Mutex
	chStop             chan struct{}
}

func newScheduledAction(id, deploymentID string, timeInterval time.Duration, action prov.Action) *scheduledAction {
	return &scheduledAction{
		id:           id,
		deploymentID: deploymentID,
		timeInterval: timeInterval,
		action:       action,
	}
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
			log.Debugf("Stop scheduling action with id:%s", sca.id)
			ticker.Stop()
			return
		case <-ticker.C:
			err := run(sca)
			if err != nil {
				log.Printf("Failed to schedule action:%+v due to err:%+v", sca, err)
				ticker.Stop()
			}
		}
	}
}
