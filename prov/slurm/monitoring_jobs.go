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
	"github.com/pkg/errors"
	"path"
	"strings"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/events"
	"github.com/ystia/yorc/v3/helper/sshutil"
	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/prov"
)

type actionOperator struct {
}

type actionData struct {
	stepName   string
	jobID      string
	taskID     string
	workingDir string
	artifacts  []string
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
		err        error
		deregister bool
		ok         bool
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
	// Check workingDir
	actionData.workingDir, ok = action.Data["workingDir"]
	if !ok {
		return true, errors.Errorf("Missing mandatory information workingDir for actionType:%q", action.ActionType)
	}
	// Check taskID
	actionData.taskID, ok = action.Data["taskID"]
	if !ok {
		return true, errors.Errorf("Missing mandatory information taskID for actionType:%q", action.ActionType)
	}
	// Check artifacts (optional)
	artifactsStr, ok := action.Data["artifacts"]
	if ok {
		actionData.artifacts = strings.Split(artifactsStr, ",")
	}

	// Get a sshClient to connect to slurm client node, and execute slurm commands such as squeue, or system commands such as cp, mv, mkdir, etc.
	sshClient, err := getSSHClient(action.Data["userName"], action.Data["privateKey"], action.Data["password"], cfg)
	if err != nil {
		return true, err
	}

	info, err := getJobInfo(sshClient, actionData.jobID)
	if err != nil {
		return true, errors.Wrapf(err, "failed to get job info with jobID:%q", actionData.jobID)
	}

	var mess string
	if info["Reason"] != "None" {
		mess = fmt.Sprintf("Job Name:%s, ID:%s, State:%s, Reason:%s, Execution Time:%s", info["JobName"], info["JobId"], info["JobState"], info["Reason"], info["RunTime"])
	} else {
		mess = fmt.Sprintf("Job Name:%s, ID:%s, State:%s, Execution Time:%s", info["JobName"], info["JobId"], info["JobState"], info["RunTime"])
	}
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(mess)

	stdOut, existStdOut := info["StdOut"]
	stdErr, existStdErr := info["StdErr"]
	if existStdOut && existStdErr && stdOut == stdErr {
		o.logFile(ctx, deploymentID, stdOut, "StdOut/StdErr", sshClient)
	} else {
		if existStdOut {
			o.logFile(ctx, deploymentID, stdOut, "StdOut", sshClient)
		}
		if existStdErr {
			o.logFile(ctx, deploymentID, stdErr, "StdErr", sshClient)
		}
	}

	// See default output if nothing is specified here
	if !existStdOut && !existStdErr {
		o.logFile(ctx, deploymentID, fmt.Sprintf("slurm-%s.out", actionData.jobID), "StdOut/Stderr", sshClient)
	}

	// See if monitoring must be continued and set job state if terminated
	switch info["JobState"] {
	case "COMPLETED":
		// job has been done successfully : unregister monitoring
		deregister = true
	case "RUNNING", "PENDING", "COMPLETING", "CONFIGURING", "SIGNALING", "RESIZING":
		// job's still running or its state is about to be set definitively: monitoring is keeping on
	default:
		// Other cases as FAILED, CANCELLED, STOPPED, SUSPENDED, TIMEOUT, etc : error is return with job state and job info is logged
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, deploymentID).RegisterAsString(fmt.Sprintf("job info:%+v", info))
		deregister = true
		err = errors.Errorf("job with ID:%q finished unsuccessfully with state:%q", actionData.jobID, info["JobState"])
	}

	// cleanup except if error occurred or explicitly specified in config
	if deregister && err == nil {
		if !cfg.Infrastructures[infrastructureName].GetBool("keep_job_remote_artifacts") {
			o.removeArtifacts(actionData, sshClient)
		}
	}
	return deregister, err
}

func (o *actionOperator) removeArtifacts(actionData *actionData, sshClient *sshutil.SSHClient) {
	for _, art := range actionData.artifacts {
		p := path.Join(actionData.workingDir, art)
		log.Debugf("Remove artifact %q", p)
		cmd := fmt.Sprintf("rm -f %s", p)
		_, err := sshClient.RunCommand(cmd)
		if err != nil {
			log.Printf("an error:%+v occurred during removing artifact %q", err, p)
		}
	}
}

func (o *actionOperator) logFile(ctx context.Context, deploymentID, filePath, fileType string, sshClient *sshutil.SSHClient) {
	cmd := fmt.Sprintf("cat %s", filePath)
	output, err := sshClient.RunCommand(cmd)
	if err != nil {
		mess := fmt.Sprintf("an error:%+v occurred during logging file:%q", err, filePath)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).RegisterAsString(mess)
	}
	if strings.TrimSpace(output) != "" {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).RegisterAsString(fmt.Sprintf("Run the command: %q", cmd))
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(fmt.Sprintf("%s %s:", fileType, filePath))
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString("\n" + output)
	}
}
