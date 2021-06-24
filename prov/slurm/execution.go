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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/sshutil"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/prov/operations"
	"github.com/ystia/yorc/v4/tasks"
	"github.com/ystia/yorc/v4/tosca"
)

const home = "~"
const batchScript = "b-%s.batch"
const srunCommand = "srun"

type execution interface {
	resolveExecution(ctx context.Context) error
	executeAsync(ctx context.Context) (*prov.Action, time.Duration, error)
	execute(ctx context.Context) error
}

type noJobFound struct {
	msg string
}

func (jid *noJobFound) Error() string {
	return jid.msg
}

func isNoJobFoundError(err error) bool {
	cause := errors.Cause(err)
	_, ok := cause.(*noJobFound)
	return ok
}

type executionCommon struct {
	cfg            config.Configuration
	locationProps  config.DynamicMap
	deploymentID   string
	taskID         string
	client         sshutil.Client
	NodeName       string
	operation      prov.Operation
	NodeType       string
	OverlayPath    string
	Artifacts      map[string]string
	EnvInputs      []*operations.EnvInput
	VarInputsNames []string
	Primary        string
	PrimaryFile    string
	nodeInstances  []string
	jobInfo        *jobInfo
	stepName       string
	isSingularity  bool
}

func newExecution(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName, stepName string, operation prov.Operation) (execution, error) {
	isSingularity, err := deployments.IsTypeDerivedFrom(ctx, deploymentID, operation.ImplementationArtifact, artifactImageImplementation)
	if err != nil {
		return nil, err
	}

	var locationProps config.DynamicMap
	locationMgr, err := locations.GetManager(cfg)
	if err == nil {
		locationProps, err = locationMgr.GetLocationPropertiesForNode(ctx, deploymentID, nodeName, infrastructureType)
	}
	if err != nil {
		return nil, err
	}

	execCommon := &executionCommon{
		cfg:            cfg,
		locationProps:  locationProps,
		deploymentID:   deploymentID,
		NodeName:       nodeName,
		operation:      operation,
		VarInputsNames: make([]string, 0),
		EnvInputs:      make([]*operations.EnvInput, 0),
		taskID:         taskID,
		stepName:       stepName,
		isSingularity:  isSingularity,
	}
	if err := execCommon.resolveOperation(ctx); err != nil {
		return nil, err
	}
	// Get user credentials from credentials node property
	// Its not a capability, so capabilityName set to empty string
	creds, err := getUserCredentials(ctx, locationProps, deploymentID, nodeName, "")
	if err != nil {
		return nil, err
	}
	// Create sshClient using user credentials from credentials property if the are provided, or from yorc config otherwise
	execCommon.client, err = getSSHClient(cfg, creds, locationProps)
	if err != nil {
		return nil, err
	}

	if isSingularity {
		execSingularity := &executionSingularity{executionCommon: execCommon}
		return execSingularity, execCommon.resolveExecution(ctx)
	}

	return execCommon, execCommon.resolveExecution(ctx)
}

func (e *executionCommon) executeAsync(ctx context.Context) (*prov.Action, time.Duration, error) {
	// Only runnable operation is currently supported
	log.Debugf("Execute the operation:%+v", e.operation)
	// Fill log optional fields for log registration
	switch strings.ToLower(e.operation.Name) {
	case strings.ToLower(tosca.RunnableRunOperationName):
		// Build Job Information
		var err error
		e.jobInfo, err = e.getJobInfoFromTaskContext()
		if err != nil {
			return nil, 0, err
		}
		return e.buildJobMonitoringAction(), e.jobInfo.MonitoringTimeInterval, nil
	default:
		return nil, 0, errors.Errorf("Unsupported operation %q", e.operation.Name)
	}
}

func (e *executionCommon) execute(ctx context.Context) error {
	// Only runnable operation is currently supported
	log.Debugf("Execute the operation:%+v", e.operation)
	// Fill log optional fields for log registration
	switch strings.ToLower(e.operation.Name) {
	case strings.ToLower(tosca.RunnableSubmitOperationName):
		log.Debugf("Submit the job: %s", e.operation.Name)
		// Build Job Information
		if err := e.buildJobInfo(ctx); err != nil {
			return errors.Wrap(err, "failed to build job information")
		}
		if e.jobInfo.ExecutionOptions.Command != "" && e.Primary != "" {
			// If both primary artifact is provided (script) and command: return an error
			return errors.Errorf("Either a script artifact or a command must be provided, but not both.")
		}

		// Add the primary artifact to the artifacts map if not already included
		var is bool
		if e.Primary != "" && e.PrimaryFile != "" {
			for _, artPath := range e.Artifacts {
				if strings.HasPrefix(e.Primary, artPath) {
					is = true
				}
			}
			if !is {
				e.Artifacts[e.PrimaryFile] = e.Primary
			}
		}

		// Copy the artifacts
		if err := e.uploadArtifacts(ctx); err != nil {
			return errors.Wrap(err, "failed to upload artifact")
		}
		err := e.prepareAndSubmitJob(ctx)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(err.Error())
			return errors.Wrapf(err, "failed to submit job with ID:%s", e.jobInfo.ID)
		}

		jobInfoJSON, err := json.Marshal(e.jobInfo)
		if err != nil {
			return errors.Wrap(err, "Failed to marshal Slurm job information")
		}
		err = tasks.SetTaskData(e.taskID, e.NodeName+"-jobInfo", string(jobInfoJSON))
		if err != nil {
			return err
		}
		// Set the JobID attribute
		// TODO(should be contextual to the current workflow)
		err = deployments.SetAttributeForAllInstances(ctx, e.deploymentID, e.NodeName, "job_id", e.jobInfo.ID)
		if err != nil {
			return errors.Wrap(err, "failed to retrieve job id an manual cleanup may be necessary: ")
		}
	case strings.ToLower(tosca.RunnableCancelOperationName):
		var jobID string
		if jobInfo, err := e.getJobInfoFromTaskContext(); err != nil {
			if !tasks.IsTaskDataNotFoundError(err) {
				return err
			}
			// TODO(loicalbertin) for now we consider only instance 0 (https://github.com/ystia/yorc/issues/670)
			// Not cancelling within the same task try to get jobID from attribute
			id, err := deployments.GetInstanceAttributeValue(ctx, e.deploymentID, e.NodeName, "0", "job_id")
			if err != nil {
				return err
			} else if id != nil && id.RawString() != "" {
				jobID = id.String()
			}
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf(
				"Slurm job cancellation called from a dedicated \"cancel\" workflow. JobID retrieved from node %q attribute. This may cause issues if multiple workflows are running in parallel. Prefer using a workflow cancellation.", e.NodeName)
		} else {
			jobID = jobInfo.ID
		}
		return cancelJobID(jobID, e.client)
	default:
		return errors.Errorf("Unsupported operation %q", e.operation.Name)
	}
	return nil
}

func (e *executionCommon) getJobInfoFromTaskContext() (*jobInfo, error) {
	jobInfoJSON, err := tasks.GetTaskData(e.taskID, e.NodeName+"-jobInfo")
	if err != nil {
		return nil, err
	}
	jobInfo := new(jobInfo)
	err = json.Unmarshal([]byte(jobInfoJSON), jobInfo)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal stored Slurm job information")
	}
	log.Debugf("Unmarshal Job info for task %s, Job ID %q.", e.taskID, jobInfo.ID)
	return jobInfo, nil
}

func (e *executionCommon) buildJobMonitoringAction() *prov.Action {
	// Fill all used data for job monitoring
	data := make(map[string]string)
	data["taskID"] = e.taskID
	data["jobID"] = e.jobInfo.ID
	data["stepName"] = e.stepName
	data["nodeName"] = e.NodeName
	data["workingDir"] = e.jobInfo.WorkingDir
	data["artifacts"] = strings.Join(e.jobInfo.Artifacts, ",")

	return &prov.Action{ActionType: "job-monitoring", Data: data}
}

func (e *executionCommon) buildJobInfo(ctx context.Context) error {
	// Get main properties from node
	e.jobInfo = &jobInfo{}
	jobName, err := deployments.GetNodePropertyValue(ctx, e.deploymentID, e.NodeName, "slurm_options", "name")
	if err != nil {
		return err
	}
	if jobName == nil || jobName.RawString() == "" {
		e.jobInfo.Name = e.locationProps.GetString("default_job_name")
		if e.jobInfo.Name == "" {
			e.jobInfo.Name = e.deploymentID
		}
	} else {
		e.jobInfo.Name = jobName.RawString()
	}

	if ts, err := deployments.GetNodePropertyValue(ctx, e.deploymentID, e.NodeName, "slurm_options", "tasks"); err != nil {
		return err
	} else if ts != nil && ts.RawString() != "" {
		if e.jobInfo.Tasks, err = strconv.Atoi(ts.RawString()); err != nil {
			return err
		}
	}

	var nodes = 1
	if ns, err := deployments.GetNodePropertyValue(ctx, e.deploymentID, e.NodeName, "slurm_options", "nodes"); err != nil {
		return err
	} else if ns != nil && ns.RawString() != "" {
		if nodes, err = strconv.Atoi(ns.RawString()); err != nil {
			return err
		}
	}
	e.jobInfo.Nodes = nodes

	if m, err := deployments.GetNodePropertyValue(ctx, e.deploymentID, e.NodeName, "slurm_options", "mem_per_node"); err != nil {
		return err
	} else if m != nil && m.RawString() != "" {
		if e.jobInfo.Mem, err = toSlurmMemFormat(m.RawString()); err != nil {
			return err
		}
	}

	if c, err := deployments.GetNodePropertyValue(ctx, e.deploymentID, e.NodeName, "slurm_options", "cpus_per_task"); err != nil {
		return err
	} else if c != nil && c.RawString() != "" {
		if e.jobInfo.Cpus, err = strconv.Atoi(c.RawString()); err != nil {
			return err
		}
	}

	if maxTime, err := deployments.GetNodePropertyValue(ctx, e.deploymentID, e.NodeName, "slurm_options", "time"); err != nil {
		return err
	} else if maxTime != nil {
		e.jobInfo.MaxTime = maxTime.RawString()
	}

	if monitoringTime, err := deployments.GetNodePropertyValue(ctx, e.deploymentID, e.NodeName, "monitoring_time_interval"); err != nil {
		return err
	} else if monitoringTime != nil && monitoringTime.RawString() != "" {
		e.jobInfo.MonitoringTimeInterval, err = time.ParseDuration(monitoringTime.RawString())
		if err != nil {
			return err
		}
	}
	if e.jobInfo.MonitoringTimeInterval == 0 {
		e.jobInfo.MonitoringTimeInterval = e.locationProps.GetDuration("job_monitoring_time_interval")
		if e.jobInfo.MonitoringTimeInterval <= 0 {
			// Default value
			e.jobInfo.MonitoringTimeInterval = 5 * time.Second
		}
	}

	if extra, err := deployments.GetNodePropertyValue(ctx, e.deploymentID, e.NodeName, "slurm_options", "extra_options"); err != nil {
		return err
	} else if extra != nil && extra.RawString() != "" {
		if err = json.Unmarshal([]byte(extra.RawString()), &e.jobInfo.Opts); err != nil {
			return err
		}
	}
	e.jobInfo.Inputs = make(map[string]string)
	for _, input := range e.EnvInputs {
		if !strings.Contains(input.Name, "credentials") {
			e.jobInfo.Inputs[input.Name] = input.Value
		}
	}

	// Retrieve job id from attribute if it was previously set (otherwise will be retrieved when running the job)
	// TODO(loicalbertin) right now I can't see any notion of multi-instances for Slurm jobs but this sounds bad to me
	id, err := deployments.GetInstanceAttributeValue(ctx, e.deploymentID, e.NodeName, "0", "job_id")
	if err != nil {
		return err
	} else if id != nil && id.RawString() != "" {
		e.jobInfo.ID = id.String()
	}

	// Job account
	if acc, err := deployments.GetNodePropertyValue(ctx, e.deploymentID, e.NodeName, "slurm_options", "account"); err != nil {
		return err
	} else if acc != nil && acc.RawString() != "" {
		e.jobInfo.Account = acc.RawString()
	} else if e.locationProps.GetBool("enforce_accounting") {
		return errors.Errorf("Job account must be set as configuration enforces accounting")
	}

	// Reservation
	if res, err := deployments.GetNodePropertyValue(ctx, e.deploymentID, e.NodeName, "slurm_options", "reservation"); err != nil {
		return err
	} else if res != nil && res.RawString() != "" {
		e.jobInfo.Reservation = res.RawString()
	}

	// Execution options
	eo, err := deployments.GetNodePropertyValue(ctx, e.deploymentID, e.NodeName, "execution_options")
	if err != nil {
		return err
	}
	if eo != nil && eo.RawString() != "" {
		err = mapstructure.Decode(eo.Value, &e.jobInfo.ExecutionOptions)
		if err != nil {
			return errors.Wrapf(err, `invalid execution options datatype for attribute "execution_options" for node %q`, e.NodeName)
		}
	}

	if e.jobInfo.ExecutionOptions.Command == "" && e.Primary == "" {
		return errors.Errorf("Either job command property must be filled or batch script must be provided")
	}

	// Working directory: default is user's home
	if wd, err := deployments.GetNodePropertyValue(ctx, e.deploymentID, e.NodeName, "working_directory"); err != nil {
		return err
	} else if wd != nil && wd.RawString() != "" {
		e.jobInfo.WorkingDir = wd.RawString()
	} else {
		e.jobInfo.WorkingDir = home
	}

	envFile, err := deployments.GetNodePropertyValue(ctx, e.deploymentID, e.NodeName, "environment_file")
	if err != nil {
		return err
	}
	if envFile != nil {
		e.jobInfo.EnvFile = envFile.RawString()
	}
	return nil
}

func (e *executionCommon) buildJobOpts() string {
	var opts string
	opts += fmt.Sprintf(" --job-name='%s'", e.jobInfo.Name)
	if e.jobInfo.Tasks > 1 {
		opts += fmt.Sprintf(" --ntasks=%d", e.jobInfo.Tasks)
	}
	opts += fmt.Sprintf(" --nodes=%d", e.jobInfo.Nodes)
	if e.jobInfo.Mem != "" {
		opts += fmt.Sprintf(" --mem='%s'", e.jobInfo.Mem)
	}
	if e.jobInfo.Cpus != 0 {
		opts += fmt.Sprintf(" --cpus-per-task=%d", e.jobInfo.Cpus)
	}
	if e.jobInfo.MaxTime != "" {
		opts += fmt.Sprintf(" --time='%s'", e.jobInfo.MaxTime)
	}
	if e.jobInfo.Opts != nil && len(e.jobInfo.Opts) > 0 {
		opts += fmt.Sprintf(" %s", strings.Join(e.jobInfo.Opts, " "))
	}
	if e.jobInfo.Reservation != "" {
		opts += fmt.Sprintf(" --reservation='%s'", e.jobInfo.Reservation)
	}
	if e.jobInfo.Account != "" {
		opts += fmt.Sprintf(" --account='%s'", e.jobInfo.Account)
	}
	log.Debugf("opts=%q", opts)
	return opts
}

func (e *executionCommon) prepareAndSubmitJob(ctx context.Context) error {
	var cmd string
	if e.jobInfo.ExecutionOptions.Command != "" {
		if strings.HasPrefix(strings.TrimSpace(e.jobInfo.ExecutionOptions.Command), srunCommand+" ") {
			e.jobInfo.ExecutionOptions.Command = e.jobInfo.ExecutionOptions.Command[5:]
		}
		inner := fmt.Sprintf("%s %s %s", srunCommand, e.jobInfo.ExecutionOptions.Command, quoteArgs(e.jobInfo.ExecutionOptions.Args))
		var err error
		cmd, err = e.wrapCommand(inner)
		if err != nil {
			return err
		}
	} else {
		cmd = fmt.Sprintf("%s%s%ssbatch -D %s%s %s", e.sourceEnvFile(), e.addWorkingDirCmd(), e.buildEnvVars(), e.jobInfo.WorkingDir, e.buildJobOpts(), path.Join(e.jobInfo.WorkingDir, e.PrimaryFile))
	}
	return e.submitJob(ctx, cmd)
}

func (e *executionCommon) wrapCommand(innerCmd string) (string, error) {
	// Generate a random UUID to add it to the sbatch wrapper script name
	// this will prevent collisions when running several jobs in parallel
	// see https://github.com/ystia/yorc/issues/522
	id, err := uuid.NewRandom()
	if err != nil {
		return "", errors.Wrap(err, "failed to generate UUID for generated slurm batch script name")
	}
	scriptName := fmt.Sprintf(batchScript, id.String())
	pathScript := path.Join(e.jobInfo.WorkingDir, scriptName)
	// Add the script to the artifact's list
	e.jobInfo.Artifacts = append(e.jobInfo.Artifacts, scriptName)
	// Write script
	cat := fmt.Sprintf(`cat <<'EOF' > %s
#!/bin/bash
%s
%s
EOF
`, pathScript, e.buildInlineSBatchoptions(), innerCmd)
	// Ensure generated script removal after its submission
	return fmt.Sprintf("%s%s%s%ssbatch -D %s%s %s; rm -f %s", e.sourceEnvFile(), e.addWorkingDirCmd(), e.buildEnvVars(), cat, e.jobInfo.WorkingDir, e.buildJobOpts(), pathScript, pathScript), nil
}

func (e *executionCommon) buildInlineSBatchoptions() string {
	var b strings.Builder
	for _, opt := range e.jobInfo.ExecutionOptions.InScriptOptions {
		if strings.HasPrefix(opt, "#") {
			b.WriteString(opt)
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (e *executionCommon) addWorkingDirCmd() string {
	var cmd string
	if e.jobInfo.WorkingDir != home {
		cmd = fmt.Sprintf("mkdir -p %s;", e.jobInfo.WorkingDir)
	}
	return cmd
}

func (e *executionCommon) sourceEnvFile() string {
	var cmd string
	if e.jobInfo.EnvFile != "" {
		cmd = fmt.Sprintf("[ -f %s ] && { source %s ; } ;", e.jobInfo.EnvFile, e.jobInfo.EnvFile)
	}
	return cmd
}

func (e *executionCommon) buildEnvVars() string {
	var exports string
	for _, v := range e.jobInfo.ExecutionOptions.EnvVars {
		if is, key, val := parseKeyValue(v); is {
			log.Debugf("Add env var with key:%q and value:%q", key, val)
			export := fmt.Sprintf("export %s='%s';", key, val)
			exports += export
		}
	}
	for k, v := range e.jobInfo.Inputs {
		log.Debugf("Add env var with key:%q and value:%q", k, v)
		if strings.TrimSpace(k) != "" && strings.TrimSpace(v) != "" {
			export := fmt.Sprintf("export %s='%s';", k, v)
			exports += export
		}
	}
	return exports
}

func (e *executionCommon) submitJob(ctx context.Context, cmd string) error {
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).RegisterAsString(fmt.Sprintf("Run the command: %s", cmd))
	out, err := e.client.RunCommand(cmd)
	if err != nil {
		log.Debugf("stderr:%q", out)
		return errors.Wrap(err, out)
	}
	out = strings.Trim(out, "\n")
	if e.jobInfo.ID, err = retrieveJobID(out); err != nil {
		return err
	}
	log.Debugf("JobID:%q", e.jobInfo.ID)
	return nil
}

func (e *executionCommon) uploadArtifacts(ctx context.Context) error {
	log.Debugf("Upload artifacts to remote host")
	// Add artifact to job artifact's list for monitoring actions
	e.jobInfo.Artifacts = make([]string, 0)
	for k := range e.Artifacts {
		e.jobInfo.Artifacts = append(e.jobInfo.Artifacts, k)
	}

	var g errgroup.Group
	for artName, artPath := range e.Artifacts {
		log.Debugf("handle artifact path:%q, name:%q", artPath, artName)
		func(artPath string) {
			g.Go(func() error {
				sourcePath := path.Join(e.OverlayPath, artPath)
				fileInfo, err := os.Stat(sourcePath)
				if err != nil {
					return err
				}
				if fileInfo.IsDir() {
					return e.walkArtifactDirectory(ctx, sourcePath, fileInfo, path.Dir(sourcePath))
				}
				return e.uploadArtifact(ctx, sourcePath, artName)
			})
		}(artPath)
	}
	return g.Wait()
}

func (e *executionCommon) walkArtifactDirectory(ctx context.Context, rootPath string, fileInfo os.FileInfo, artifactBaseName string) error {
	return filepath.Walk(rootPath, func(pathFile string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		log.Debugf("Walk path:%s", pathFile)
		if !info.IsDir() {
			return e.uploadArtifact(ctx, pathFile, artifactBaseName)
		}
		return nil
	})
}

func (e *executionCommon) uploadArtifact(ctx context.Context, pathFile, artifactBaseName string) error {
	log.Debugf("artifactBaseName:%s", artifactBaseName)
	var relPath string
	if strings.HasSuffix(pathFile, artifactBaseName) {
		relPath = artifactBaseName
	} else {
		var err error
		relPath, err = filepath.Rel(artifactBaseName, pathFile)
		if err != nil {
			return err
		}
	}

	// Read file in bytes
	source, err := ioutil.ReadFile(pathFile)
	if err != nil {
		return err
	}

	remotePath := path.Join(e.jobInfo.WorkingDir, relPath)
	log.Debugf("uploadArtifact file from source path:%q to:%q", pathFile, remotePath)
	return e.client.CopyFile(bytes.NewReader(source), remotePath, "0755")
}

func (e *executionCommon) resolveOperation(ctx context.Context) error {
	var err error
	e.NodeType, err = deployments.GetNodeType(ctx, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}

	// Only Submit operation need to retrieve primary/operation implementation file
	if strings.ToLower(e.operation.Name) != strings.ToLower(tosca.RunnableSubmitOperationName) {
		return nil
	}

	// Only operation file is required for Singularity execution
	if e.isSingularity {
		e.Primary, err = deployments.GetOperationImplementationFile(ctx, e.deploymentID, e.operation.ImplementedInNodeTemplate, e.NodeType, e.operation.Name)
		if err != nil {
			return err
		}
	} else {
		operationImpl, err := deployments.GetOperationImplementation(ctx, e.deploymentID, e.operation.ImplementedInNodeTemplate, e.operation.ImplementedInType, e.operation.Name)
		if err != nil {
			return err
		}
		if operationImpl != nil {
			e.Primary = operationImpl.Primary
		}
	}

	e.Primary = strings.TrimSpace(e.Primary)
	if e.operation.ImplementedInType == "yorc.nodes.slurm.Job" && e.Primary == "embedded" {
		e.Primary = ""
	}

	// Get operation implementation file for upload purpose
	if !e.isSingularity && e.Primary != "" {
		e.PrimaryFile, err = deployments.GetOperationImplementationFile(ctx, e.deploymentID, e.operation.ImplementedInNodeTemplate, e.NodeType, e.operation.Name)
		if err != nil {
			return err
		}
	}

	log.Debugf("primary implementation: %q", e.Primary)
	return e.resolveInstances(ctx)
}

func (e *executionCommon) resolveInstances(ctx context.Context) error {
	var err error
	if e.nodeInstances, err = tasks.GetInstances(ctx, e.taskID, e.deploymentID, e.NodeName); err != nil {
		return err
	}
	return nil
}

func (e *executionCommon) resolveExecution(ctx context.Context) error {
	log.Debugf("Preparing execution of operation %q on node %q for deployment %q", e.operation.Name, e.NodeName, e.deploymentID)
	ovPath, err := operations.GetOverlayPath(e.cfg, e.taskID, e.deploymentID)
	if err != nil {
		return err
	}
	e.OverlayPath = ovPath

	if err = e.resolveInputs(ctx); err != nil {
		return err
	}
	if err = e.resolveArtifacts(ctx); err != nil {
		return err
	}

	return err
}

func (e *executionCommon) resolveInputs(ctx context.Context) error {
	var err error
	e.EnvInputs, e.VarInputsNames, err = operations.ResolveInputsWithInstances(ctx, e.deploymentID, e.NodeName, e.taskID, e.operation, nil, nil)
	return err
}

func (e *executionCommon) resolveArtifacts(ctx context.Context) error {
	var err error
	log.Debugf("Get artifacts for node:%q", e.NodeName)
	e.Artifacts, err = deployments.GetFileArtifactsForNode(ctx, e.deploymentID, e.NodeName)
	log.Debugf("Resolved artifacts: %v", e.Artifacts)
	return err
}
