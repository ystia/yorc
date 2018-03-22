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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// FIXME need to put code in common with ansible execution

type execution interface {
	resolveExecution() error
	execute(ctx context.Context) error
}

type operationNotImplemented struct {
	msg string
}

// This describes relative slurm job information
type jobInfo struct {
	ID       string
	name     string
	state    string
	tasks    int
	cpus     int
	nodes    int
	mem      int
	maxTime  string
	opts     []string
	execArgs []string
}

func (oni operationNotImplemented) Error() string {
	return oni.msg
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
		OperationRemoteBaseDir: ".yorc",
		jobInfoPolling:         5 * time.Second,
	}
	if err := execCommon.resolveOperation(); err != nil {
		return nil, err
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
	e.OperationPath, e.Primary, err = deployments.GetOperationPathAndPrimaryImplementationForNodeType(e.kv, e.deploymentID, operationNodeType, e.operation.Name)
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

func (e *executionCommon) execute(ctx context.Context) (err error) {
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

		// Pool JobInformation
		chEnd := make(chan struct{})
		chErr := make(chan error)
		go e.poolJobInformation(ctx, chEnd, chErr)

		// Run the command
		out, err := e.runCommand(ctx)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, e.deploymentID).RegisterAsString(err.Error())
			return errors.Wrap(err, "failed to run command")
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.INFO, e.deploymentID).RegisterAsString(out)
		log.Debugf("output:%q", out)
		close(chEnd)
		close(chErr)
		return e.cleanUp()
	default:
		return errors.Errorf("Unsupported operation %q", e.operation.Name)
	}
	return nil
}

func (e *executionCommon) poolJobInformation(ctx context.Context, chEnd chan struct{}, chErr chan error) {
	ticker := time.NewTicker(e.jobInfoPolling)
	go func() {
		for range ticker.C {
			e.getJobInformation(ctx, chErr)
		}
	}()
	for {
		select {
		case <-ctx.Done():
			log.Debug("Stop 1")
			ticker.Stop()
			return
		case <-chEnd:
			log.Debug("Stop 2")
			ticker.Stop()
			return
		case m := <-chErr:
			log.Debugf("Unblocking error occurred retrieving job information due to:%s", m)
			ticker.Stop()
			return
		}
	}
}

func (e *executionCommon) getJobInformation(ctx context.Context, chErr chan error) {
	cmd := fmt.Sprintf("squeue --noheader --name=%s -o \"%%A,%%T\"", e.jobInfo.name)
	output, err := e.client.RunCommand(cmd)
	if err != nil {
		log.Debugf("stderr:%q", output)
		chErr <- errors.Wrap(err, output)
	}
	out := strings.Trim(output, "\" \t\n\x00")
	if out != "" {
		d := strings.Split(out, ",")
		if len(d) != 2 {
			log.Debugf("Unexpected format job information:%q", out)
			chErr <- errors.Errorf("Unexpected format job information:%q", out)
		}
		e.jobInfo.ID = d[0]
		e.jobInfo.state = d[1]
		mess := fmt.Sprintf("Job ID:%s, Job State:%s", e.jobInfo.ID, e.jobInfo.state)
		log.Debugf(mess)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.INFO, e.deploymentID).RegisterAsString(mess)
	}
}

func (e *executionCommon) buildJobInfo(ctx context.Context) error {
	job := jobInfo{}
	// Get main properties from node
	found, jobName, err := deployments.GetNodeProperty(e.kv, e.deploymentID, e.NodeName, "name")
	if err != nil {
		return err
	}
	if !found || jobName == "" {
		jobName = e.cfg.Infrastructures[infrastructureName].GetString("default_job_name")
		if jobName == "" {
			jobName = e.deploymentID
		}
	}
	job.name = jobName

	var tsks = 0
	if _, ts, err := deployments.GetNodeProperty(e.kv, e.deploymentID, e.NodeName, "tasks"); err != nil {
		return err
	} else if ts != "" {
		if tsks, err = strconv.Atoi(ts); err != nil {
			return err
		}
	}
	job.tasks = tsks

	var nodes = 1
	if _, ns, err := deployments.GetNodeProperty(e.kv, e.deploymentID, e.NodeName, "nodes"); err != nil {
		return err
	} else if ns != "" {
		if nodes, err = strconv.Atoi(ns); err != nil {
			return err
		}
	}
	job.nodes = nodes

	var mem = 0
	if _, m, err := deployments.GetNodeProperty(e.kv, e.deploymentID, e.NodeName, "mem_per_node"); err != nil {
		return err
	} else if m != "" {
		if mem, err = strconv.Atoi(m); err != nil {
			return err
		}
	}
	job.mem = mem

	var cpus = 0
	if _, c, err := deployments.GetNodeProperty(e.kv, e.deploymentID, e.NodeName, "cpus_per_task"); err != nil {
		return err
	} else if c != "" {
		if cpus, err = strconv.Atoi(c); err != nil {
			return err
		}
	}
	job.cpus = cpus

	if _, job.maxTime, err = deployments.GetNodeProperty(e.kv, e.deploymentID, e.NodeName, "time"); err != nil {
		return err
	}

	var extraOpts []string
	if _, extra, err := deployments.GetNodeProperty(e.kv, e.deploymentID, e.NodeName, "extra_options"); err != nil {
		return err
	} else if extra != "" {
		if err = json.Unmarshal([]byte(extra), &extraOpts); err != nil {
			return err
		}
	}
	job.opts = extraOpts

	var args []string
	for _, input := range e.EnvInputs {
		if input.Name == "args" && input.Value != "" {
			if err = json.Unmarshal([]byte(input.Value), &args); err != nil {
				return err
			}
		}
	}
	job.execArgs = args
	e.jobInfo = &job
	return nil
}

func (e *executionCommon) runCommand(ctx context.Context) (string, error) {
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

	execFile := path.Join(e.OperationRemoteBaseDir, e.NodeName, e.operation.Name, e.Primary)
	cmd := fmt.Sprintf("srun %s %s %s", opts, execFile, strings.Join(e.jobInfo.execArgs, " "))
	events.WithContextOptionalFields(ctx).NewLogEntry(events.INFO, e.deploymentID).RegisterAsString(fmt.Sprintf("Run the command: %q", cmd))
	output, err := e.client.RunCommand(cmd)
	if err != nil {
		log.Debugf("stderr:%q", output)
		return "", errors.Wrap(err, output)
	}
	return output, nil
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

	if err := g.Wait(); err != nil {
		return err
	}
	return nil
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
	if err != nil {
		return err
	}

	return nil
}

func (e *executionCommon) resolveInputs() error {
	var err error
	e.EnvInputs, e.VarInputsNames, err = operations.ResolveInputsWithInstances(e.kv, e.deploymentID, e.NodeName, e.taskID, e.operation, nil, nil)
	return err
}

func (e *executionCommon) resolveArtifacts() error {
	var err error
	e.Artifacts, err = deployments.GetArtifactsForNode(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}
	log.Debugf("Resolved artifacts: %v", e.Artifacts)
	return nil
}
