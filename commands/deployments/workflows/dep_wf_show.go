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

package workflows

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ystia/yorc/v4/commands/deployments"
	"github.com/ystia/yorc/v4/commands/httputil"
	"github.com/ystia/yorc/v4/rest"
)

func init() {
	var workflowName string
	var wfShowCmd = &cobra.Command{
		Use:     "show <id>",
		Short:   "Show a human readable textual representation of a given TOSCA workflow.",
		Aliases: []string{"display", "sh", "disp"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting an id (got %d parameters)", len(args))
			}
			client, err := httputil.GetClient(deployments.ClientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}
			if workflowName == "" {
				return errors.New("Missing mandatory \"workflow-name\" parameter")
			}
			request, err := client.NewRequest("GET", fmt.Sprintf("/deployments/%s/workflows/%s", args[0], workflowName), nil)
			if err != nil {
				httputil.ErrExit(err)
			}
			request.Header.Add("Accept", "application/json")
			response, err := client.Do(request)
			if err != nil {
				httputil.ErrExit(err)
			}
			defer response.Body.Close()
			ids := args[0] + "/" + workflowName
			httputil.HandleHTTPStatusCode(response, ids, "deployment/workflow", http.StatusOK)

			var wf rest.Workflow
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				httputil.ErrExit(err)
			}
			err = json.Unmarshal(body, &wf)
			if err != nil {
				httputil.ErrExit(err)
			}
			fmt.Printf("Workflow %s:\n", workflowName)
			for stepName, step := range wf.Steps {
				fmt.Printf("  Step %s:\n", stepName)
				if step.Target != "" {
					fmt.Println("    Target:", step.Target)
				}
				fmt.Println("    Activities:")
				for _, activity := range step.Activities {
					if activity.CallOperation != nil {
						fmt.Println("      - Call Operation:", activity.CallOperation.Operation)
					}
					if activity.Delegate != nil {
						fmt.Println("      - Delegate:", activity.Delegate.Workflow)
					}
					if activity.SetState != "" {
						fmt.Println("      - Set State:", activity.SetState)
					}
					if activity.Inline != nil {
						fmt.Println("      - Inline:", activity.Inline.Workflow)
					}
				}
				if len(step.OnSuccess) > 0 {
					fmt.Println("    On Success:")
					for _, next := range step.OnSuccess {
						fmt.Println("      -", next)
					}
				}
			}
			return nil
		},
	}
	wfShowCmd.PersistentFlags().StringVarP(&workflowName, "workflow-name", "w", "", "The workflows name")
	workflowsCmd.AddCommand(wfShowCmd)
}
