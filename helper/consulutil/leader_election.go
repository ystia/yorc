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
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/log"
)

// GetSession allows to return the session ID for a defined service leader key
func GetSession(cc *api.Client, serviceKey string) (string, error) {
	sessions, _, err := cc.Session().List(nil)
	if err != nil {
		return "", errors.Wrap(err, "Failed to list Consul sessions")
	}
	for _, session := range sessions {
		if session.Name == serviceKey {
			return session.ID, nil
		}
	}

	// Create the session if not exists
	log.Println("Creating session with key:%q for leader election", serviceKey)
	sessionEntry := &api.SessionEntry{Name: serviceKey}
	session, _, err := cc.Session().Create(sessionEntry, nil)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to create session with key:%q", serviceKey)
	}
	return session, nil
}

// IsAnyLeader allows to return true if any leader exists for a defined service key
func IsAnyLeader(cc *api.Client, serviceKey string, waitIndex uint64) (bool, string, uint64, error) {
	q := &api.QueryOptions{WaitIndex: waitIndex}
	kvPair, rMeta, err := cc.KV().Get(serviceKey, q)
	if err != nil {
		return false, "", 0, err
	}
	if kvPair == nil || kvPair.Session == "" || len(kvPair.Value) == 0 {
		return false, "", rMeta.LastIndex, nil
	}

	log.Debugf("Current leader is:%q for service:%q", kvPair.Value, serviceKey)
	log.Debugf("Current session is:%q for service :%q", kvPair.Session, serviceKey)
	return true, string(kvPair.Value), rMeta.LastIndex, nil
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

// WatchLeaderElection allows to watch for leader election for defined service. It elects leader if needed and can run the related concern function
func WatchLeaderElection(cc *api.Client, serviceKey string, chStop chan struct{}, leaderConcernFct func()) {
	log.Debugf("WatchLeaderElection for service:%q", serviceKey)
	var (
		waitIndex uint64
		isAny     bool
		leader    string
	)
	agentName, err := GetAgentName(cc)
	if err != nil {
		handleError(err, serviceKey)
	}

	for {
		select {
		case <-chStop:
			log.Printf("Stop watching leader election for service:%q", serviceKey)
			return
		default:
		}

		isAny, leader, waitIndex, err = IsAnyLeader(cc, serviceKey, waitIndex)
		if err != nil {
			handleError(err, serviceKey)
			continue
		}
		if !isAny {
			// No leader has been elected : try to acquire leadership on service
			session, err := GetSession(cc, serviceKey)
			if err != nil {
				handleError(err, serviceKey)
				continue
			}

			kvPair := &api.KVPair{
				Key:     serviceKey,
				Value:   []byte(agentName),
				Session: session,
			}
			acquired, _, err := cc.KV().Acquire(kvPair, nil)
			if acquired {
				log.Printf("I am leader for service:%q as %q", serviceKey, agentName)
				// Proceed to leader concern execution if needed
				if leaderConcernFct != nil {
					leaderConcernFct()
				}
			} else {
				log.Debugf("Failed to acquire service leadership. Someone else did it for service:%q", serviceKey)
			}
		} else if leader == agentName {
			log.Printf("I am still the leader for service:%q as %q", serviceKey, agentName)
			// Proceed to leader concern execution if needed
			if leaderConcernFct != nil {
				leaderConcernFct()
			}
		}
	}
}
