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
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/stringutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tasks"
	"path"
	"strings"
)

type executionSingularity struct {
	*executionCommon
	singularityInfo *singularityInfo
}

func (e *executionSingularity) execute(ctx context.Context) (err error) {
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

		// Build singularity information
		if err := e.buildSingularityInfo(ctx); err != nil {
			return errors.Wrap(err, "failed to build singularity information")
		}

		// Run the command
		out, err := e.runCommand(ctx)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, e.deploymentID).RegisterAsString(err.Error())
			return errors.Wrap(err, "failed to run command")
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.INFO, e.deploymentID).RegisterAsString(out)
		log.Debugf("output:%q", out)
		return e.cleanUp()
	default:
		return errors.Errorf("Unsupported operation %q", e.operation.Name)
	}
	return nil
}

func (e *executionSingularity) runCommand(ctx context.Context) (string, error) {
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

	stopCh := make(chan struct{})
	errCh := make(chan error)
	go e.pollInteractiveJobInfo(ctx, stopCh, errCh)

	cmd := fmt.Sprintf("srun singularity %s %s %s", e.singularityInfo.command, e.singularityInfo.imageURI, e.singularityInfo.exec)
	events.WithContextOptionalFields(ctx).NewLogEntry(events.INFO, e.deploymentID).RegisterAsString(fmt.Sprintf("Run the command: %q", cmd))
	output, err := e.client.RunCommand(cmd)
	if err != nil {
		log.Debugf("stderr:%q", output)
		return "", errors.Wrap(err, output)
	}
	// Stop polling information in interactive mode
	close(stopCh)
	return output, nil
}

func (e *executionSingularity) buildSingularityInfo(ctx context.Context) error {
	singularityInfo := singularityInfo{}
	for _, input := range e.EnvInputs {
		if input.Name == "container_image" && input.Value != "" {
			singularityInfo.imageName = input.Value
		}

		if input.Name == "exec_command" && input.Value != "" {
			singularityInfo.exec = input.Value
			singularityInfo.command = "exec"
		}
	}
	// imageName is mandatory
	if singularityInfo.imageName == "" {
		return errors.Errorf("Missing mandatory input 'container_image' for %s", e.NodeName)
	}
	// Default singularity command is "run"
	if singularityInfo.command == "" {
		singularityInfo.command = "run"
	}
	log.Debugf("singularity Info:%+v", singularityInfo)
	e.singularityInfo = &singularityInfo
	return nil
}
