// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package operations

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/tasks"
)

func purgePreChecks(ctx context.Context, deploymentID string) error {
	status, err := deployments.GetDeploymentStatus(ctx, deploymentID)
	if err != nil {
		return err
	}
	if status != deployments.UNDEPLOYED {
		return errors.Errorf("can't purge a deployment not in %q state, actual status is %q", deployments.UNDEPLOYED, status)
	}
	return nil
}

func purgeTasks(ctx context.Context, deploymentID string, force bool, ignoreTasks ...string) error {
	var finalError *multierror.Error
	// Remove from KV all tasks from the current target deployment, except ignoredTasks tasks (generally the purge task when done asynchronously)
	tasksList, err := deployments.GetDeploymentTaskList(ctx, deploymentID)
	finalError = multierror.Append(finalError, err)
	for _, tid := range tasksList {

		if collections.ContainsString(ignoreTasks, tid) {
			continue
		}
		err = tasks.DeleteTask(tid)
		finalError = multierror.Append(finalError, err)
		err = consulutil.Delete(path.Join(consulutil.WorkflowsPrefix, tid)+"/", true)
		finalError = multierror.Append(finalError, err)

	}
	return finalError.ErrorOrNil()
}

func handleError(merr *multierror.Error, continueOnError bool, f func() error) error {
	err := f()
	multierror.Append(merr, err)
	if continueOnError {
		return nil
	}
	return merr.ErrorOrNil()
}

// PurgeDeployment allows to completely remove references of a deployment within yorc
//
// The deployment should be in UDEPLOYED status to be purged except if force parameter is set to true.
// In this case the deployment status is not even checked.
//
// If continueOnError is set to true, purge do not stop on errors and try to delete the maximum of elements while normal purge stops on the first error.
// The error returned by this function may be multi-evaluated use the standard errors.Unwrap method to access individual errors.
// In any case if the purge process encounter an error the deployment status is set to PURGE_FAILED
//
// ignoreTasks allows to prevent removing a given list of tasks this is particularly useful when calling it within a task.
// This option will probably be transitory for Yorc 4.x before switching to a full synchronous purge model
func PurgeDeployment(ctx context.Context, deploymentID, filepathWorkingDirectory string, force, continueOnError bool, ignoreTasks ...string) error {

	finalError := new(multierror.Error)

	if !force {
		err := purgePreChecks(ctx, deploymentID)
		if err != nil {
			finalError = multierror.Append(finalError, err)
			return finalError
		}
	}

	// Set status to PURGE_IN_PROGRESS
	err := deployments.SetDeploymentStatus(ctx, deploymentID, deployments.PURGE_IN_PROGRESS)
	if err != nil {
		if !continueOnError {
			finalError = multierror.Append(finalError, err)
			return finalError
		}
		// In force mode this error could be ignored
	}

	err = performPurgeDeployment(ctx, finalError, deploymentID, filepathWorkingDirectory, continueOnError, ignoreTasks...)
	if err != nil {
		ensurePurgeFailedStatus(ctx, deploymentID)
	}
	return err
}

// Will try to set the deployment status to purge_failed even if the deployment doesn't exist.
func ensurePurgeFailedStatus(ctx context.Context, deploymentID string) {
	err := deployments.SetDeploymentStatus(ctx, deploymentID, deployments.PURGE_FAILED)
	if err != nil {
		if deployments.IsDeploymentNotFoundError(err) {
			_, err := consulutil.GetKV().Put(&api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "status"), Value: []byte(deployments.PURGE_FAILED.String())}, nil)
			if err == nil {
				events.PublishAndLogDeploymentStatusChange(ctx, deploymentID, strings.ToLower(deployments.PURGE_FAILED.String()))
			}
		}
	}
}

func performPurgeDeployment(ctx context.Context, finalError *multierror.Error, deploymentID, filepathWorkingDirectory string, continueOnError bool, ignoreTasks ...string) error {

	err := handleError(finalError, continueOnError, func() error {
		return purgeTasks(ctx, deploymentID, continueOnError, ignoreTasks...)
	})
	if err != nil {
		return err
	}

	// Delete events tree corresponding to the deployment TaskExecution
	err = handleError(finalError, continueOnError, func() error {
		return events.PurgeDeploymentEvents(ctx, deploymentID)
	})
	if err != nil {
		return err
	}
	// Delete logs tree corresponding to the deployment
	err = handleError(finalError, continueOnError, func() error {
		return events.PurgeDeploymentLogs(ctx, deploymentID)
	})
	if err != nil {
		return err
	}
	// Remove the working directory of the current target deployment
	overlayPath := filepath.Join(filepathWorkingDirectory, "deployments", deploymentID)
	err = handleError(finalError, continueOnError, func() error {
		err := os.RemoveAll(overlayPath)
		return errors.Wrapf(err, "failed to remove deployments artifacts stored on disk: %q", overlayPath)
	})
	if err != nil {
		return err
	}

	err = handleError(finalError, continueOnError, func() error {
		return deployments.DeleteDeployment(ctx, deploymentID)
	})
	if err != nil {
		return err
	}

	// Ensure this is effectively deleted in case of force (may not be the case de store deployment deletion fails in previous function)
	err = handleError(finalError, continueOnError, func() error {
		return consulutil.Delete(path.Join(consulutil.DeploymentKVPrefix, deploymentID)+"/", true)
	})
	if err != nil {
		return err
	}

	return finalError.ErrorOrNil()
}
