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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ystia/yorc/commands/deployments"
	"github.com/ystia/yorc/commands/httputil"
	"github.com/ystia/yorc/tasks"
)

func init() {
	tasksCmd.AddCommand(updateTaskStepCmd)
}

var updateTaskStepCmd = &cobra.Command{
	Use:   "fix <DeploymentId> <TaskId> <StepName>",
	Short: "Fix a deployment task step on error",
	Long: `Fix a task step specifying the deployment id, the task id and the step name.
	The task step must be on error to be fixed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 3 {
			return errors.Errorf("Expecting a deployment id, a task id and a step name(got %d parameters)", len(args))
		}
		client, err := httputil.GetClient(deployments.ClientConfig)
		if err != nil {
			httputil.ErrExit(err)
		}

		// The task step status is set to "done"
		step := &tasks.TaskStep{Name: args[2], Status: strings.ToLower(tasks.TaskStepStatusDONE.String())}
		body, err := json.Marshal(step)
		if err != nil {
			log.Panic(err)
		}

		url := path.Join("/deployments", args[0], "tasks", args[1], "steps", args[2])
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
		ids := args[0] + "/" + args[1]
		httputil.HandleHTTPStatusCode(response, ids, "deployment/task/step", http.StatusOK)
		return nil
	},
}
