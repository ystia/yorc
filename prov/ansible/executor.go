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

	"github.com/moby/moby/client"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/stringutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
)

type defaultExecutor struct {
	r   *rand.Rand
	cli *client.Client
}

// NewExecutor returns an Executor
func NewExecutor() prov.OperationExecutor {
	cli, err := client.NewEnvClient()
	if err != nil {
		err = errors.Wrap(err, "failed to create docker execution client, docker sandboxing for operation hosted on orchestrator is disabled")
		log.Printf("%v", err)
	}

	return &defaultExecutor{r: rand.New(rand.NewSource(time.Now().UnixNano())), cli: cli}
}

func (e *defaultExecutor) ExecOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) error {
	consulClient, err := conf.GetConsulClient()
	if err != nil {
		return err
	}
	kv := consulClient.KV()

	logOptFields, ok := events.FromContext(ctx)
	if !ok {
		return errors.New("Missing context log fields")
	}
	logOptFields[events.NodeID] = nodeName
	logOptFields[events.ExecutionID] = taskID
	logOptFields[events.OperationName] = stringutil.GetLastElement(operation.Name, ".")
	logOptFields[events.InterfaceName] = stringutil.GetAllExceptLastElement(operation.Name, ".")
	ctx = events.NewContext(ctx, logOptFields)

	exec, err := newExecution(ctx, kv, conf, taskID, deploymentID, nodeName, operation, e.cli)
	if err != nil {
		if IsOperationNotImplemented(err) {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).Registerf("Voluntary bypassing error: %s", err.Error())
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
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).Registerf("Caught a retriable error from Ansible: '%v'. Let's retry in few seconds (%d/%d)", err, i+1, conf.Ansible.ConnectionRetries)
			time.Sleep(time.Duration(e.r.Int63n(10)) * time.Second)
			err = exec.execute(ctx, i != 0)
			if err == nil {
				return nil
			}
			if !IsRetriable(err) {
				return err
			}
		}

		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, deploymentID).Registerf("Giving up retries for Ansible error: '%v' (%d/%d)", err, conf.Ansible.ConnectionRetries, conf.Ansible.ConnectionRetries)

	}

	return err
}
