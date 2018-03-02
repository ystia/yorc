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

package ansible

import (
	"context"
	"math/rand"
	"time"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
)

type defaultExecutor struct {
	r *rand.Rand
}

// NewExecutor returns an Executor
func NewExecutor() prov.OperationExecutor {
	return &defaultExecutor{r: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func (e *defaultExecutor) ExecOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) error {
	consulClient, err := conf.GetConsulClient()
	if err != nil {
		return err
	}
	kv := consulClient.KV()
	exec, err := newExecution(kv, conf, taskID, deploymentID, nodeName, operation)
	if err != nil {
		if IsOperationNotImplemented(err) {
			log.Debugf("Voluntary bypassing error: %s.", err.Error())
			return nil
		}
		return err
	}

	// Execute operation
	err = exec.execute(ctx, conf.Ansible.ConnectionRetries != 0)
	if err == nil {
		return nil
	}
	if !IsRetriable(err) {
		return err
	}

	// Retry operation if error is retriable and AnsibleConnectionRetries > 0
	log.Debugf("Ansible Connection Retries:%d", conf.Ansible.ConnectionRetries)
	if conf.Ansible.ConnectionRetries > 0 {
		for i := 0; i < conf.Ansible.ConnectionRetries; i++ {
			log.Printf("Deployment: %q, Node: %q, Operation: %s: Caught a retriable error from Ansible: '%s'. Let's retry in few seconds (%d/%d)", deploymentID, nodeName, operation, err, i+1, conf.Ansible.ConnectionRetries)
			time.Sleep(time.Duration(e.r.Int63n(10)) * time.Second)
			err = exec.execute(ctx, i != 0)
			if err == nil {
				return nil
			}
			if !IsRetriable(err) {
				return err
			}
		}

		log.Printf("Deployment: %q, Node: %q, Operation: %s: Giving up retries for Ansible error: '%s' (%d/%d)", deploymentID, nodeName, operation, err, conf.Ansible.ConnectionRetries, conf.Ansible.ConnectionRetries)
	}

	return err
}
