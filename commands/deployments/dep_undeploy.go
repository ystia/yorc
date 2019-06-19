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

package deployments

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ystia/yorc/v4/commands/httputil"
)

func init() {
	var purge bool
	var shouldStreamLogs bool
	var shouldStreamEvents bool
	var stopOnError bool
	var undeployCmd = &cobra.Command{
		Use:   "undeploy <DeploymentId>",
		Short: "Undeploy an application",
		Long:  `Undeploy an application specifying the deployment ID.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting a deployment id (got %d parameters)", len(args))
			}
			client, err := httputil.GetClient(ClientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}

			urlStr := "/deployments/" + args[0]

			q := url.Values{}
			if purge {
				q.Add("purge", "true")
			}
			if stopOnError {
				q.Add("stopOnError", "true")
			}
			request, err := client.NewRequest("DELETE", urlStr, nil)
			if err != nil {
				httputil.ErrExit(err)
			}

			request.URL.RawQuery = q.Encode()
			request.Header.Add("Accept", "application/json")
			response, err := client.Do(request)
			if err != nil {
				httputil.ErrExit(err)
			}
			defer response.Body.Close()
			httputil.HandleHTTPStatusCode(response, args[0], "deployment", http.StatusAccepted)

			fmt.Println("Undeployment submitted. In progress...")
			if shouldStreamLogs && !shouldStreamEvents {
				StreamsLogs(client, args[0], !NoColor, false, false)
			} else if !shouldStreamLogs && shouldStreamEvents {
				StreamsEvents(client, args[0], !NoColor, false, false)
			} else if shouldStreamLogs && shouldStreamEvents {
				return errors.Errorf("You can't provide stream-events and stream-logs flags at same time")
			}

			return nil
		},
	}

	DeploymentsCmd.AddCommand(undeployCmd)
	undeployCmd.PersistentFlags().BoolVarP(&purge, "purge", "p", false, "To use if you want to purge instead of undeploy")
	undeployCmd.PersistentFlags().BoolVarP(&stopOnError, "stop-on-error", "", false, "By default if an error occurs during the undeployment, the error is bypassed and the undeployment continues. This flag allows to stop if an error occurs.")
	undeployCmd.PersistentFlags().BoolVarP(&shouldStreamLogs, "stream-logs", "l", false, "Stream logs after undeploying the application. In this mode logs can't be filtered, to use this feature see the \"log\" command.")
	undeployCmd.PersistentFlags().BoolVarP(&shouldStreamEvents, "stream-events", "e", false, "Stream events after undeploying the CSAR.")

}
