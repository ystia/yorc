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

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ystia/yorc/v3/commands/deployments"
	"github.com/ystia/yorc/v3/commands/httputil"
	"github.com/ystia/yorc/v3/helper/tabutil"
	"github.com/ystia/yorc/v3/rest"
	"github.com/ystia/yorc/v3/tasks"
)

func init() {
	deployments.DeploymentsCmd.AddCommand(tasksCmd)
}

var commErrorMsg = httputil.YorcAPIDefaultErrorMsg
var tasksCmd = &cobra.Command{
	Use:   "tasks <DeploymentId>",
	Short: "List tasks of a deployment",
	Long: `Display info about the tasks related to a given deployment.
    It prints the tasks ID, type and status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.Errorf("Expecting a deployment id (got %d parameters)", len(args))
		}
		client, err := httputil.GetClient(deployments.ClientConfig)
		if err != nil {
			httputil.ErrExit(err)
		}
		colorize := !deployments.NoColor
		if colorize {
			commErrorMsg = color.New(color.FgHiRed, color.Bold).SprintFunc()(commErrorMsg)
		}
		request, err := client.NewRequest("GET", "/deployments/"+args[0], nil)
		if err != nil {
			httputil.ErrExit(err)
		}
		request.Header.Add("Accept", "application/json")
		response, err := client.Do(request)
		if err != nil {
			httputil.ErrExit(err)
		}
		defer response.Body.Close()
		httputil.HandleHTTPStatusCode(response, args[0], "deployment", http.StatusOK)
		var dep rest.Deployment
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			httputil.ErrExit(err)
		}
		err = json.Unmarshal(body, &dep)
		if err != nil {
			httputil.ErrExit(err)
		}
		if colorize {
			defer color.Unset()
		}
		fmt.Println("Tasks:")
		tasksTable := tabutil.NewTable()
		tasksTable.AddHeaders("Id", "Type", "Status")
		errs := make([]error, 0)
		for _, atomLink := range dep.Links {
			if atomLink.Rel == rest.LinkRelTask {
				var task rest.Task
				err = httputil.GetJSONEntityFromAtomGetRequest(client, atomLink, &task)
				if err != nil {
					errs = append(errs, err)
					tasksTable.AddRow(path.Base(atomLink.Href), commErrorMsg)
					continue
				}
				// Ignore TaskTypeAction
				if tasks.TaskTypeAction.String() != task.Type {
					tasksTable.AddRow(task.ID, task.Type, deployments.GetColoredTaskStatus(colorize, task.Status))
				}
			}
		}
		fmt.Println(tasksTable.Render())
		if len(errs) > 0 {
			fmt.Fprintln(os.Stderr, "\n\nErrors encountered:")
			for _, err := range errs {
				fmt.Fprintln(os.Stderr, "###################\n", err)
			}
		}
		return nil
	},
}
