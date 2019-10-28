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
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tasks"
	"github.com/ystia/yorc/v4/tosca"
)

type executionSingularity struct {
	*executionCommon
	imageURI       string
	commandOptions []string
	debug          bool
}

func (e *executionSingularity) execute(ctx context.Context) error {
	// Only runnable operation is currently supported
	log.Debugf("Execute the operation:%+v", e.operation)
	// Fill log optional fields for log registration
	switch strings.ToLower(e.operation.Name) {
	case strings.ToLower(tosca.RunnableSubmitOperationName):
		log.Printf("Submit the job: %s", e.operation.Name)
		if e.Primary == "" {
			return errors.New("Image artifact is mandatory and must be filled in the operation implementation")
		}
		// Build Job Information
		if err := e.buildJobInfo(ctx); err != nil {
			return errors.Wrap(err, "failed to build job information")
		}
		// Build singularity information
		if err := e.resolveImageURI(ctx); err != nil {
			return errors.Wrap(err, "failed to resolve singularity image URI")
		}
		// Retrieve singularity job props
		if err := e.getSingularityProps(); err != nil {
			return errors.Wrap(err, "failed to retrieve singularity command options")
		}
		// Copy the artifacts
		if err := e.uploadArtifacts(ctx); err != nil {
			return errors.Wrap(err, "failed to upload artifact")
		}
		err := e.prepareAndSubmitSingularityJob(ctx)
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
		err = deployments.SetAttributeForAllInstances(e.deploymentID, e.NodeName, "job_id", e.jobInfo.ID)
		if err != nil {
			return errors.Wrap(err, "failed to retrieve job id an manual cleanup may be necessary: ")
		}
	case strings.ToLower(tosca.RunnableCancelOperationName):
		jobInfo, err := e.getJobInfoFromTaskContext()
		if err != nil {
			return err
		}
		return cancelJobID(jobInfo.ID, e.client)
	default:
		return errors.Errorf("Unsupported operation %q", e.operation.Name)
	}
	return nil
}

func (e *executionSingularity) prepareAndSubmitSingularityJob(ctx context.Context) error {
	var debug, inner string
	if e.debug {
		debug = "-d -v"
	}
	cmdOpts := strings.Join(e.commandOptions, " ")
	if e.jobInfo.ExecutionOptions.Command != "" {
		inner = fmt.Sprintf("srun singularity %s exec %s %s %s %s", debug, cmdOpts, e.imageURI, e.jobInfo.ExecutionOptions.Command, quoteArgs(e.jobInfo.ExecutionOptions.Args))
	} else {
		inner = fmt.Sprintf("srun singularity %s run %s %s", debug, cmdOpts, e.imageURI)
	}
	cmd, err := e.wrapCommand(inner)
	if err != nil {
		return err
	}
	return e.submitJob(ctx, cmd)
}

func (e *executionSingularity) resolveImageURI(ctx context.Context) error {
	switch {
	// Docker image
	case strings.HasPrefix(e.Primary, "docker://"):
		if err := e.buildImageURI(ctx, "docker://"); err != nil {
			return err
		}
	// Singularity image
	case strings.HasPrefix(e.Primary, "shub://"):
		if err := e.buildImageURI(ctx, "shub://"); err != nil {
			return err
		}
	// File image
	case strings.HasSuffix(e.Primary, ".simg") || strings.HasSuffix(e.Primary, ".img"):
		e.imageURI = e.Primary
	default:
		return errors.Errorf("Unable to resolve image URI from image with name:%q", e.Primary)
	}
	return nil
}

func (e *executionSingularity) buildImageURI(ctx context.Context, prefix string) error {
	repoName, err := deployments.GetOperationImplementationRepository(e.deploymentID, e.operation.ImplementedInNodeTemplate, e.NodeType, e.operation.Name)
	if err != nil {
		return err
	}
	if repoName == "" {
		e.imageURI = e.Primary
	} else {
		repoURL, err := deployments.GetRepositoryURLFromName(e.deploymentID, repoName)
		if err != nil {
			return err
		}
		// Just ignore default public Docker and Singularity registries
		if repoURL == deployments.DockerHubURL || repoURL == deployments.SingularityHubURL {
			e.imageURI = e.Primary
		} else if repoURL != "" {
			urlStruct, err := url.Parse(repoURL)
			if err != nil {
				return err
			}
			tabs := strings.Split(e.Primary, prefix)
			imageURI := prefix + path.Join(urlStruct.Host, tabs[1])
			log.Debugf("imageURI:%q", imageURI)
			e.imageURI = imageURI
		} else {
			e.imageURI = e.Primary
		}
	}
	return nil
}

func (e *executionSingularity) getSingularityProps() error {
	var err error
	if o, err := deployments.GetNodePropertyValue(e.deploymentID, e.NodeName, "singularity_command_options"); err != nil {
		return err
	} else if o != nil && o.RawString() != "" {
		if err = json.Unmarshal([]byte(o.RawString()), &e.commandOptions); err != nil {
			return err
		}
	}
	if e.debug, err = deployments.GetBooleanNodeProperty(e.deploymentID, e.NodeName, "singularity_debug"); err != nil {
		return err
	}
	return nil
}
