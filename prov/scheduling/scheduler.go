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
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
	"github.com/ystia/yorc/tasks/collector"
	"path"
	"strings"
	"sync"
	"time"
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
	actionsLock      sync.RWMutex
}

// RegisterAction allows to register a scheduled action and to start scheduling it
func RegisterAction(deploymentID string, timeInterval time.Duration, action *prov.Action) error {
	log.Debugf("Action:%+v has been requested to be registered for scheduling with [deploymentID:%q, timeInterval:%q]", action, deploymentID, timeInterval.String())
	id := uuid.NewV4().String()

	// Check mandatory parameters
	if deploymentID == "" {
		return errors.New("deploymentID is mandatory parameter to register scheduled action")
	}
	if action == nil || action.ActionType == "" {
		return errors.New("actionType is mandatory parameter to register scheduled action")
	}
	scaPath := path.Join(consulutil.SchedulingKVPrefix, "actions", id)
	scaOps := api.KVTxnOps{
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(scaPath, "deploymentID"),
			Value: []byte(deploymentID),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(scaPath, "type"),
			Value: []byte(action.ActionType),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(scaPath, "interval"),
			Value: []byte(timeInterval.String()),
		},
	}

	if action.Data != nil {
		for k, v := range action.Data {
			scaOps = append(scaOps, &api.KVTxnOp{
				Verb:  api.KVSet,
				Key:   path.Join(scaPath, k),
				Value: []byte(v),
			})
		}
	}

	ok, response, _, err := defaultScheduler.cc.KV().Txn(scaOps, nil)
	if err != nil {
		return errors.Wrapf(err, "Failed to register scheduled action for deploymentID:%q, type:%q, id:%q", deploymentID, action.ActionType, id)
	}
	if !ok {
		// Check the response
		errs := make([]string, 0)
		for _, e := range response.Errors {
			errs = append(errs, e.What)
		}
		return errors.Errorf("Failed to register scheduled action for deploymentID:%q, type:%q, id:%q due to:%s", deploymentID, action.ActionType, id, strings.Join(errs, ", "))
	}
	return nil
}

// UnregisterAction allows to unregister a scheduled action and to stop scheduling it
func UnregisterAction(id string) error {
	return defaultScheduler.flagForRemoval(id)
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

// flagForRemoval allows to flag a scheduled action for removal
func (sc *scheduler) flagForRemoval(id string) error {
	log.Debugf("Flag for removal scheduled action with id:%q", id)
	scaPath := path.Join(consulutil.SchedulingKVPrefix, "actions", id)
	kvp := &api.KVPair{Key: path.Join(scaPath, ".unregisterFlag"), Value: []byte("true")}
	_, err := sc.cc.KV().Put(kvp, nil)
	return errors.Wrap(err, "Failed to flag scheduled action for removal")
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

func (sc scheduler) startScheduling() {
	if sc.isActive {
		log.Println("Scheduling service is already running.")
		return
	}
	log.Debugf("Scheduling service is now running.")

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
						sc.actionsLock.Lock()
						delete(sc.actions, id)
						sc.actionsLock.Unlock()
						// Unregister it definitively
						sc.unregisterAction(id)
					}
					continue
				}
				sca, err := sc.buildScheduledAction(id)
				if err != nil {
					handleError(err)
					continue
				}
				// Store the action if not already present and start it
				sc.actionsLock.RLock()
				for k, v := range sc.actions {
					log.Debugf("actions k:%q, v:%+v", k, v)
				}
				_, is := sc.actions[id]
				sc.actionsLock.RUnlock()
				if !is {
					log.Debugf("start action id:%q", id)
					sc.actionsLock.Lock()
					sc.actions[id] = sca
					sc.actionsLock.Unlock()
					sca.start()
				}
			}
		}
	}()
}

func (sc scheduler) stopScheduling() {
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

func (sc scheduler) buildScheduledAction(id string) (*scheduledAction, error) {
	kvps, _, err := sc.cc.KV().List(path.Join(consulutil.SchedulingKVPrefix, "actions", id), nil)
	if err != nil {
		return nil, err
	}
	sca := &scheduledAction{}
	sca.ID = id
	sca.Data = make(map[string]string)
	for _, kvp := range kvps {
		key := path.Base(kvp.Key)
		switch key {
		case "deploymentID":
			if kvp != nil && len(kvp.Value) > 0 {
				sca.deploymentID = string(kvp.Value)
			}
		case "type":
			if kvp != nil && len(kvp.Value) > 0 {
				sca.ActionType = string(kvp.Value)
			}
		case "interval":
			if kvp != nil && len(kvp.Value) > 0 {
				d, err := time.ParseDuration(string(kvp.Value))
				if err != nil {
					return nil, err
				}
				sca.timeInterval = d
			}
		default:
			if kvp != nil && len(kvp.Value) > 0 {
				sca.Data[key] = string(kvp.Value)
			}
		}
	}
	return sca, nil
}
