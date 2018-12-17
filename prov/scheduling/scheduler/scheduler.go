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
	"encoding/json"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tasks/collector"
)

var defaultScheduler *scheduler

type scheduler struct {
	cc               *api.Client
	collector        *collector.Collector
	chStopScheduling chan struct{}
	serviceKey       string
	chShutdown       chan struct{}
	isActive         bool
	isActiveLock     sync.Mutex
	cfg              config.Configuration
	actions          map[string]*scheduledAction
}

// unregisterAction allows to unregister a scheduled action
func (sc *scheduler) unregisterAction(id string) error {
	log.Debugf("Removing scheduled action with id:%q", id)
	scaPath := path.Join(consulutil.SchedulingKVPrefix, "actions", id)
	_, err := sc.cc.KV().DeleteTree(scaPath, nil)
	if err != nil {
		return errors.Wrapf(err, "Failed to delete scheduled action with id:%q", id)
	}
	return nil
}

// Start allows to instantiate a default scheduler, poll for scheduled actions and schedule them
func Start(cfg config.Configuration, cc *api.Client) {
	defaultScheduler = &scheduler{
		cc:         cc,
		collector:  collector.NewCollector(cc),
		chShutdown: make(chan struct{}),
		isActive:   false,
		serviceKey: path.Join(consulutil.YorcServicePrefix, "/scheduling/leader"),
		cfg:        cfg,
	}
	// Watch leader election for scheduler
	go consulutil.WatchLeaderElection(defaultScheduler.cc, defaultScheduler.serviceKey, defaultScheduler.chShutdown, defaultScheduler.startScheduling, defaultScheduler.stopScheduling)
}

// Stop allows to stop polling and schedule actions
func Stop() {
	// Stop scheduling actions
	defaultScheduler.stopScheduling()

	// Stop watch leader election
	close(defaultScheduler.chShutdown)
}

func handleError(err error) {
	err = errors.Wrap(err, "[WARN] Error during polling scheduled actions")
	log.Print(err)
	log.Debugf("%+v", err)
}

func (sc *scheduler) startScheduling() {
	if sc.isActive {
		log.Println("Scheduling service is already running.")
		return
	}
	sc.isActiveLock.Lock()
	sc.isActive = true
	sc.isActiveLock.Unlock()
	sc.chStopScheduling = make(chan struct{})
	sc.actions = make(map[string]*scheduledAction)
	var waitIndex uint64
	go func() {
		for {
			select {
			case <-sc.chStopScheduling:
				log.Debugf("Ending scheduling service has been requested: stop it now.")
				return
			case <-sc.chShutdown:
				log.Debugf("Shutdown has been sent: stop scheduled actions now.")
				return
			default:
			}

			q := &api.QueryOptions{WaitIndex: waitIndex}
			actions, rMeta, err := sc.cc.KV().Keys(path.Join(consulutil.SchedulingKVPrefix, "actions")+"/", "/", q)
			log.Debugf("Found %d actions", len(actions))
			if err != nil {
				handleError(err)
				continue
			}
			if waitIndex == rMeta.LastIndex {
				// long pool ended due to a timeout
				// there is no new items go back to the pooling
				continue
			}
			waitIndex = rMeta.LastIndex
			log.Debugf("Scheduling Wait index: %d", waitIndex)
			for _, key := range actions {
				id := path.Base(key)
				// Handle unregistration
				kvp, _, err := sc.cc.KV().Get(path.Join(key, ".unregisterFlag"), nil)
				if err != nil {
					handleError(err)
					continue
				}
				if kvp != nil && len(kvp.Value) > 0 && strings.ToLower(string(kvp.Value)) == "true" {
					log.Debugf("Scheduled action with id:%q has been requested to be stopped and unregistered", id)
					actionToStop, is := sc.actions[id]
					if is {
						// Stop the scheduled action
						actionToStop.stop()
						// Remove it from actions
						delete(sc.actions, id)
					}
					// Unregister it definitively
					sc.unregisterAction(id)
					continue
				}

				// Store the action if not already present and start it
				_, is := sc.actions[id]
				if !is {
					log.Debugf("start action id:%q", id)
					sca, err := sc.buildScheduledAction(id)
					if err != nil {
						handleError(err)
						continue
					}
					sc.actions[id] = sca
					sca.start()
				}
			}
		}
	}()
}

func (sc *scheduler) stopScheduling() {
	if defaultScheduler.isActive {
		log.Debugf("Scheduling service is about to be stopped")
		close(defaultScheduler.chStopScheduling)
		defaultScheduler.isActiveLock.Lock()
		defaultScheduler.isActive = false
		defaultScheduler.isActiveLock.Unlock()

		// Stop all running actions
		for _, action := range defaultScheduler.actions {
			action.stop()
		}
	}
}

func (sc *scheduler) buildScheduledAction(id string) (*scheduledAction, error) {
	sca := &scheduledAction{}
	sca.kv = sc.cc.KV()
	sca.ID = id

	actionPrefix := path.Join(consulutil.SchedulingKVPrefix, "actions", id)
	kvp, _, err := sc.cc.KV().Get(path.Join(actionPrefix, "deploymentID"), nil)
	if err != nil {
		return nil, err
	}
	if kvp != nil {
		sca.deploymentID = string(kvp.Value)
	}
	kvp, _, err = sc.cc.KV().Get(path.Join(actionPrefix, "type"), nil)
	if err != nil {
		return nil, err
	}
	if kvp != nil {
		sca.ActionType = string(kvp.Value)
	}
	kvp, _, err = sc.cc.KV().Get(path.Join(actionPrefix, "interval"), nil)
	if err != nil {
		return nil, err
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return nil, errors.Errorf("Missing interval for action: %q", id)
	}
	d, err := time.ParseDuration(string(kvp.Value))
	if err != nil {
		return nil, err
	}
	sca.timeInterval = d
	kvp, _, err = sc.cc.KV().Get(path.Join(actionPrefix, "async_op"), nil)
	if err != nil {
		return nil, err
	}
	if kvp != nil && len(kvp.Value) > 0 {
		sca.asyncOperationString = string(kvp.Value)
		err = json.Unmarshal(kvp.Value, &sca.AsyncOperation)
		if err != nil {
			return nil, err
		}
	}
	kvp, _, err = sc.cc.KV().Get(path.Join(actionPrefix, "latestTaskID"), nil)
	if err != nil {
		return nil, err
	}
	if kvp != nil && len(kvp.Value) > 0 {
		sca.latestTaskID = string(kvp.Value)
	}

	kvps, _, err := sc.cc.KV().List(path.Join(consulutil.SchedulingKVPrefix, "actions", id, "data"), nil)
	if err != nil {
		return nil, err
	}
	sca.Data = make(map[string]string, len(kvps))

	for _, kvp := range kvps {
		key := path.Base(kvp.Key)
		if len(kvp.Value) > 0 {
			sca.Data[key] = string(kvp.Value)
		}
	}
	return sca, nil
}
