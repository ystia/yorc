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

package tasks

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ystia/yorc/v4/commands/deployments"
	"github.com/ystia/yorc/v4/commands/httputil"
	"github.com/ystia/yorc/v4/tasks"
)

func init() {
	tasksCmd.AddCommand(updateTaskStepStateCmd)
}

func updateTaskStepState(deploymentID, taskID, stepName, statusStr string) {
	client, err := httputil.GetClient(deployments.ClientConfig)
	if err != nil {
		httputil.ErrExit(err)
	}

	statusStr = strings.ToLower(statusStr)

	// Safety check
	_, err = tasks.ParseTaskStepStatus(statusStr)
	if err != nil {
		httputil.ErrExit(err)
	}

	// The task step status is set to "done"
	step := &tasks.TaskStep{Status: statusStr}
	body, err := json.Marshal(step)
	if err != nil {
		log.Panic(err)
	}

	url := path.Join("/deployments", deploymentID, "tasks", taskID, "steps", stepName)
	request, err := client.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		httputil.ErrExit(err)
	}

	request.Header.Add("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		httputil.ErrExit(err)
	}
	defer response.Body.Close()
	ids := deploymentID + "/" + taskID
	httputil.HandleHTTPStatusCode(response, ids, "deployment/task/step", http.StatusOK)
}

var updateTaskStepStateCmd = &cobra.Command{
	Use:     "update-step-state <DeploymentId> <TaskId> <StepName> <state>",
	Aliases: []string{"update-state", "update-step", "uss", "us"},
	Short:   "Update a deployment task step on error",
	Args:    cobra.ExactArgs(4),
	Long: `Update a task step specifying the deployment id, the task id, the step name and a valid state for the step.
	The task step must be on error to be update.
	This allows to bypass or retry errors happening in a workflow. So accepted state are either DONE or INITIAL.
	Use 'yorc deployment tasks resume' command to restart a workflow after having updated it's errors`,
	Run: func(cmd *cobra.Command, args []string) {
		updateTaskStepState(args[0], args[1], args[2], args[3])
	},
}
