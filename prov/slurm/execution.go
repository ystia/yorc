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

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/helper/sshutil"
	"github.com/ystia/yorc/helper/stringutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
	"github.com/ystia/yorc/prov/operations"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/tosca"
)

type execution interface {
	resolveExecution() error
	executeAsync(ctx context.Context) (*prov.Action, time.Duration, error)
	execute(ctx context.Context) error
}

type operationNotImplemented struct {
	msg string
}

type noJobFound struct {
	msg string
}

func (oni operationNotImplemented) Error() string {
	return oni.msg
}

func (jid *noJobFound) Error() string {
	return jid.msg
}

type executionCommon struct {
	kv                     *api.KV
	cfg                    config.Configuration
	deploymentID           string
	taskID                 string
	client                 *sshutil.SSHClient
	NodeName               string
	operation              prov.Operation
	OperationRemoteBaseDir string
	NodeType               string
	OverlayPath            string
	Artifacts              map[string]string
	EnvInputs              []*operations.EnvInput
	VarInputsNames         []string
	NodePath               string
	Primary                string
	nodeInstances          []string
	jobInfo                *jobInfo
	stepName               string
}

func newExecution(kv *api.KV, cfg config.Configuration, taskID, deploymentID, nodeName, stepName string, operation prov.Operation) (execution, error) {
	execCommon := &executionCommon{kv: kv,
		cfg:                    cfg,
		deploymentID:           deploymentID,
		NodeName:               nodeName,
		operation:              operation,
		VarInputsNames:         make([]string, 0),
		EnvInputs:              make([]*operations.EnvInput, 0),
		taskID:                 taskID,
		OperationRemoteBaseDir: stringutil.UniqueTimestampedName(".yorc_", ""),
		stepName:               stepName,
	}
	if err := execCommon.resolveOperation(); err != nil {
		return nil, err
	}

	isSingularity, err := deployments.IsTypeDerivedFrom(kv, deploymentID, operation.ImplementationArtifact, artifactImageImplementation)
	if err != nil {
		return nil, err
	}

	if isSingularity {
		execSingularity := &executionSingularity{executionCommon: execCommon}
		return execSingularity, execCommon.resolveExecution()
	}

	return execCommon, execCommon.resolveExecution()
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
		log.Printf("Running the job: %s", e.operation.Name)
		// Copy the artifacts
		if err := e.uploadArtifacts(ctx); err != nil {
			return errors.Wrap(err, "failed to upload artifact")
		}
		if e.Primary != "" {
			// Copy the operation implementation
			if err := e.uploadFile(ctx, path.Join(e.OverlayPath, e.Primary), e.OverlayPath); err != nil {
				return errors.Wrap(err, "failed to upload operation implementation")
			}
		}

		// Build Job Information
		if err := e.buildJobInfo(ctx); err != nil {
			return errors.Wrap(err, "failed to build job information")
		}

		// Run the command
		err := e.runJobCommand(ctx)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(err.Error())
			return errors.Wrap(err, "failed to run command")
		}

		jobInfoJSON, err := json.Marshal(e.jobInfo)
		if err != nil {
			return errors.Wrap(err, "Failed to marshal Slurm job information")
		}
		err = tasks.SetTaskData(e.kv, e.taskID, e.NodeName+"-jobInfo", string(jobInfoJSON))
		if err != nil {
			return err
		}
		// Set the JobID attribute
		// TODO(should be contextual to the current workflow)
		err = deployments.SetAttributeForAllInstances(e.kv, e.deploymentID, e.NodeName, "job_id", e.jobInfo.ID)
		if err != nil {
			return errors.Wrap(err, "failed to retrieve job id an manual cleanup may be necessary: ")
		}
	case strings.ToLower(tosca.RunnableCancelOperationName):
		var jobID string
		if jobInfo, err := e.getJobInfoFromTaskContext(); err != nil {
			if !tasks.IsTaskDataNotFoundError(err) {
				return err
			}
			// Not cancelling within the same task try to get jobID from attribute
			_, jobID, err = deployments.GetInstanceAttribute(e.kv, e.deploymentID, e.NodeName, "0", "job_id")
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
	jobInfoJSON, err := tasks.GetTaskData(e.kv, e.taskID, e.NodeName+"-jobInfo")
	if err != nil {
		return nil, err
	}
	jobInfo := new(jobInfo)
	err = json.Unmarshal([]byte(jobInfoJSON), jobInfo)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal stored Slurm job information")
	}
	return jobInfo, nil
}

func (e *executionCommon) resolveOperation() error {
	e.NodePath = path.Join(consulutil.DeploymentKVPrefix, e.deploymentID, "topology/nodes", e.NodeName)
	var err error
	e.NodeType, err = deployments.GetNodeType(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}

	_, e.Primary, err = deployments.GetOperationPathAndPrimaryImplementation(e.kv, e.deploymentID, e.operation.ImplementedInNodeTemplate, e.operation.ImplementedInType, e.operation.Name)
	if err != nil {
		return err
	}

	e.Primary = strings.TrimSpace(e.Primary)

	if e.operation.ImplementedInType == "yorc.nodes.slurm.Job" && e.Primary == "embedded" {
		e.Primary = ""
	}

	log.Debugf("primary implementation: %q", e.Primary)
	return e.resolveInstances()
}

func (e *executionCommon) buildJobMonitoringAction() *prov.Action {
	// Fill all used data for job monitoring
	data := make(map[string]string)
	data["taskID"] = e.taskID
	data["jobID"] = e.jobInfo.ID
	data["stepName"] = e.stepName
	data["isBatch"] = strconv.FormatBool(e.jobInfo.BatchMode)
	data["remoteBaseDirectory"] = e.OperationRemoteBaseDir
	data["remoteExecDirectory"] = e.jobInfo.OperationRemoteExecDir
	data["outputs"] = strings.Join(e.jobInfo.Outputs, ",")
	return &prov.Action{ActionType: "job-monitoring", Data: data}
}

func (e *executionCommon) retrieveJobID(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Debugf("Task has been cancelled. The job information polling is stopping now")
			return nil
		case <-ticker.C:
			jobInfo, err := getJobInfo(e.client, "", e.jobInfo.Name)
			if err != nil {
				_, notFound := err.(*noJobFound)
				// If job is not found, we assume it still hasn't be created
				if !notFound {
					return err
				}
			} else if jobInfo.ID != "" {
				// JobID has been retrieved
				e.jobInfo.ID = jobInfo.ID
				return nil
			}
		}
	}
}

func (e *executionCommon) buildJobInfo(ctx context.Context) error {
	job := jobInfo{}
	// Get main properties from node
	jobName, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.NodeName, "name")
	if err != nil {
		return err
	}
	if jobName == nil || jobName.RawString() == "" {
		job.Name = e.cfg.Infrastructures[infrastructureName].GetString("default_job_name")
		if job.Name == "" {
			job.Name = e.deploymentID
		}
	} else {
		job.Name = jobName.RawString()
	}

	if ts, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.NodeName, "tasks"); err != nil {
		return err
	} else if ts != nil && ts.RawString() != "" {
		if job.Tasks, err = strconv.Atoi(ts.RawString()); err != nil {
			return err
		}
	}

	var nodes = 1
	if ns, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.NodeName, "nodes"); err != nil {
		return err
	} else if ns != nil && ns.RawString() != "" {
		if nodes, err = strconv.Atoi(ns.RawString()); err != nil {
			return err
		}
	}
	job.Nodes = nodes

	if m, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.NodeName, "mem_per_node"); err != nil {
		return err
	} else if m != nil && m.RawString() != "" {
		if job.Mem, err = strconv.Atoi(m.RawString()); err != nil {
			return err
		}
	}

	if c, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.NodeName, "cpus_per_task"); err != nil {
		return err
	} else if c != nil && c.RawString() != "" {
		if job.Cpus, err = strconv.Atoi(c.RawString()); err != nil {
			return err
		}
	}

	if maxTime, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.NodeName, "time"); err != nil {
		return err
	} else if maxTime != nil {
		job.MaxTime = maxTime.RawString()
	}

	if monitoringTime, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.NodeName, "monitoring_time_interval"); err != nil {
		return err
	} else if monitoringTime != nil && monitoringTime.RawString() != "" {
		job.MonitoringTimeInterval, err = time.ParseDuration(monitoringTime.RawString())
		if err != nil {
			return err
		}
	}
	if job.MonitoringTimeInterval == 0 {
		job.MonitoringTimeInterval = e.cfg.Infrastructures[infrastructureName].GetDuration("job_monitoring_time_interval")
		if job.MonitoringTimeInterval <= 0 {
			// Default value
			job.MonitoringTimeInterval = 5 * time.Second
		}
	}

	// BatchMode default is true
	var batchMode = true
	if bm, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.NodeName, "batch"); err != nil {
		return err
	} else if bm != nil && bm.RawString() != "" {
		if batchMode, err = strconv.ParseBool(bm.RawString()); err != nil {
			return err
		}
	}
	job.BatchMode = batchMode

	var extraOpts []string
	if extra, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.NodeName, "extra_options"); err != nil {
		return err
	} else if extra != nil && extra.RawString() != "" {
		if err = json.Unmarshal([]byte(extra.RawString()), &extraOpts); err != nil {
			return err
		}
	}
	job.Opts = extraOpts

	var execArgs []string
	if ea, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.NodeName, "exec_args"); err != nil {
		return err
	} else if ea != nil && ea.RawString() != "" {
		if err = json.Unmarshal([]byte(ea.RawString()), &execArgs); err != nil {
			return err
		}
	}

	var args []string
	job.Inputs = make(map[string]string)
	for _, input := range e.EnvInputs {
		if input.Name == "args" && input.Value != "" {
			if err = json.Unmarshal([]byte(input.Value), &args); err != nil {
				return err
			}
		} else {
			job.Inputs[input.Name] = input.Value
		}
	}

	job.ExecArgs = append(execArgs, args...)

	// Retrieve job id from attribute if it was previously set (otherwise will be retrieved when running the job)
	// TODO(loicalbertin) right now I can't see any notion of multi-instances for Slurm jobs but this sounds bad to me
	_, job.ID, err = deployments.GetInstanceAttribute(e.kv, e.deploymentID, e.NodeName, "0", "job_id")
	if err != nil {
		return err
	}
	e.jobInfo = &job
	return nil
}

func (e *executionCommon) fillJobCommandOpts() string {
	var opts string
	opts += fmt.Sprintf(" --job-name=%s", e.jobInfo.Name)

	if e.jobInfo.Tasks > 1 {
		opts += fmt.Sprintf(" --ntasks=%d", e.jobInfo.Tasks)
	}
	opts += fmt.Sprintf(" --nodes=%d", e.jobInfo.Nodes)

	if e.jobInfo.Mem != 0 {
		opts += fmt.Sprintf(" --mem=%dG", e.jobInfo.Mem)
	}
	if e.jobInfo.Cpus != 0 {
		opts += fmt.Sprintf(" --cpus-per-task=%d", e.jobInfo.Cpus)
	}
	if e.jobInfo.MaxTime != "" {
		opts += fmt.Sprintf(" --time=%s", e.jobInfo.MaxTime)
	}
	if e.jobInfo.Opts != nil && len(e.jobInfo.Opts) > 0 {
		for _, opt := range e.jobInfo.Opts {
			opts += fmt.Sprintf(" --%s", opt)
		}
	}
	return opts
}

func (e *executionCommon) runJobCommand(ctx context.Context) error {
	opts := e.fillJobCommandOpts()
	execFile := ""
	if e.Primary != "" {
		execFile = path.Join(e.OperationRemoteBaseDir, e.NodeName, e.operation.Name, e.Primary)
	}
	if e.jobInfo.BatchMode {
		if e.Primary != "" {
			e.jobInfo.OperationRemoteExecDir = path.Dir(execFile)
		} else {
			e.jobInfo.OperationRemoteExecDir = path.Join(e.OperationRemoteBaseDir, e.NodeName, e.operation.Name)
		}
		err := e.findBatchOutput(ctx)
		if err != nil {
			return err
		}
		return e.runBatchMode(ctx, opts, execFile)
	}
	err := e.runInteractiveMode(ctx, opts, execFile)
	if err != nil {
		return err
	}
	// retrieve jobInfo
	return e.retrieveJobID(ctx)
}

func (e *executionCommon) runInteractiveMode(ctx context.Context, opts, execFile string) error {
	// Add inputs as env variables
	var exports string
	for k, v := range e.jobInfo.Inputs {
		log.Debugf("Add env var with key:%q and value:%q", k, v)
		export := fmt.Sprintf("export %s=%s;", k, v)
		exports += export
	}
	// srun stdout/stderr is redirected on output file and run in asynchronous mode
	redirectFile := stringutil.UniqueTimestampedName("yorc_", "")
	e.jobInfo.Outputs = []string{redirectFile}
	cmd := fmt.Sprintf("%ssrun %s %s %s > %s &", exports, opts, execFile, strings.Join(e.jobInfo.ExecArgs, " "), redirectFile)
	cmd = strings.Trim(cmd, "")
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).RegisterAsString(fmt.Sprintf("Run the command: %q", cmd))
	output, err := e.client.RunCommand(cmd)
	if err != nil {
		log.Debugf("stderr:%q", output)
		return errors.Wrap(err, output)
	}
	return nil
}

func (e *executionCommon) runBatchMode(ctx context.Context, opts, execFile string) error {
	// Exec args are passed via env var to sbatch script if "key1=value1, key2=value2" format
	var exports string
	for _, arg := range e.jobInfo.ExecArgs {
		if is, key, val := parseKeyValue(arg); is {
			log.Debugf("Add env var with key:%q and value:%q", key, val)
			export := fmt.Sprintf("export %s=%s;", key, val)
			exports += export
		}
	}
	for k, v := range e.jobInfo.Inputs {
		log.Debugf("Add env var with key:%q and value:%q", k, v)
		export := fmt.Sprintf("export %s=%s;", k, v)
		exports += export
	}
	cmd := fmt.Sprintf("%scd %s;sbatch %s %s", exports, path.Dir(execFile), opts, path.Base(execFile))
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).RegisterAsString(fmt.Sprintf("Run the command: %q", cmd))
	output, err := e.client.RunCommand(cmd)
	if err != nil {
		log.Debugf("stderr:%q", output)
		return errors.Wrap(err, output)
	}
	output = strings.Trim(output, "\n")
	if e.jobInfo.ID, err = parseJobIDFromBatchOutput(output); err != nil {
		return err
	}
	// Set default output if nothing is specified by user
	// this is the default output generated by sbatch
	if len(e.jobInfo.Outputs) == 0 {
		e.jobInfo.Outputs = []string{fmt.Sprintf("slurm-%s.out", e.jobInfo.ID)}
	}
	log.Debugf("JobID:%q", e.jobInfo.ID)
	return nil
}

func (e *executionCommon) findBatchOutput(ctx context.Context) error {
	pathExecFile := path.Join(e.OverlayPath, e.Primary)
	script, err := os.Open(pathExecFile)
	if err != nil {
		return errors.Wrapf(err, "Failed to open file with path:%q", pathExecFile)
	}

	var all = true
	outputs := parseOutputConfigFromOpts(e.jobInfo.Opts)
	log.Debugf("outputs:%+v", outputs)
	if len(outputs) > 0 {
		// options override SBATCH parameters, only get srun outputs options
		all = false
	}

	o, err := parseOutputConfigFromBatchScript(script, all)
	if err != nil {
		return errors.Wrapf(err, "Failed to parse batch file to retrieve outputs with path:%q", pathExecFile)
	}
	e.jobInfo.Outputs = append(outputs, o...)
	log.Debugf("job outputs:%+v", e.jobInfo.Outputs)
	return nil
}

func (e *executionCommon) uploadArtifacts(ctx context.Context) error {
	log.Debugf("Upload artifacts to remote host")
	var g errgroup.Group
	for _, artPath := range e.Artifacts {
		log.Debugf("handle artifact path:%q", artPath)
		func(artPath string) {
			g.Go(func() error {
				sourcePath := path.Join(e.OverlayPath, artPath)
				fileInfo, err := os.Stat(sourcePath)
				if err != nil {
					return err
				}
				if fileInfo.IsDir() {
					return e.walkArtifactDirectory(ctx, sourcePath, fileInfo, e.OverlayPath)
				}
				return e.uploadFile(ctx, sourcePath, e.OverlayPath)
			})
		}(artPath)
	}
	return g.Wait()
}

func (e *executionCommon) walkArtifactDirectory(ctx context.Context, rootPath string, fileInfo os.FileInfo, artifactBaseDir string) error {
	return filepath.Walk(rootPath, func(pathFile string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		log.Debugf("Walk path:%s", pathFile)
		if !info.IsDir() {
			return e.uploadFile(ctx, pathFile, artifactBaseDir)
		}
		return nil
	})
}

func (e *executionCommon) uploadFile(ctx context.Context, pathFile, artifactBaseDir string) error {
	relPath, err := filepath.Rel(artifactBaseDir, pathFile)
	if err != nil {
		return err
	}

	// Read file in bytes
	source, err := ioutil.ReadFile(pathFile)
	if err != nil {
		return err
	}

	remotePath := path.Join(e.OperationRemoteBaseDir, e.NodeName, e.operation.Name, relPath)
	log.Debugf("uploadFile file from source path:%q to remote relative path:%q", pathFile, remotePath)
	if err := e.client.CopyFile(bytes.NewReader(source), remotePath, "0755"); err != nil {
		log.Debugf("an error occurred:%+v", err)
		return err
	}
	return nil
}

func (e *executionCommon) resolveInstances() error {
	var err error
	if e.nodeInstances, err = tasks.GetInstances(e.kv, e.taskID, e.deploymentID, e.NodeName); err != nil {
		return err
	}
	return nil
}

func (e *executionCommon) resolveExecution() error {
	log.Debugf("Preparing execution of operation %q on node %q for deployment %q", e.operation.Name, e.NodeName, e.deploymentID)
	ovPath, err := filepath.Abs(filepath.Join(e.cfg.WorkingDirectory, "deployments", e.deploymentID, "overlay"))
	if err != nil {
		return err
	}
	e.OverlayPath = ovPath

	if err = e.resolveInputs(); err != nil {
		return err
	}
	if err = e.resolveArtifacts(); err != nil {
		return err
	}

	e.client, err = GetSSHClient(e.cfg)
	return err
}

func (e *executionCommon) resolveInputs() error {
	var err error
	e.EnvInputs, e.VarInputsNames, err = operations.ResolveInputsWithInstances(e.kv, e.deploymentID, e.NodeName, e.taskID, e.operation, nil, nil)
	return err
}

func (e *executionCommon) resolveArtifacts() error {
	var err error
	log.Debugf("Get artifacts for node:%q", e.NodeName)
	e.Artifacts, err = deployments.GetArtifactsForNode(e.kv, e.deploymentID, e.NodeName)
	log.Debugf("Resolved artifacts: %v", e.Artifacts)
	return err
}
