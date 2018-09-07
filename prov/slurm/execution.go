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
	execute(ctx context.Context, resultCh chan string, errCh chan error)
}

type operationNotImplemented struct {
	msg string
}

type jobIsDone struct {
	msg string
}

func (oni operationNotImplemented) Error() string {
	return oni.msg
}

func (jid jobIsDone) Error() string {
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
	lof                    events.LogOptionalFields
	jobInfoPolling         time.Duration
	OperationRemoteDir     string
}

func newExecution(kv *api.KV, cfg config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) (execution, error) {
	execCommon := &executionCommon{kv: kv,
		cfg:                    cfg,
		deploymentID:           deploymentID,
		NodeName:               nodeName,
		operation:              operation,
		VarInputsNames:         make([]string, 0),
		EnvInputs:              make([]*operations.EnvInput, 0),
		taskID:                 taskID,
		OperationRemoteBaseDir: stringutil.UniqueTimestampedName(".yorc_", ""),
		jobInfoPolling:         5 * time.Second,
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

func (e *executionCommon) execute(ctx context.Context, resultCh chan string, errCh chan error) {
	// Only runnable operation is currently supported
	log.Debugf("Execute the operation:%+v", e.operation)
	// Fill log optional fields for log registration
	wfName, _ := tasks.GetTaskData(e.kv, e.taskID, "workflowName")
	logOptFields := events.LogOptionalFields{
		events.WorkFlowID:    wfName,
		events.NodeID:        e.NodeName,
		events.OperationName: stringutil.GetLastElement(e.operation.Name, "."),
		events.InterfaceName: stringutil.GetAllExceptLastElement(e.operation.Name, "."),
	}
	ctx = events.NewContext(ctx, logOptFields)

	switch strings.ToLower(e.operation.Name) {
	case "tosca.interfaces.node.lifecycle.runnable.run":
		log.Printf("Running the job: %s", e.operation.Name)
		// Copy the artifacts
		if err := e.uploadArtifacts(ctx); err != nil {
			errCh <- errors.Wrap(err, "failed to upload artifact")
		}

		// Copy the operation implementation
		if err := e.uploadFile(ctx, path.Join(e.OverlayPath, e.Primary), e.OverlayPath); err != nil {
			errCh <- errors.Wrap(err, "failed to upload operation implementation")
		}

		// Build Job Information
		if err := e.buildJobInfo(ctx); err != nil {
			errCh <- errors.Wrap(err, "failed to build job information")
		}

		// Run the command
		out, err := e.runJobCommand(ctx)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(err.Error())
			errCh <- errors.Wrap(err, "failed to run command")
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).RegisterAsString(out)
		log.Debugf("output:%q", out)
		if !e.jobInfo.batchMode {
			err := e.cleanUp()
			if err != nil {
				errCh <- err
			}
		}
		resultCh <- e.jobInfo.ID
	default:
		errCh <- errors.Errorf("Unsupported operation %q", e.operation.Name)
	}
}

func (e *executionCommon) pollJobInfo(ctx context.Context, stopCh chan struct{}, errCh chan error) {
	ticker := time.NewTicker(e.jobInfoPolling)
	for {
		select {
		case <-ctx.Done():
			log.Debugf("Task has been cancelled. The job information polling is stopping now")
			ticker.Stop()
			return
		case <-stopCh:
			log.Debugf("The job is done so job information polling is stopping now")
			ticker.Stop()
			return
		case m := <-errCh:
			log.Debugf("Unblocking error occurred retrieving job information due to:%s", m)
			ticker.Stop()
			return
		case <-ticker.C:
			err := e.getJobInfo(ctx)
			if err != nil {
				log.Printf("stderr:%q", err)
				_, jobIsDone := err.(jobIsDone)
				if jobIsDone {
					e.endBatchExecution(ctx, stopCh)
				} else {
					errCh <- err
				}
			}
		}
	}
}

func (e *executionCommon) retrieveJobID(ctx context.Context, stopCh chan struct{}, errCh chan error) {
	ticker := time.NewTicker(e.jobInfoPolling)
	for {
		select {
		case <-ctx.Done():
			log.Debugf("Task has been cancelled. The job information polling is stopping now")
			ticker.Stop()
			return
		case <-stopCh:
			log.Debugf("The job is done so job information polling is stopping now")
			ticker.Stop()
			return
		case m := <-errCh:
			log.Debugf("Unblocking error occurred retrieving job information due to:%s", m)
			ticker.Stop()
			return
		case <-ticker.C:
			err := e.getJobInfo(ctx)
			if err != nil {
				errCh <- err
			} else {
				// JobID has been retrieved
				ticker.Stop()
			}
		}
	}
}

func (e *executionCommon) getJobInfo(ctx context.Context) error {
	var cmd string
	if e.jobInfo.ID != "" {
		cmd = fmt.Sprintf("squeue --noheader --job=%s -o \"%%A,%%T\"", e.jobInfo.ID)
	} else {
		cmd = fmt.Sprintf("squeue --noheader --name=%s -o \"%%A,%%T\"", e.jobInfo.name)
	}

	output, err := e.client.RunCommand(cmd)
	if err != nil {
		return errors.Wrap(err, output)
	}
	out := strings.Trim(output, "\" \t\n\x00")
	if out != "" {
		d := strings.Split(out, ",")
		if len(d) != 2 {
			log.Debugf("Unexpected format job information:%q", out)
			return errors.Errorf("Unexpected format job information:%q", out)
		}
		e.jobInfo.ID = d[0]
		e.jobInfo.state = d[1]
		mess := fmt.Sprintf("Job Name:%s, Job ID:%s, Job State:%s", e.jobInfo.name, e.jobInfo.ID, e.jobInfo.state)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).RegisterAsString(mess)
	} else if e.jobInfo.batchMode {
		return &jobIsDone{msg: "No more job info in squeue command"}
	}
	return nil
}

func (e *executionCommon) endBatchExecution(ctx context.Context, stopCh chan struct{}) {
	// We consider job is done and we stop the polling
	close(stopCh)
	err := e.handleBatchOutputs(ctx)
	if err != nil {
		log.Printf("%+v", err)
	}
	err = e.cleanUp()
	if err != nil {
		log.Printf("%+v", err)
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

func (e *executionCommon) runJobCommand(ctx context.Context) (string, error) {
	opts := e.fillJobCommandOpts()
	execFile := path.Join(e.OperationRemoteBaseDir, e.NodeName, e.operation.Name, e.Primary)
	e.OperationRemoteDir = path.Dir(execFile)
	if e.jobInfo.batchMode {
		// get outputs for batch mode
		err := e.searchForBatchOutputs(ctx)
		if err != nil {
			return "", err
		}
		//go e.pollJobInfo(ctx, stopCh, errCh)
		out, err := e.runBatchMode(ctx, opts, execFile)
		return out, err
	}
	// In interactive mode, we need to retrieve the jobID elsewhere than in the stdout
	stopCh := make(chan struct{})
	errCh := make(chan error)
	go e.retrieveJobID(ctx, stopCh, errCh)
	out, err := e.runInteractiveMode(ctx, opts, execFile)
	return out, err
}

func (e *executionCommon) runInteractiveMode(ctx context.Context, opts, execFile string) (string, error) {
	// Add inputs as env variables
	var exports string
	for k, v := range e.jobInfo.inputs {
		log.Debugf("Add env var with key:%q and value:%q", k, v)
		export := fmt.Sprintf("export %s=%s;", k, v)
		exports += export
	}
	cmd := fmt.Sprintf("%s; srun %s %s %s", exports, opts, execFile, strings.Join(e.jobInfo.execArgs, " "))
	cmd = strings.Trim(cmd, "")
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).RegisterAsString(fmt.Sprintf("Run the command: %q", cmd))
	output, err := e.client.RunCommand(cmd)
	if err != nil {
		log.Debugf("stderr:%q", output)
		return "", errors.Wrap(err, output)
	}
	return output, nil
}

func (e *executionCommon) runBatchMode(ctx context.Context, opts, execFile string) (string, error) {
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
		return "", errors.Wrap(err, output)
	}
	output = strings.Trim(output, "\n")
	if e.jobInfo.ID, err = parseJobIDFromBatchOutput(output); err != nil {
		return "", err
	}
	log.Debugf("JobID:%q", e.jobInfo.ID)
	return output, nil
}

func (e *executionCommon) searchForBatchOutputs(ctx context.Context) error {
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

func (e *executionCommon) handleBatchOutputs(ctx context.Context) error {
	if len(e.jobInfo.outputs) == 0 {
		e.jobInfo.outputs = []string{fmt.Sprintf("slurm-%s.out", e.jobInfo.ID)}
	}
	// Copy the outputs in <JOB_ID>_outputs directory at root level
	outputDir := fmt.Sprintf("job_" + e.jobInfo.ID + "_outputs")
	cmd := fmt.Sprintf("mkdir %s", outputDir)
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).RegisterAsString(fmt.Sprintf("Run the command: %q", cmd))
	output, err := e.client.RunCommand(cmd)
	if err != nil {
		return errors.Wrap(err, output)
	}
	log.Debugf("job outputs:%+v", e.jobInfo.outputs)
	for _, out := range e.jobInfo.outputs {
		oldPath := path.Join(e.OperationRemoteDir, out)
		newPath := path.Join(outputDir, out)
		// Copy the file in the output dir
		cmd := fmt.Sprintf("cp -f %s %s", oldPath, newPath)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).RegisterAsString(fmt.Sprintf("Run the command: %q", cmd))
		output, err := e.client.RunCommand(cmd)
		if err != nil {
			return errors.Wrap(err, output)
		}
		// Log the output
		cmd = fmt.Sprintf("cat %s", newPath)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).RegisterAsString(fmt.Sprintf("Run the command: %q", cmd))
		output, err = e.client.RunCommand(cmd)
		if err != nil {
			return errors.Wrap(err, output)
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).RegisterAsString("\n" + output)
	}
	return nil
}

func (e *executionCommon) cleanUp() error {
	log.Debugf("Cleanup the operation remote base directory")
	cmd := fmt.Sprintf("rm -rf %s", e.OperationRemoteBaseDir)
	_, err := e.client.RunCommand(cmd)
	if err != nil {
		return errors.Wrap(err, "failed to cleanup remote base directory")
	}
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
