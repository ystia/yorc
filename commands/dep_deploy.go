package commands

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
	"novaforge.bull.com/starlings-janus/janus/helper/ziputil"
	"novaforge.bull.com/starlings-janus/janus/rest"
)

func init() {
	var shouldStreamLogs bool
	var shouldStreamEvents bool
	var deploymentID string
	var deployCmd = &cobra.Command{
		Use:   "deploy <csar_path>",
		Short: "Deploy a CSAR",
		Long: `Deploy a file or directory pointed by <csar_path>
	If <csar_path> point to a valid zip archive it is submitted to Janus as it.
	If <csar_path> point to a file or directory it is zipped before beeing submitted to Janus.
	If <csar_path> point to a single file it should be TOSCA YAML description.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("Expecting a path to a file or directory (got %d parameters)", len(args))
			}
			client, err := getClient()
			if err != nil {
				errExit(err)
			}
			absPath, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}
			fileInfo, err := os.Stat(absPath)
			if err != nil {
				return err
			}
			var location string = ""
			if !fileInfo.IsDir() {
				file, err := os.Open(absPath)
				if err != nil {
					errExit(err)
				}

				defer file.Close()

				buff, err := ioutil.ReadAll(file)
				if err != nil {
					errExit(err)
				}
				fileType := http.DetectContentType(buff)
				if fileType == "application/zip" {
					location, err = submitCSAR(buff, client, deploymentID)
					if err != nil {
						errExit(err)
					}
				}
			}

			if location == "" {
				csarZip, err := ziputil.ZipPath(absPath)
				if err != nil {
					errExit(err)
				}
				location, err = submitCSAR(csarZip, client, deploymentID)

				if err != nil {
					errExit(err)
				}
			}

			fmt.Println("Deployment submitted. Deployment Id:", path.Base(location))
			if shouldStreamLogs && !shouldStreamEvents {
				streamsLogs(client, path.Base(location), !noColor, true, false)
			} else if !shouldStreamLogs && shouldStreamEvents {
				streamsEvents(client, path.Base(location), !noColor, true, false)
			} else if shouldStreamLogs && shouldStreamEvents {
				return errors.Errorf("You can't provide stream-events and stream-logs flags at same time")
			}
			return nil
		},
	}
	deployCmd.PersistentFlags().BoolVarP(&shouldStreamLogs, "stream-logs", "l", false, "Stream logs after deploying the CSAR. In this mode logs can't be filtered, to use this feature see the \"log\" command.")
	deployCmd.PersistentFlags().BoolVarP(&shouldStreamEvents, "stream-events", "e", false, "Stream events after deploying the CSAR.")
	deployCmd.PersistentFlags().StringVarP(&deploymentID, "id", "", "", fmt.Sprintf("Specify a id for this deployment. This id should not already exists, should respect the following format: %q and should be less than %d characters long", rest.JanusDeploymentIDPattern, rest.JanusDeploymentIDMaxLength))
	deploymentsCmd.AddCommand(deployCmd)
}

func submitCSAR(csarZip []byte, client *janusClient, deploymentID string) (string, error) {
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
	r, err := client.Do(request)
	if err != nil {
		return "", err
	}
	if r.StatusCode != 201 {
		// Try to get the reason
		printErrors(r.Body)
		return "", fmt.Errorf("POST failed: Expecting HTTP Status code 201 got %d, reason %q", r.StatusCode, r.Status)
	}
	if location := r.Header.Get("Location"); location != "" {
		return location, nil
	}
	return "", errors.New("No \"Location\" header returned in Janus response")
}
