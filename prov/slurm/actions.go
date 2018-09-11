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
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
	"github.com/ystia/yorc/prov/scheduling"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/tasks/collector"
	"path"
	"strconv"
	"strings"
)

type actionOperator struct {
	client       *sshutil.SSHClient
	consulClient *api.Client
	action       *prov.Action
}

func (o actionOperator) ExecAction(ctx context.Context, cfg config.Configuration, taskID, deploymentID string, action *prov.Action) error {
	log.Debugf("Execute Action:%+v with taskID:%q, deploymentID:%q", action, taskID, deploymentID)
	var err error
	o.client, err = GetSSHClient(cfg)
	if err != nil {
		return err
	}
	o.consulClient, err = cfg.GetConsulClient()
	if err != nil {
		return err
	}
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
	// Check stepName as mandatory parameter
	stepName, ok := o.action.Data["stepName"]
	if !ok {
		return errors.Errorf("Missing mandatory information stepName for actionType:%q", o.action.ActionType)
	}
	// Check isBatch as mandatory parameter
	isBatchStr, ok := o.action.Data["isBatch"]
	if !ok {
		return errors.Errorf("Missing mandatory information isBatch for actionType:%q", o.action.ActionType)
	}
	isBatch, err := strconv.ParseBool(isBatchStr)
	if err != nil {
		return errors.Errorf("Invalid information isBatch for actionType:%q", o.action.ActionType)
	}

	// Check remoteBaseDirectory as mandatory parameter
	remoteBaseDirectory, ok := o.action.Data["remoteBaseDirectory"]
	if !ok {
		return errors.Errorf("Missing mandatory information remoteBaseDirectory for actionType:%q", o.action.ActionType)
	}
	// Check remoteExecDirectory as mandatory parameter
	remoteExecDirectory, ok := o.action.Data["remoteExecDirectory"]
	if !ok {
		return errors.Errorf("Missing mandatory information remoteExecDirectory for actionType:%q", o.action.ActionType)
	}

	// Fill log optional fields for log registration
	var wfName string
	taskID, ok := o.action.Data["taskID"]
	if ok {
		wfName, _ = tasks.GetTaskData(o.consulClient.KV(), taskID, "workflowName")
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
		_, done := err.(*jobIsDone)
		if done {
			defer func() {
				// action scheduling needs to be unregistered
				err = scheduling.UnregisterAction(o.action.ID)
				if err != nil {
					log.Printf("failed to unregister job Monitoring job info with actionID:%q, jobID:%q due to error:%+v", o.action.ID, jobID, err)
				}
			}()
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(fmt.Sprintf("Job with JobID:%s is DONE", jobID))
			// If batch job, cleanup needs to be processed after logging output files
			if isBatch {
				outputs, _ := o.action.Data["outputs"]
				outputsSlice := strings.Split(outputs, ",")
				err = o.handleBatchOutputs(ctx, deploymentID, jobID, remoteExecDirectory, outputsSlice)
				if err != nil {
					return errors.Wrapf(err, "failed to handle batch outputs with jobID:%q", jobID)
				}
				o.cleanUp(remoteBaseDirectory)
			}

			// job running step must be set to done and workflow must be resumed
			step := &tasks.TaskStep{Status: tasks.TaskStepStatusDONE.String(), Name: stepName}
			err = tasks.UpdateTaskStepStatus(o.consulClient.KV(), taskID, step)
			if err != nil {
				return errors.Wrapf(err, "failed to update step status to DONE for taskID:%q, stepName:%q", taskID, stepName)
			}
			collector := collector.NewCollector(o.consulClient)
			err = collector.ResumeTask(taskID)
			if err != nil {
				return errors.Wrapf(err, "failed to resume task with taskID:%q", taskID)
			}
			return nil
		}
		return errors.Wrapf(err, "failed to get job info with jobID:%q", jobID)
	}

	mess := fmt.Sprintf("Job Name:%s, Job ID:%s, Job State:%s", info.name, info.ID, info.state)
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(mess)
	return nil
}

// Need to copy output files in specific folder "job_JOBID_outputs" but only for output with relative path
func (o *actionOperator) handleBatchOutputs(ctx context.Context, deploymentID, jobID, execDirectory string, outputs []string) error {
	// If no output file is specified, default one is slurm-JOBID.out and needs to be copied
	if len(outputs) == 0 {
		outputs = []string{fmt.Sprintf("slurm-%s.out", jobID)}
	}

	// Look for outputs with relative path
	relOutputs := make([]string, 0)
	for _, output := range outputs {
		if !path.IsAbs(output) {
			relOutputs = append(relOutputs, output)
		} else {
			o.logFile(ctx, deploymentID, output)
		}
	}

	if len(relOutputs) > 0 {
		// Copy the outputs with relative path in <JOB_ID>_outputs directory at root level
		outputDir := fmt.Sprintf("job_" + jobID + "_outputs")
		cmd := fmt.Sprintf("mkdir %s", outputDir)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(fmt.Sprintf("Run the command: %q", cmd))
		output, err := o.client.RunCommand(cmd)
		if err != nil {
			return errors.Wrap(err, output)
		}
		for _, output := range relOutputs {
			oldPath := path.Join(execDirectory, output)
			newPath := path.Join(outputDir, output)
			// Copy the file in the output dir
			cmd := fmt.Sprintf("cp -f %s %s", oldPath, newPath)
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(fmt.Sprintf("Run the command: %q", cmd))
			output, err := o.client.RunCommand(cmd)
			if err != nil {
				return errors.Wrap(err, output)
			}
			o.logFile(ctx, deploymentID, newPath)
		}
	}
	return nil
}

func (o *actionOperator) cleanUp(remoteBaseDirectory string) {
	log.Debugf("Cleanup the operation remote base directory")
	cmd := fmt.Sprintf("rm -rf %s", remoteBaseDirectory)
	_, err := o.client.RunCommand(cmd)
	if err != nil {
		log.Printf("an error:%+v occurred during cleanup for remote base directory:%q", err, remoteBaseDirectory)
	}
}

func (o *actionOperator) logFile(ctx context.Context, deploymentID, filePath string) {
	// Log the output
	cmd := fmt.Sprintf("cat %s", filePath)
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(fmt.Sprintf("Run the command: %q", cmd))
	output, err := o.client.RunCommand(cmd)
	if err != nil {
		log.Printf("an error:%+v occurred during logging file:%q", err, filePath)
	}
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString("\n" + output)
}
