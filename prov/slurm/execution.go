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
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//FIXME need to put code in common with ansible execution

type execution interface {
	resolveExecution() error
	execute(ctx context.Context) error
}

type operationNotImplemented struct {
	msg string
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
	NodeTypePath           string
	OperationPath          string
	Primary                string
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
	e.NodeTypePath = path.Join(consulutil.DeploymentKVPrefix, e.deploymentID, "topology/types", e.NodeType)

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
	return nil
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
	switch strings.ToLower(e.operation.Name) {
	case "tosca.interfaces.node.lifecycle.runnable.run":
		log.Printf("Running the job: %s", e.operation.Name)
		// Copy the artifacts
		if err := e.uploadArtifacts(ctx); err != nil {
			return err
		}
		// Run the command
		out, err := e.runScript(ctx)
		if err != nil {
			events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, e.deploymentID).RegisterAsString(err.Error())
			return err
		}
		log.Debugf("output:%q", out)
		events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, e.deploymentID).RegisterAsString(out)
		return e.cleanUp()
	default:
		return errors.Errorf("Unsupported operation %q", e.operation.Name)
	}
	return nil
}

func (e *executionCommon) cleanUp() error {
	log.Debugf("Cleanup the operation remote base directory")
	cmd := fmt.Sprintf("rm -rf %s", e.OperationRemoteBaseDir)
	_, err := e.client.RunCommand(cmd)
	if err != nil {
		return err
	}
	return nil
}

func (e *executionCommon) runScript(ctx context.Context) (string, error) {
	scriptPath := path.Join(e.OperationRemoteBaseDir, e.NodeName, e.operation.Name, e.Primary)
	dirToRunCmd := path.Dir(scriptPath)
	script := path.Base(scriptPath)
	cmd := fmt.Sprintf("cd %s && source %s", dirToRunCmd, script)
	output, err := e.client.RunCommand(cmd)
	if err != nil {
		log.Debugf("stderr:%q", output)
		return "", errors.Wrap(err, output)
	}
	return output, nil
}

func (e *executionCommon) uploadArtifacts(ctx context.Context) error {
	log.Debugf("Upload artifacts to remote host")
	var g errgroup.Group
	for _, artPath := range e.Artifacts {
		log.Debugf("handle artifact with path:%q", artPath)
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
				return e.handleFile(ctx, sourcePath, sourcePath)
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
			return e.handleFile(ctx, pathFile, artifactBaseDir)
		}
		return nil
	})
}

func (e *executionCommon) handleFile(ctx context.Context, pathFile, artifactBaseDir string) error {
	log.Debugf("handle file with path:%q", pathFile)
	relPath, err := filepath.Rel(artifactBaseDir, pathFile)
	if err != nil {
		return err
	}

	source, err := ioutil.ReadFile(pathFile)
	if err != nil {
		return err
	}

	remotePath := path.Join(e.OperationRemoteBaseDir, e.NodeName, e.operation.Name, relPath)
	return e.uploadFile(ctx, bytes.NewReader(e.substituteInputs(source)), remotePath)
}

func (e *executionCommon) substituteInputs(input []byte) []byte {
	log.Debugf("Substitute input var in artifact files")
	for _, envInput := range e.EnvInputs {
		input = bytes.Replace(input, []byte("${"+envInput.Name+"}"), []byte(envInput.Value), -1)
	}
	return input
}

func (e *executionCommon) uploadFile(ctx context.Context, source io.Reader, remotePath string) error {
	if err := e.client.CopyFile(source, remotePath, "0755"); err != nil {
		log.Debugf("an error occurred:%+v", err)
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
