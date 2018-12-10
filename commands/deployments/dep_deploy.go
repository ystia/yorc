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
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/ystia/yorc/commands/httputil"
	"github.com/ystia/yorc/helper/ziputil"
	"github.com/ystia/yorc/rest"
)

func init() {
	var shouldStreamLogs bool
	var shouldStreamEvents bool
	var deploymentID string
	var deployCmd = &cobra.Command{
		Use:   "deploy <csar_path>",
		Short: "Deploy an application",
		Long: `Deploy a file or directory pointed by <csar_path>
	If <csar_path> point to a valid zip archive it is submitted to Yorc as it.
	If <csar_path> point to a file or directory it is zipped before being submitted to Yorc.
	If <csar_path> point to a single file it should be TOSCA YAML description.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting a path to a file or directory (got %d parameters)", len(args))
			}
			client, err := httputil.GetClient(ClientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}
			absPath, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}
			fileInfo, err := os.Stat(absPath)
			if err != nil {
				return err
			}
			var location = ""
			if !fileInfo.IsDir() {
				file, err := os.Open(absPath)
				if err != nil {
					httputil.ErrExit(err)
				}

				defer file.Close()

				buff, err := ioutil.ReadAll(file)
				if err != nil {
					httputil.ErrExit(err)
				}
				fileType := http.DetectContentType(buff)
				if fileType == "application/zip" {
					location, err = SubmitCSAR(buff, client, deploymentID)
					if err != nil {
						httputil.ErrExit(err)
					}
				}
			}

			if location == "" {
				csarZip, err := ziputil.ZipPath(absPath)
				if err != nil {
					httputil.ErrExit(err)
				}
				location, err = SubmitCSAR(csarZip, client, deploymentID)

				if err != nil {
					httputil.ErrExit(err)
				}
			}
			taskID := path.Base(location)
			if deploymentID == "" {
				deploymentID = path.Base(path.Clean(location + "/../.."))
			}
			fmt.Printf("Deployment submitted. Deployment Id: %s\t(Deployment Task Id: %s)\n", deploymentID, taskID)
			if shouldStreamLogs && !shouldStreamEvents {
				StreamsLogs(client, deploymentID, !NoColor, true, false)
			} else if !shouldStreamLogs && shouldStreamEvents {
				StreamsEvents(client, deploymentID, !NoColor, true, false)
			} else if shouldStreamLogs && shouldStreamEvents {
				return errors.Errorf("You can't provide stream-events and stream-logs flags at same time")
			}
			return nil
		},
	}
	deployCmd.PersistentFlags().BoolVarP(&shouldStreamLogs, "stream-logs", "l", false, "Stream logs after deploying the CSAR. In this mode logs can't be filtered, to use this feature see the \"log\" command.")
	deployCmd.PersistentFlags().BoolVarP(&shouldStreamEvents, "stream-events", "e", false, "Stream events after deploying the CSAR.")
	// Do not impose a max id length as it doesn't have a concrete impact for now
	//deployCmd.PersistentFlags().StringVarP(&deploymentID, "id", "", "", fmt.Sprintf("Specify a id for this deployment. This id should not already exists, should respect the following format: %q and should be less than %d characters long", rest.YorcDeploymentIDPattern, rest.YorcDeploymentIDMaxLength))
	deployCmd.PersistentFlags().StringVarP(&deploymentID, "id", "", "", fmt.Sprintf("Specify a id for this deployment. This id should not already exists, should respect the following format: %q", rest.YorcDeploymentIDPattern))
	DeploymentsCmd.AddCommand(deployCmd)
}

// SubmitCSAR submits the deployment of an archive
func SubmitCSAR(csarZip []byte, client *httputil.YorcClient, deploymentID string) (string, error) {
	var request *http.Request
	var err error
	if deploymentID != "" {
		request, err = client.NewRequest(http.MethodPut, path.Join("/deployments", deploymentID), bytes.NewReader(csarZip))
	} else {
		request, err = client.NewRequest(http.MethodPost, "/deployments", bytes.NewReader(csarZip))
	}
	if err != nil {
		return "", err
	}
	request.Header.Add("Content-Type", "application/zip")
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != 201 {
		// Try to get the reason
		httputil.PrintErrors(response.Body)
		return "", errors.Errorf("POST failed: Expecting HTTP Status code 201 got %d, reason %q", response.StatusCode, response.Status)
	}
	if location := response.Header.Get("Location"); location != "" {
		return location, nil
	}
	return "", errors.New("No \"Location\" header returned in Yorc response")
}
