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

package consulutil

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/log"
	"time"
)

var commonCheckID = "yorc-common-check"

// GetSession allows to return the session ID for a defined service leader key
func GetSession(cc *api.Client, serviceKey, agentName string) (string, error) {
	sessions, _, err := cc.Session().List(nil)
	if err != nil {
		return "", errors.Wrap(err, "Failed to list Consul sessions")
	}
	for _, session := range sessions {
		if session.Name == serviceKey && session.Node == agentName {
			log.Debugf("session = %+v", session)
			return session.ID, nil
		}
	}

	// Create the session if not exists
	log.Printf("Creating session with key:%q for leader election", serviceKey)
	var sessionChecks []string
	// Default serf Health check
	sessionChecks = append(sessionChecks, "serfHealth")
	sessionChecks = append(sessionChecks, commonCheckID)

	sessionEntry := &api.SessionEntry{
		Name:      serviceKey,
		Behavior:  "release",
		LockDelay: 0 * time.Nanosecond,
		Checks:    sessionChecks,
	}

	// Creating session can fail if check states are Critical : retry 3 times
	totalTries := 3
	log.Debugf("Try to create a session with key:%q", serviceKey)
	for try := 1; try <= totalTries; try++ {
		session, _, err := cc.Session().Create(sessionEntry, nil)
		if err == nil {
			log.Debugf("session created with id:%q", session)
			return session, nil
		}

		if try < totalTries {
			log.Printf("Failed to create session with key %q (try n°%d). %d another tries will be done", serviceKey, try, totalTries-try)
		}
		time.Sleep(2 * time.Second)
	}
	return "", errors.Wrapf(err, "Failed to create session with key:%q", serviceKey)
}

// GetAgentName allows to return the local consul agent name
func GetAgentName(cc *api.Client) (string, error) {
	agent, err := cc.Agent().Self()
	if err != nil {
		return "", errors.Wrap(err, "Failed to get agent name")
	}
	return agent["Config"]["NodeName"].(string), nil
}

func handleError(err error, serviceKey string) {
	err = errors.Wrapf(err, "[WARN] Error during watching leader election for service:%q", serviceKey)
	log.Print(err)
	log.Debugf("%+v", err)
}

// RegisterCommonCheck allows to register common checks for yorc. It must be used for all leader elections
func RegisterCommonCheck(cfg config.Configuration, cc *api.Client) error {
	checks, err := cc.Agent().Checks()
	if err != nil {
		return err
	}

	_, exist := checks[commonCheckID]
	log.Debugf("Check with id:%q already exists", commonCheckID)
	if !exist {
		log.Debugf("Creating check with id:%q", commonCheckID)
		check := &api.AgentCheckRegistration{
			ID:   commonCheckID,
			Name: "Common TCP check for yorc",
			AgentServiceCheck: api.AgentServiceCheck{
				Interval: "5s",
				TCP:      fmt.Sprintf("localhost:%d", cfg.HTTPPort),
				Status:   "passing",
			},
		}
		return cc.Agent().CheckRegister(check)
	}
	return nil
}

// IsAnyLeader allows to return true if any leader exists for a defined service key
func IsAnyLeader(cc *api.Client, serviceKey string, waitIndex uint64) (bool, string, uint64, error) {
	q := &api.QueryOptions{WaitIndex: waitIndex}
	kvPair, rMeta, err := cc.KV().Get(serviceKey, q)
	if err != nil {
		return false, "", 0, err
	}

	if kvPair == nil || kvPair.Session == "" || len(kvPair.Value) == 0 {
		log.Debugf("Session has been invalidated")
		return false, "", rMeta.LastIndex, nil
	}

	log.Debugf("Current leader:%q for service:%q", kvPair.Value, serviceKey)
	log.Debugf("Leader session:%q for service :%q", kvPair.Session, serviceKey)
	return true, string(kvPair.Value), rMeta.LastIndex, nil
}

// WatchLeaderElection allows to watch for leader election for defined service. It elects leader if needed and can start the related service or stop it if node is no longer leader
func WatchLeaderElection(cfg config.Configuration, cc *api.Client, serviceKey string, chStop chan struct{}, leaderServiceStart func(), leaderServiceStop func()) {
	log.Debugf("WatchLeaderElection for service:%q", serviceKey)
	var (
		waitIndex uint64
		lastIndex uint64
		isAny     bool
		leader    string
	)
	agentName, err := GetAgentName(cc)
	if err != nil {
		handleError(err, serviceKey)
		return
	}

	// Register common check used for controlling all services and leader elections
	if err := RegisterCommonCheck(cfg, cc); err != nil {
		log.Printf("Failed to register checks for leader election due to error:%+v", err)
	}

	for {
		select {
		case <-chStop:
			log.Printf("Stop watching leader election for service:%q", serviceKey)
			return
		default:
		}

		isAny, leader, lastIndex, err = IsAnyLeader(cc, serviceKey, waitIndex)
		log.Debugf("Wait Index is %d for watching leader election for service:%q", lastIndex, serviceKey)
		if err != nil {
			handleError(err, serviceKey)
			continue
		}
		if waitIndex == lastIndex {
			continue
		}
		waitIndex = lastIndex
		if !isAny {
			// No leader has been elected : try to acquire leadership on service
			session, err := GetSession(cc, serviceKey, agentName)
			if err != nil {
				handleError(err, serviceKey)
				continue
			}
			kvPair := &api.KVPair{
				Key:     serviceKey,
				Value:   []byte(agentName),
				Session: session,
			}

			// Because of default lock delay = 15s issue, we temporize before trying again to acquire the lock.
			totalTries := 3
			for try := 1; try <= totalTries; try++ {
				log.Debugf("Try to acquire a lock on key:%q", serviceKey)

				acquired, _, err := cc.KV().Acquire(kvPair, nil)
				if err != nil {
					handleError(err, serviceKey)
					continue
				}
				if acquired {
					log.Printf("I am now leader for service:%q as %q", serviceKey, agentName)
					// Proceed to leader concern execution if needed
					if leaderServiceStart != nil {
						leaderServiceStart()
					}
					break
				} else {
					if try < totalTries {
						log.Printf("Failed to acquire service leadership for service %q (try n°%d). %d another tries will be done", serviceKey, try, totalTries-try)
					} else {
						log.Printf("Failed to acquire service leadership (%d tries). Another node did it for service:%q", totalTries, serviceKey)
					}

				}
				time.Sleep(15 * time.Second)
			}
		} else {
			if leader == agentName {
				log.Printf("I am still the leader for service:%q as %q", serviceKey, agentName)
				// Proceed to leader concern execution if needed
				if leaderServiceStart != nil {
					leaderServiceStart()
				}
			} else {
				log.Debugf("leader is %q", leader)
				if leaderServiceStop != nil {
					leaderServiceStop()
				}
			}
		}
	}
}
