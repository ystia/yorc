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
	"github.com/ystia/yorc/log"
	"strings"
	"time"
)

const yorcServiceCheck = "service:yorc"

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
	sessionEntry := &api.SessionEntry{
		Name:     serviceKey,
		Behavior: "release",
		// Consul Issue : LockDelay = 0 is not allowed by API. https://github.com/hashicorp/consul/issues/1077
		LockDelay: 1 * time.Nanosecond,
		// Default serf Health check + yorc service check
		Checks: []string{"serfHealth", yorcServiceCheck},
	}

	// Retry 3 times to create a session if node check is in critical state after a restart
	retries := 3
	for i := 0; i <= retries; i++ {
		session, _, err := cc.Session().Create(sessionEntry, nil)
		if err == nil {
			return session, nil
		}
		if isNodeInCriticalState(err) && i <= retries {
			log.Printf("Failed to create session with key %q because node check is still in critical state. %d another tries will be done in a few time", serviceKey, retries-i)
		} else {
			return "", errors.Wrapf(err, "Failed to create session with key:%q", serviceKey)
		}
		time.Sleep(5 * time.Second)
	}

	return "", errors.Wrapf(err, "Failed to create session with key:%q", serviceKey)
}

func isNodeInCriticalState(err error) bool {
	mess := fmt.Sprintf("Check '%s' is in critical state", yorcServiceCheck)
	return strings.Contains(err.Error(), mess)
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

// IsAnyLeader allows to return true if any leader exists for a defined service key
func IsAnyLeader(cc *api.Client, serviceKey string, waitIndex uint64) (bool, string, uint64, error) {
	q := &api.QueryOptions{WaitIndex: waitIndex}
	kvPair, rMeta, err := cc.KV().Get(serviceKey, q)
	if err != nil {
		return false, "", 0, err
	}

	if kvPair == nil || kvPair.Session == "" || len(kvPair.Value) == 0 {
		log.Debugf("No leader has been found for service:%q", serviceKey)
		return false, "", rMeta.LastIndex, nil
	}
	return true, string(kvPair.Value), rMeta.LastIndex, nil
}

// WatchLeaderElection allows to watch for leader election for defined service. It elects leader if needed and can start the related service or stop it if node is no longer leader
func WatchLeaderElection(cc *api.Client, serviceKey string, chStop chan struct{}, leaderServiceStart func(), leaderServiceStop func()) {
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

			// Try to acquire the service leadership
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
			} else {
				log.Printf("Failed to acquire service leadership. Another node did it for service:%q", serviceKey)
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
