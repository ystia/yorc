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

package slurm

import (
	"context"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/sshutil"
	"github.com/ystia/yorc/helper/stringutil"
	"github.com/ystia/yorc/prov"
	"github.com/ystia/yorc/tasks"
)

type actionOperator struct {
	client *sshutil.SSHClient
	kv     *api.KV
	action prov.Action
}

func (o actionOperator) ExecAction(ctx context.Context, cfg config.Configuration, action prov.Action, deploymentID string) error {
	var err error
	o.client, err = GetSSHClient(cfg)
	if err != nil {
		return err
	}
	consulClient, err := cfg.GetConsulClient()
	if err != nil {
		return err
	}
	o.kv = consulClient.KV()
	o.action = action
	switch action.ActionType {
	case "job-monitoring":
		return o.monitorJob(ctx, deploymentID)
	default:
		return errors.Errorf("Unsupported actionType %q", action.ActionType)
	}
	return nil
}

func (o actionOperator) monitorJob(ctx context.Context, deploymentID string) error {
	// Check jobID as mandatory parameter
	jobID, ok := o.action.Data["jobID"]
	if !ok {
		return errors.Errorf("Missing mandatory information jobID for actionType:%q", o.action.ActionType)
	}

	// Fill log optional fields for log registration
	var wfName string
	taskID, ok := o.action.Data["taskID"]
	if ok {
		wfName, _ = tasks.GetTaskData(o.kv, taskID, "workflowName")
	}
	logOptFields := events.LogOptionalFields{
		events.WorkFlowID:    wfName,
		events.NodeID:        o.action.Data["nodeName"],
		events.OperationName: stringutil.GetLastElement(o.action.Data["operationName"], "."),
		events.InterfaceName: stringutil.GetAllExceptLastElement(o.action.Data["operationName"], "."),
	}
	ctx = events.NewContext(ctx, logOptFields)

	info, err := getJobInfo(o.client, jobID, "")
	if err != nil {
		_, jobIsDone := err.(jobIsDone)
		if jobIsDone {
			// If batch job, cleanup needs to be processed
			//e.endBatchExecution(ctx, stopCh)
			// action scheduling needs to be unregistered
			// job running step must be set to done and workflow must be resumed

		} else {
			return errors.Wrapf(err, "failed to get job info with jobID:%q", jobID)
		}
	}

	mess := fmt.Sprintf("Job Name:%s, Job ID:%s, Job State:%s", info.name, info.ID, info.state)
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(mess)
	return nil
}
