package commands

import (
	"fmt"
	"net/http"

	"path"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	var shouldStreamLogs bool
	var shouldStreamEvents bool
	var workflowName string
	var wfExecCmd = &cobra.Command{
		Use:     "execute <id>",
		Short:   "Trigger a custom workflow on deployment <id>",
		Aliases: []string{"exec"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting an id (got %d parameters)", len(args))
			}
			client, err := getClient()
			if err != nil {
				errExit(err)
			}
			if workflowName == "" {
				return errors.New("Missing mandatory \"workflow-name\" parameter")
			}

			request, err := client.NewRequest("POST", fmt.Sprintf("/deployments/%s/workflows/%s", args[0], workflowName), nil)
			if err != nil {
				errExit(err)
			}
			request.Header.Add("Content-Type", "application/json")
			response, err := client.Do(request)
			if err != nil {
				errExit(err)
			}
			if response.StatusCode != http.StatusAccepted && response.StatusCode != http.StatusCreated {
				// Try to get the reason
				printErrors(response.Body)
				errExit(fmt.Errorf("Expecting HTTP Status code 201 or 202 got %d, reason %q", response.StatusCode, response.Status))
			}

			fmt.Println("New task ", path.Base(response.Header.Get("Location")), " created to execute ", workflowName)
			if shouldStreamLogs && !shouldStreamEvents {
				streamsLogs(client, args[0], !noColor, false, false)
			} else if !shouldStreamLogs && shouldStreamEvents {
				streamsEvents(client, args[0], !noColor, false, false)
			} else if shouldStreamLogs && shouldStreamEvents {
				return errors.Errorf("You can't provide stream-events and stream-logs flags at same time")
			}
			return nil
		},
	}
	wfExecCmd.PersistentFlags().StringVarP(&workflowName, "workflow-name", "w", "", "The workflows name")
	wfExecCmd.PersistentFlags().BoolVarP(&shouldStreamLogs, "stream-logs", "l", false, "Stream logs after triggering a workflow. In this mode logs can't be filtered, to use this feature see the \"log\" command.")
	wfExecCmd.PersistentFlags().BoolVarP(&shouldStreamEvents, "stream-events", "e", false, "Stream events after riggering a workflow.")
	workflowsCmd.AddCommand(wfExecCmd)
}
