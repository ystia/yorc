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
	"os"
	"strconv"

	"net/http"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/ystia/yorc/commands/httputil"
	"github.com/ystia/yorc/rest"
)

func init() {
	var fromBeginning bool
	var noStream bool
	var eventCmd = &cobra.Command{
		Use:     "events [<DeploymentId>]",
		Short:   "Stream events for a deployment or all deployments",
		Long:    `Stream all the events, or events for a given deployment id`,
		Aliases: []string{"events"},
		RunE: func(cmd *cobra.Command, args []string) error {
			var deploymentID string
			if len(args) == 1 {
				// One deploymentID is provided
				deploymentID = args[0]
			} else if len(args) == 0 {
				fmt.Println("No deployment id provided, events for all deployments will be returned")
			} else {
				return errors.Errorf("Expecting one deployment id or none (got %d parameters)", len(args))
			}

			client, err := httputil.GetClient(ClientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}
			colorize := !NoColor

			StreamsEvents(client, deploymentID, colorize, fromBeginning, noStream)
			return nil
		},
	}
	eventCmd.PersistentFlags().BoolVarP(&fromBeginning, "from-beginning", "b", false, "Show events from the beginning of deployments")
	eventCmd.PersistentFlags().BoolVarP(&noStream, "no-stream", "n", false, "Show events then exit. Do not stream events. It implies --from-beginning")
	DeploymentsCmd.AddCommand(eventCmd)
}

// StreamsEvents allows to stream events
func StreamsEvents(client *httputil.YorcClient, deploymentID string, colorize, fromBeginning, stop bool) {
	if colorize {
		defer color.Unset()
	}
	var lastIdx uint64
	var err error
	var response *http.Response
	var request *http.Request
	if !fromBeginning && !stop {
		if deploymentID != "" {
			response, err = client.Head("/deployments/" + deploymentID + "/events")
			httputil.HandleHTTPStatusCode(response, deploymentID, "deployment", http.StatusOK)
		} else {
			response, err = client.Head("/events")
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
			fmt.Println("Streaming new events...")
		} else {
			fmt.Fprint(os.Stderr, "Failed to get latest events index from Yorc, events will appear from the beginning.")
		}
	}
	for {
		if deploymentID != "" {
			request, err = client.NewRequest("GET", fmt.Sprintf("/deployments/%s/events?index=%d", deploymentID, lastIdx), nil)
		} else {
			request, err = client.NewRequest("GET", fmt.Sprintf("/events?index=%d", lastIdx), nil)
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

		var evts rest.EventsCollection
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			httputil.ErrExit(err)
		}
		err = json.Unmarshal(body, &evts)
		if err != nil {
			httputil.ErrExit(err)
		}
		if lastIdx == evts.LastIndex {
			continue
		}
		lastIdx = evts.LastIndex
		for _, event := range evts.Events {
			if colorize {
				fmt.Printf("%s\n", color.MagentaString("%s", format(event)))
			} else {
				fmt.Printf("%s\n", format(event))
			}
		}
		response.Body.Close()
		if stop {
			return
		}
	}
}
