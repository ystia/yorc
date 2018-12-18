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
	"encoding/json"
	"math/rand"
	"strings"
	"time"

	"github.com/moby/moby/client"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/tosca"
)

type defaultExecutor struct {
	r   *rand.Rand
	cli *client.Client
}

func newExecutor() *defaultExecutor {
	cli, err := client.NewEnvClient()
	if err != nil {
		err = errors.Wrap(err, "failed to create docker execution client, docker sandboxing for operation hosted on orchestrator is disabled")
		log.Printf("%v", err)
	}
	return &defaultExecutor{r: rand.New(rand.NewSource(time.Now().UnixNano())), cli: cli}
}

func (e *defaultExecutor) ExecAsyncOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation, stepName string) (*prov.Action, time.Duration, error) {
	if strings.ToLower(operation.Name) != strings.ToLower(tosca.RunnableRunOperationName) {
		return nil, 0, errors.Errorf("%q operation is not supported by the Ansible asynchronous executor only %q is.", operation.Name, tosca.RunnableRunOperationName)
	}

	jsonOp, err := json.Marshal(operation)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to marshal asynchronous operation")
	}
	// Fill all used data for job monitoring
	data := make(map[string]string)
	data["originalTaskID"] = taskID
	data["nodeName"] = nodeName
	data["operation"] = string(jsonOp)
	// TODO deal with outputs?
	// data["outputs"] = strings.Join(e.jobInfo.outputs, ",")
	checkPeriod := conf.Ansible.JobsChecksPeriod
	if checkPeriod <= 0 {
		checkPeriod = 15 * time.Second
		log.Debugf("\"job_monitoring_time_interval\" configuration parameter is missing in Ansible configuration. Using default %s.", checkPeriod)
	}

	return &prov.Action{ActionType: "ansible-job-monitoring", Data: data}, checkPeriod, nil

}

func (e *defaultExecutor) ExecOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) error {
	consulClient, err := conf.GetConsulClient()
	if err != nil {
		return err
	}
	kv := consulClient.KV()

	exec, err := newExecution(ctx, kv, conf, taskID, deploymentID, nodeName, operation, e.cli)
	if err != nil {
		if IsOperationNotImplemented(err) {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).Registerf("Voluntary bypassing error: %s", err.Error())
			return nil
		}
		return err
	}

	instances, err := tasks.GetInstances(kv, taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}
	logForAllInstances(ctx, deploymentID, instances, events.LogLevelINFO, "Start the ansible execution of: %s with operation : %s", nodeName, operation.Name)

	// Execute operation
	err = exec.execute(ctx, conf.Ansible.ConnectionRetries != 0)
	if err == nil {
		return nil
	}
	if !IsRetriable(err) {
		logForAllInstances(ctx, deploymentID, instances, events.LogLevelERROR, "Ansible execution for operation %q on node %q failed", operation.Name, nodeName)
		return err
	}

	// Retry operation if error is retriable and AnsibleConnectionRetries > 0
	log.Debugf("Ansible Connection Retries:%d", conf.Ansible.ConnectionRetries)
	if conf.Ansible.ConnectionRetries > 0 {
		for i := 0; i < conf.Ansible.ConnectionRetries; i++ {
			logForAllInstances(ctx, deploymentID, instances, events.LogLevelWARN, "Caught a retriable error from Ansible: '%v'. Let's retry in few seconds (%d/%d)", err, i+1, conf.Ansible.ConnectionRetries)
			time.Sleep(time.Duration(e.r.Int63n(10)) * time.Second)
			err = exec.execute(ctx, i != 0)
			if err == nil {
				return nil
			}
			if !IsRetriable(err) {
				logForAllInstances(ctx, deploymentID, instances, events.LogLevelERROR, "Ansible execution for operation %q on node %q failed", operation.Name, nodeName)
				return err
			}
		}
		logForAllInstances(ctx, deploymentID, instances, events.LogLevelERROR, "Giving up retries for Ansible error: '%v' (%d/%d)", err, conf.Ansible.ConnectionRetries, conf.Ansible.ConnectionRetries)
	}
	logForAllInstances(ctx, deploymentID, instances, events.LogLevelERROR, "Ansible execution for operation %q on node %q failed", operation.Name, nodeName)
	return err
}

func logForAllInstances(ctx context.Context, deploymentID string, instances []string, level events.LogLevel, msg string, args ...interface{}) {
	for _, instanceID := range instances {
		events.WithContextOptionalFields(events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.InstanceID: instanceID})).NewLogEntry(level, deploymentID).Registerf(msg, args...)
	}
}
