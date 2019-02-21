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
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/events"
	"github.com/ystia/yorc/v3/helper/sshutil"
	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/prov"
)

type actionOperator struct {
}

type actionData struct {
	stepName            string
	jobID               string
	taskID              string
	remoteBaseDirectory string
	remoteExecDirectory string
	outputs             []string
}

func (o *actionOperator) ExecAction(ctx context.Context, cfg config.Configuration, taskID, deploymentID string, action *prov.Action) (bool, error) {
	log.Debugf("Execute Action with ID:%q, taskID:%q, deploymentID:%q", action.ID, taskID, deploymentID)

	if action.ActionType == "job-monitoring" {
		deregister, err := o.monitorJob(ctx, cfg, deploymentID, action)
		if err != nil {
			// action scheduling needs to be unregistered
			return true, err
		}

		return deregister, nil
	}
	return true, errors.Errorf("Unsupported actionType %q", action.ActionType)
}

func (o *actionOperator) monitorJob(ctx context.Context, cfg config.Configuration, deploymentID string, action *prov.Action) (bool, error) {
	var (
		err error
		ok  bool
	)

	actionData := &actionData{}
	// Check jobID
	actionData.jobID, ok = action.Data["jobID"]
	if !ok {
		return true, errors.Errorf("Missing mandatory information jobID for actionType:%q", action.ActionType)
	}
	// Check stepName
	actionData.stepName, ok = action.Data["stepName"]
	if !ok {
		return true, errors.Errorf("Missing mandatory information stepName for actionType:%q", action.ActionType)
	}
	// Check remoteBaseDirectory
	actionData.remoteBaseDirectory, ok = action.Data["remoteBaseDirectory"]
	if !ok {
		return true, errors.Errorf("Missing mandatory information remoteBaseDirectory for actionType:%q", action.ActionType)
	}
	// Check taskID
	actionData.taskID, ok = action.Data["taskID"]
	if !ok {
		return true, errors.Errorf("Missing mandatory information taskID for actionType:%q", action.ActionType)
	}
	// Check outputs
	outputStr, ok := action.Data["outputs"]
	if !ok {
		return true, errors.Errorf("Missing mandatory information outputs for actionType:%q", action.ActionType)
	}
	actionData.outputs = strings.Split(outputStr, ",")
	actionData.remoteExecDirectory = action.Data["remoteExecDirectory"]

	// Get a sshClient to connect to slurm client node, and execute slurm commands such as squeue, or system commands such as cp, mv, mkdir, etc.
	sshClient, err := getSSHClient(action.Data["userName"], action.Data["privateKey"], action.Data["password"], cfg)
	if err != nil {
		return true, err
	}

	info, err := getJobInfo(sshClient, actionData.jobID, "")
	if err != nil {
		_, done := err.(*noJobFound)
		if done {
			err = o.endJob(ctx, deploymentID, actionData, sshClient)
			return true, err
		}
		return true, errors.Wrapf(err, "failed to get job info with jobID:%q", actionData.jobID)
	}

	mess := fmt.Sprintf("Job Name:%s, ID:%s, State:%s, Reason:%s, Execution Time:%s", info.name, info.ID, info.state, info.reason, info.time)
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(mess)
	o.displayTempOutput(ctx, deploymentID, actionData, sshClient)
	return false, nil
}

func (o *actionOperator) endJob(ctx context.Context, deploymentID string, actionData *actionData, sshClient *sshutil.SSHClient) error {
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(fmt.Sprintf("Job with JobID:%s is DONE", actionData.jobID))
	err := o.endJobOutput(ctx, deploymentID, actionData, sshClient)
	if err != nil {
		return errors.Wrapf(err, "failed to handle job outputs with jobID:%q", actionData.jobID)
	}
	o.cleanUp(actionData, sshClient)
	return nil
}

func (o *actionOperator) endJobOutput(ctx context.Context, deploymentID string, actionData *actionData, sshClient *sshutil.SSHClient) error {
	// Look for outputs with relative path
	relOutputs := make([]string, 0)
	for _, output := range actionData.outputs {
		if !path.IsAbs(output) {
			relOutputs = append(relOutputs, output)
		} else {
			o.logFile(ctx, deploymentID, output, sshClient, false)
		}
	}

	if len(relOutputs) > 0 {
		// Copy the outputs with relative path in <JOB_ID>_outputs directory at root level
		outputDir := fmt.Sprintf("job_" + actionData.jobID + "_outputs")
		cmd := fmt.Sprintf("mkdir %s", outputDir)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).RegisterAsString(fmt.Sprintf("Run the command: %q", cmd))
		output, err := sshClient.RunCommand(cmd)
		if err != nil {
			return errors.Wrap(err, output)
		}
		for _, relOutput := range relOutputs {
			oldPath := path.Join(actionData.remoteExecDirectory, relOutput)
			newPath := path.Join(outputDir, relOutput)
			// Copy the file in the output dir
			cmd := fmt.Sprintf("cp -f %s %s", oldPath, newPath)
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).RegisterAsString(fmt.Sprintf("Run the command: %q", cmd))
			output, err := sshClient.RunCommand(cmd)
			if err != nil {
				return errors.Wrap(err, output)
			}
			err = o.logFile(ctx, deploymentID, newPath, sshClient, false)
			if err != nil {
				return err
			}
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(fmt.Sprintf("outputs are available in folder:%q in %s's home directory", outputDir, sshClient.Config.User))
	}
	return nil
}

func (o *actionOperator) displayTempOutput(ctx context.Context, deploymentID string, actionData *actionData, sshClient *sshutil.SSHClient) {
	for _, output := range actionData.outputs {
		var tempFile string
		if path.IsAbs(output) {
			tempFile = output
		} else {
			tempFile = path.Join(actionData.remoteExecDirectory, output)
		}
		// log file ignoring errors as outputs can not yet be created if job is pending
		o.logFile(ctx, deploymentID, tempFile, sshClient, true)
	}
}

func (o *actionOperator) cleanUp(actionData *actionData, sshClient *sshutil.SSHClient) {
	log.Debugf("Cleanup the operation remote base directory")
	cmd := fmt.Sprintf("rm -rf %s", actionData.remoteBaseDirectory)
	_, err := sshClient.RunCommand(cmd)
	if err != nil {
		log.Printf("an error:%+v occurred during cleanup for remote base directory:%q", err, actionData.remoteBaseDirectory)
	}
}

func (o *actionOperator) logFile(ctx context.Context, deploymentID, filePath string, sshClient *sshutil.SSHClient, ignoreErrors bool) error {
	var err error
	cmd := fmt.Sprintf("cat %s", filePath)
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).RegisterAsString(fmt.Sprintf("Run the command: %q", cmd))
	output, err := sshClient.RunCommand(cmd)
	if !ignoreErrors && err != nil {
		log.Debugf("an error:%+v occurred during logging file:%q", err, filePath)
		return errors.Wrapf(err, "failed to log file:%q", filePath)
	}
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString("\n" + output)
	return nil
}
