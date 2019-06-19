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
	"bytes"
	"fmt"
	"net/http"
	"path"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ystia/yorc/v4/commands/deployments"
	"github.com/ystia/yorc/v4/commands/httputil"
)

func init() {
	var shouldStreamLogs bool
	var shouldStreamEvents bool
	var continueOnError bool
	var workflowName string
	var jsonParam string
	var wfExecCmd = &cobra.Command{
		Use:     "execute <id>",
		Short:   "Trigger a custom workflow on deployment <id>",
		Aliases: []string{"exec"},
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
			url := fmt.Sprintf("/deployments/%s/workflows/%s", args[0], workflowName)
			if continueOnError {
				url = url + "?continueOnError"
			}
			var request *http.Request
			if len(jsonParam) == 0 {
				request, err = client.NewRequest("POST", url, nil)
			} else {
				request, err = client.NewRequest("POST", url, bytes.NewBuffer([]byte(jsonParam)))
			}
			if err != nil {
				httputil.ErrExit(err)
			}
			request.Header.Add("Content-Type", "application/json")
			response, err := client.Do(request)
			defer response.Body.Close()
			if err != nil {
				httputil.ErrExit(err)
			}
			ids := args[0] + "/" + workflowName
			httputil.HandleHTTPStatusCode(response, ids, "deployment/workflow", http.StatusAccepted, http.StatusCreated)

			fmt.Println("New task ", path.Base(response.Header.Get("Location")), " created to execute ", workflowName)
			if shouldStreamLogs && !shouldStreamEvents {
				deployments.StreamsLogs(client, args[0], !deployments.NoColor, false, false)
			} else if !shouldStreamLogs && shouldStreamEvents {
				deployments.StreamsEvents(client, args[0], !deployments.NoColor, false, false)
			} else if shouldStreamLogs && shouldStreamEvents {
				return errors.Errorf("You can't provide stream-events and stream-logs flags at same time")
			}
			return nil
		},
	}
	wfExecCmd.PersistentFlags().StringVarP(&workflowName, "workflow-name", "w", "", "The workflows name (mandatory)")
	wfExecCmd.PersistentFlags().BoolVarP(&continueOnError, "continue-on-error", "", false, "By default if an error occurs in a step of a workflow then other running steps are cancelled and the workflow is stopped. This flag allows to continue to the next steps even if an error occurs.")
	wfExecCmd.PersistentFlags().StringVarP(&jsonParam, "data", "d", "", "Provide the JSON format for the node instances selection")
	wfExecCmd.PersistentFlags().BoolVarP(&shouldStreamLogs, "stream-logs", "l", false, "Stream logs after triggering a workflow. In this mode logs can't be filtered, to use this feature see the \"log\" command.")
	wfExecCmd.PersistentFlags().BoolVarP(&shouldStreamEvents, "stream-events", "e", false, "Stream events after triggering a workflow.")
	workflowsCmd.AddCommand(wfExecCmd)
}
