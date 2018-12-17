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
	"encoding/json"
	"fmt"
	"io/ioutil"

	"net/http"
	"os"
	"path"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/ystia/yorc/commands/deployments"
	"github.com/ystia/yorc/commands/httputil"
	"github.com/ystia/yorc/helper/tabutil"
	"github.com/ystia/yorc/rest"
	"github.com/ystia/yorc/tasks"
)

func init() {
	var withSteps bool
	var infoTaskCmd = &cobra.Command{
		Use:   "info <DeploymentId> <TaskId>",
		Short: "Get information about a deployment task",
		Long:  `Display information about a given task specifying the deployment id and the task id.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return errors.Errorf("Expecting a deployment id and a task id (got %d parameters)", len(args))
			}
			client, err := httputil.GetClient(deployments.ClientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}

			url := path.Join("/deployments", args[0], "/tasks/", args[1])
			request, err := client.NewRequest("GET", url, nil)
			if err != nil {
				httputil.ErrExit(err)
			}

			request.Header.Add("Accept", "application/json")
			response, err := client.Do(request)
			if err != nil {
				httputil.ErrExit(err)
			}
			defer response.Body.Close()
			ids := args[0] + "/" + args[1]
			httputil.HandleHTTPStatusCode(response, ids, "deployment/task", http.StatusOK)
			var task rest.Task
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				httputil.ErrExit(err)
			}
			err = json.Unmarshal(body, &task)
			if err != nil {
				httputil.ErrExit(err)
			}
			fmt.Println("Task: ", task.ID)
			fmt.Println("Task status:", task.Status)
			fmt.Println("Task type:", task.Type)

			if withSteps {
				displayStepTables(client, args)
			}

			return nil
		},
	}

	infoTaskCmd.PersistentFlags().BoolVarP(&withSteps, "steps", "w", false, "Show steps of the related workflow associated to the task")
	tasksCmd.AddCommand(infoTaskCmd)
}

func displayStepTables(client *httputil.YorcClient, args []string) {
	colorize := !deployments.NoColor
	if colorize {
		commErrorMsg = color.New(color.FgHiRed, color.Bold).SprintFunc()(commErrorMsg)
	}
	request, err := client.NewRequest("GET", "/deployments/"+args[0]+"/tasks/"+args[1]+"/steps", nil)
	if err != nil {
		httputil.ErrExit(err)
	}
	request.Header.Add("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		httputil.ErrExit(err)
	}
	defer response.Body.Close()
	httputil.HandleHTTPStatusCode(response, args[0], "step", http.StatusOK)
	var steps []tasks.TaskStep
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		httputil.ErrExit(err)
	}
	err = json.Unmarshal(body, &steps)
	if err != nil {
		httputil.ErrExit(err)
	}
	if colorize {
		defer color.Unset()
	}
	fmt.Println("Steps:")
	tasksTable := tabutil.NewTable()
	tasksTable.AddHeaders("Name", "Status")
	errs := make([]error, 0)
	for _, step := range steps {
		tasksTable.AddRow(step.Name, getColoredTaskStepStatus(colorize, step.Status))
	}
	fmt.Println(tasksTable.Render())
	if len(errs) > 0 {
		fmt.Fprintln(os.Stderr, "\n\nErrors encountered:")
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, "###################\n", err)
		}
	}
}

func getColoredTaskStepStatus(colorize bool, status string) string {
	if !colorize {
		return status
	}
	switch strings.ToLower(status) {
	case "error":
		return color.New(color.FgHiRed, color.Bold).SprintFunc()(status)
	case "canceled", "running":
		return color.New(color.FgHiYellow, color.Bold).SprintFunc()(status)
	case "done":
		return color.New(color.FgHiGreen, color.Bold).SprintFunc()(status)
	default:
		return status
	}
}
