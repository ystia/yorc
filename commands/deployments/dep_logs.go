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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ystia/yorc/v3/commands/httputil"
	"github.com/ystia/yorc/v3/events"
	"github.com/ystia/yorc/v3/rest"
)

func init() {
	var fromBeginning bool
	var noStream bool
	var logCmd = &cobra.Command{
		Use:     "logs [<DeploymentId>]",
		Short:   "Stream logs for a deployment or all deployments",
		Long:    `Stream all the logs, or logs for a given deployment id`,
		Aliases: []string{"log"},
		RunE: func(cmd *cobra.Command, args []string) error {
			var deploymentID string
			if len(args) == 1 {
				// One deploymentID is provided
				deploymentID = args[0]
			} else if len(args) == 0 {
				fmt.Println("No deployment id provided, logs for all deployments will be returned")
			} else {
				return errors.Errorf("Expecting one deployment id or none (got %d parameters)", len(args))
			}

			client, err := httputil.GetClient(ClientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}
			colorize := !NoColor

			StreamsLogs(client, deploymentID, colorize, fromBeginning, noStream)
			return nil
		},
	}
	logCmd.PersistentFlags().BoolVarP(&fromBeginning, "from-beginning", "b", false, "Show logs from the beginning of deployments")
	logCmd.PersistentFlags().BoolVarP(&noStream, "no-stream", "n", false, "Show logs then exit. Do not stream logs. It implies --from-beginning")
	DeploymentsCmd.AddCommand(logCmd)
}

// StreamsLogs allows to stream logs
func StreamsLogs(client *httputil.YorcClient, deploymentID string, colorize, fromBeginning, stop bool) {
	if colorize {
		defer color.Unset()
	}
	var lastIdx uint64
	var err error
	var response *http.Response
	var request *http.Request
	if !fromBeginning && !stop {
		if deploymentID != "" {
			response, err = client.Head("/deployments/" + deploymentID + "/logs")
			httputil.HandleHTTPStatusCode(response, deploymentID, "deployment", http.StatusOK)
		} else {
			response, err = client.Head("/logs")
		}
		if err != nil {
			httputil.ErrExit(err)
		}

		// Get last index
		idxHd := response.Header.Get(rest.YorcIndexHeader)
		if idxHd != "" {
			lastIdx, err = strconv.ParseUint(idxHd, 10, 64)
			if err != nil {
				httputil.ErrExit(err)
			}
			fmt.Println("Streaming new logs...")
		} else {
			fmt.Fprint(os.Stderr, "Failed to get latest log index from Yorc, logs will appear from the beginning.")
		}
	}
	var filtersParam string
	for {
		if deploymentID != "" {
			request, err = client.NewRequest("GET", fmt.Sprintf("/deployments/%s/logs?index=%d%s", deploymentID, lastIdx, filtersParam), nil)
		} else {
			request, err = client.NewRequest("GET", fmt.Sprintf("/logs?index=%d%s", lastIdx, filtersParam), nil)
		}
		if err != nil {
			httputil.ErrExit(err)
		}
		request.Header.Add("Accept", "application/json")
		response, err := client.Do(request)
		if err != nil {
			httputil.ErrExit(err)
		}

		if deploymentID != "" {
			httputil.HandleHTTPStatusCode(response, deploymentID, "deployment", http.StatusOK)
		}

		var logs rest.LogsCollection
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			httputil.ErrExit(err)
		}
		err = json.Unmarshal(body, &logs)
		if err != nil {
			httputil.ErrExit(err)
		}

		lastIdx = logs.LastIndex
		for _, log := range logs.Logs {
			if colorize {
				fmt.Printf("%s\n", color.CyanString("%s", format(log)))
			} else {
				fmt.Printf("%s\n", format(log))
			}
		}

		response.Body.Close()

		if stop {
			return
		}
	}
}

func format(log json.RawMessage) string {
	var data map[string]interface{}
	err := json.Unmarshal(log, &data)
	if err != nil {
		httputil.ErrExit(err)
	}
	return events.FormatLog(data)
}
