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
	"golang.org/x/sync/errgroup"
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
	OperationPath          string
	Primary                string
	nodeInstances          []string
	jobInfo                *jobInfo
	OperationRemoteExecDir string
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
	case "tosca.interfaces.node.lifecycle.runnable.run":
		// Build Job Information
		if err := e.buildJobInfo(ctx); err != nil {
			return nil, 0, errors.Wrap(err, "failed to build job information")
		}

		return e.buildJobMonitoringAction(), e.jobInfo.monitoringTimeInterval, nil
	default:
		return nil, 0, errors.Errorf("Unsupported operation %q", e.operation.Name)
	}
}

func (e *executionCommon) execute(ctx context.Context) error {
	// Only runnable operation is currently supported
	log.Debugf("Execute the operation:%+v", e.operation)
	// Fill log optional fields for log registration
	switch strings.ToLower(e.operation.Name) {
	case "tosca.interfaces.node.lifecycle.runnable.submit":
		log.Printf("Running the job: %s", e.operation.Name)
		// Copy the artifacts
		if err := e.uploadArtifacts(ctx); err != nil {
			return errors.Wrap(err, "failed to upload artifact")
		}

		// Copy the operation implementation
		if err := e.uploadFile(ctx, path.Join(e.OverlayPath, e.Primary), e.OverlayPath); err != nil {
			return errors.Wrap(err, "failed to upload operation implementation")
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

		// Set the JobID attribute
		err = deployments.SetAttributeForAllInstances(e.kv, e.deploymentID, e.NodeName, "job_id", e.jobInfo.ID)
		if err != nil {
			return errors.Wrap(err, "failed to retrieve job id an manual cleanup may be necessary: ")
		}
	case "tosca.interfaces.node.lifecycle.runnable.cancel":
		// Retrieve job id from attribute if it was previously set (otherwise will be retrieved when running the job)
		// TODO(loicalbertin) right now I can't see any notion of multi-instances for Slurm jobs but this sounds bad to me
		found, jobID, err := deployments.GetInstanceAttribute(e.kv, e.deploymentID, e.NodeName, "0", "job_id")
		if err != nil {
			return err
		}
		if !found || jobID == "" {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, e.deploymentID).RegisterAsString("trying to cancel a job that seems not scheduled as we can't retrieve its jobID")
			return nil
		}
		return cancelJobID(jobID, e.client)
	default:
		return errors.Errorf("Unsupported operation %q", e.operation.Name)
	}
	return nil
}

func (e *executionCommon) resolveOperation() error {
	e.NodePath = path.Join(consulutil.DeploymentKVPrefix, e.deploymentID, "topology/nodes", e.NodeName)
	var err error
	e.NodeType, err = deployments.GetNodeType(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}
	operationNodeType := e.NodeType
	e.OperationPath, e.Primary, err = deployments.GetOperationPathAndPrimaryImplementation(e.kv, e.deploymentID, e.operation.ImplementedInNodeTemplate, operationNodeType, e.operation.Name)
	if err != nil {
		return err
	}
	if e.OperationPath == "" || e.Primary == "" {
		return operationNotImplemented{msg: fmt.Sprintf("primary implementation missing for operation %q of type %q in deployment %q is missing", e.operation.Name, e.NodeType, e.deploymentID)}
	}
	e.Primary = strings.TrimSpace(e.Primary)
	log.Debugf("Operation Path: %q, primary implementation: %q", e.OperationPath, e.Primary)
	return e.resolveInstances()
}

func (e *executionCommon) buildJobMonitoringAction() *prov.Action {
	// Fill all used data for job monitoring
	data := make(map[string]string)
	data["taskID"] = e.taskID
	data["jobID"] = e.jobInfo.ID
	data["stepName"] = e.stepName
	data["isBatch"] = strconv.FormatBool(e.jobInfo.batchMode)
	data["remoteBaseDirectory"] = e.OperationRemoteBaseDir
	data["remoteExecDirectory"] = e.OperationRemoteExecDir
	data["outputs"] = strings.Join(e.jobInfo.outputs, ",")
	return &prov.Action{ActionType: "job-monitoring", Data: data}
}

func (e *executionCommon) retrieveJobID(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			log.Debugf("Task has been cancelled. The job information polling is stopping now")
			ticker.Stop()
			return nil
		case <-ticker.C:
			jobInfo, err := getJobInfo(e.client, "", e.jobInfo.name)
			if err != nil {
				_, notFound := err.(*noJobFound)
				// If job is not found, we assume it still hasn't be created
				if !notFound {
					return err
				}
			} else if jobInfo.ID != "" {
				// JobID has been retrieved
				e.jobInfo.ID = jobInfo.ID
				ticker.Stop()
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
		job.name = e.cfg.Infrastructures[infrastructureName].GetString("default_job_name")
		if job.name == "" {
			job.name = e.deploymentID
		}
	} else {
		job.name = jobName.RawString()
	}

	if ts, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.NodeName, "tasks"); err != nil {
		return err
	} else if ts != nil && ts.RawString() != "" {
		if job.tasks, err = strconv.Atoi(ts.RawString()); err != nil {
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
	job.nodes = nodes

	if m, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.NodeName, "mem_per_node"); err != nil {
		return err
	} else if m != nil && m.RawString() != "" {
		if job.mem, err = strconv.Atoi(m.RawString()); err != nil {
			return err
		}
	}

	if c, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.NodeName, "cpus_per_task"); err != nil {
		return err
	} else if c != nil && c.RawString() != "" {
		if job.cpus, err = strconv.Atoi(c.RawString()); err != nil {
			return err
		}
	}

	if maxTime, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.NodeName, "time"); err != nil {
		return err
	} else if maxTime != nil {
		job.maxTime = maxTime.RawString()
	}

	if monitoringTime, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.NodeName, "monitoring_time_interval"); err != nil {
		return err
	} else if monitoringTime != nil && monitoringTime.RawString() != "" {
		job.monitoringTimeInterval, err = time.ParseDuration(monitoringTime.RawString())
		if err != nil {
			return err
		}
	}
	if job.monitoringTimeInterval == 0 {
		ti := e.cfg.Infrastructures[infrastructureName].GetString("job_monitoring_time_interval")
		if ti != "" {
			jmti, err := time.ParseDuration(ti)
			if err != nil {
				log.Printf("Invalid format for job monitoring time interval configuration:%q. Default 5s time interval will be used instead.", ti)
			}
			job.monitoringTimeInterval = jmti
		} else {
			// Default value
			job.monitoringTimeInterval = 5 * time.Second
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
	job.batchMode = batchMode

	var extraOpts []string
	if extra, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.NodeName, "extra_options"); err != nil {
		return err
	} else if extra != nil && extra.RawString() != "" {
		if err = json.Unmarshal([]byte(extra.RawString()), &extraOpts); err != nil {
			return err
		}
	}
	job.opts = extraOpts

	var args []string
	job.inputs = make(map[string]string)
	for _, input := range e.EnvInputs {
		if input.Name == "args" && input.Value != "" {
			if err = json.Unmarshal([]byte(input.Value), &args); err != nil {
				return err
			}
		} else {
			job.inputs[input.Name] = input.Value
		}
	}

	// Retrieve job id from attribute if it was previously set (otherwise will be retrieved when running the job)
	// TODO(loicalbertin) right now I can't see any notion of multi-instances for Slurm jobs but this sounds bad to me
	_, job.ID, err = deployments.GetInstanceAttribute(e.kv, e.deploymentID, e.NodeName, "0", "job_id")
	if err != nil {
		return err
	}

	job.execArgs = args
	e.jobInfo = &job
	return nil
}

func (e *executionCommon) fillJobCommandOpts() string {
	var opts string
	opts += fmt.Sprintf(" --job-name=%s", e.jobInfo.name)

	if e.jobInfo.tasks > 1 {
		opts += fmt.Sprintf(" --ntasks=%d", e.jobInfo.tasks)
	}
	opts += fmt.Sprintf(" --nodes=%d", e.jobInfo.nodes)

	if e.jobInfo.mem != 0 {
		opts += fmt.Sprintf(" --mem=%dG", e.jobInfo.mem)
	}
	if e.jobInfo.cpus != 0 {
		opts += fmt.Sprintf(" --cpus-per-task=%d", e.jobInfo.cpus)
	}
	if e.jobInfo.maxTime != "" {
		opts += fmt.Sprintf(" --time=%s", e.jobInfo.maxTime)
	}
	if e.jobInfo.opts != nil && len(e.jobInfo.opts) > 0 {
		for _, opt := range e.jobInfo.opts {
			opts += fmt.Sprintf(" --%s", opt)
		}
	}
	return opts
}

func (e *executionCommon) runJobCommand(ctx context.Context) error {
	opts := e.fillJobCommandOpts()
	execFile := path.Join(e.OperationRemoteBaseDir, e.NodeName, e.operation.Name, e.Primary)
	if e.jobInfo.batchMode {
		e.OperationRemoteExecDir = path.Dir(execFile)
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
	for k, v := range e.jobInfo.inputs {
		log.Debugf("Add env var with key:%q and value:%q", k, v)
		export := fmt.Sprintf("export %s=%s;", k, v)
		exports += export
	}
	// srun stdout/stderr is redirected on output file and run in asynchronous mode
	redirectFile := stringutil.UniqueTimestampedName("yorc_", "")
	e.jobInfo.outputs = []string{redirectFile}
	cmd := fmt.Sprintf("%ssrun %s %s %s > %s &", exports, opts, execFile, strings.Join(e.jobInfo.execArgs, " "), redirectFile)
	cmd = strings.Trim(cmd, "")
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).RegisterAsString(fmt.Sprintf("Run the command: %q", cmd))
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
	for _, arg := range e.jobInfo.execArgs {
		if is, key, val := parseKeyValue(arg); is {
			log.Debugf("Add env var with key:%q and value:%q", key, val)
			export := fmt.Sprintf("export %s=%s;", key, val)
			exports += export
		}
	}
	for k, v := range e.jobInfo.inputs {
		log.Debugf("Add env var with key:%q and value:%q", k, v)
		export := fmt.Sprintf("export %s=%s;", k, v)
		exports += export
	}
	cmd := fmt.Sprintf("%scd %s;sbatch %s %s", exports, path.Dir(execFile), opts, path.Base(execFile))
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).RegisterAsString(fmt.Sprintf("Run the command: %q", cmd))
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
	if len(e.jobInfo.outputs) == 0 {
		e.jobInfo.outputs = []string{fmt.Sprintf("slurm-%s.out", e.jobInfo.ID)}
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
	outputs := parseOutputConfigFromOpts(e.jobInfo.opts)
	log.Debugf("outputs:%+v", outputs)
	if len(outputs) > 0 {
		// options override SBATCH parameters, only get srun outputs options
		all = false
	}

	o, err := parseOutputConfigFromBatchScript(script, all)
	if err != nil {
		return errors.Wrapf(err, "Failed to parse batch file to retrieve outputs with path:%q", pathExecFile)
	}
	e.jobInfo.outputs = append(outputs, o...)
	log.Debugf("job outputs:%+v", e.jobInfo.outputs)
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
