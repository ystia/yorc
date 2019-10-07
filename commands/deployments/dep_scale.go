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
	"path"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ystia/yorc/v4/commands/httputil"
	"github.com/ystia/yorc/v4/log"
)

func init() {
	var shouldStreamLogs bool
	var shouldStreamEvents bool
	var nodeName string
	var instancesDelta int32
	var scaleCmd = &cobra.Command{
		Use:   "scale <id>",
		Short: "Scale a node",
		Long:  `Scale a given node of a deployment <id> by adding or removing the specified number of instances.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting a deployment id (got %d parameters)", len(args))
			}

			if nodeName == "" {
				return errors.New("Missing mandatory \"node\" flag")
			}

			if instancesDelta == 0 {
				return errors.New("Missing non-zero \"delta\" flag")
			}

			client, err := httputil.GetClient(ClientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}
			deploymentID := args[0]

			location, err := postScalingRequest(client, deploymentID, nodeName, instancesDelta)
			if err != nil {
				return err
			}

			fmt.Println("Scaling request submitted. Task Id:", path.Base(location))
			if shouldStreamLogs && !shouldStreamEvents {
				StreamsLogs(client, deploymentID, !NoColor, false, false)
			} else if !shouldStreamLogs && shouldStreamEvents {
				StreamsEvents(client, deploymentID, !NoColor, false, false)
			} else if shouldStreamLogs && shouldStreamEvents {
				return errors.Errorf("You can't provide stream-events and stream-logs flags at same time")
			}
			return nil
		},
	}
	scaleCmd.PersistentFlags().StringVarP(&nodeName, "node", "n", "", "The name of the node that should be scaled.")
	scaleCmd.PersistentFlags().Int32VarP(&instancesDelta, "delta", "d", 0, "The non-zero number of instance to add (if > 0) or remove (if < 0).")
	scaleCmd.PersistentFlags().BoolVarP(&shouldStreamLogs, "stream-logs", "l", false, "Stream logs after issuing the scaling request. In this mode logs can't be filtered, to use this feature see the \"log\" command.")
	scaleCmd.PersistentFlags().BoolVarP(&shouldStreamEvents, "stream-events", "e", false, "Stream events after  issuing the scaling request.")
	DeploymentsCmd.AddCommand(scaleCmd)
}

func postScalingRequest(client httputil.HTTPClient, deploymentID, nodeName string, instancesDelta int32) (string, error) {
	request, err := client.NewRequest("POST", path.Join("/deployments", deploymentID, "scale", nodeName), nil)
	if err != nil {
		httputil.ErrExit(errors.Wrap(err, httputil.YorcAPIDefaultErrorMsg))
	}

	query := request.URL.Query()
	query.Set("delta", strconv.Itoa(int(instancesDelta)))

	request.URL.RawQuery = query.Encode()

	log.Debugf("POST: %s", request.URL.String())

	response, err := client.Do(request)
	if err != nil {
		httputil.ErrExit(errors.Wrap(err, httputil.YorcAPIDefaultErrorMsg))
	}
	defer response.Body.Close()

	ids := deploymentID + "/" + nodeName
	httputil.HandleHTTPStatusCode(response, ids, "deployment/node", http.StatusAccepted)
	location := response.Header.Get("Location")
	if location == "" {
		return "", errors.New("No \"Location\" header returned in Yorc response")
	}
	return location, nil
}
