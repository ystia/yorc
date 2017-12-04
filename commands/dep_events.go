package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"net/http"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/rest"
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
				deploymentID = args[0]
			} else {
				fmt.Println("No deployment id provided, events for all deployments will be returned")
			}

			client, err := getClient()
			if err != nil {
				errExit(err)
			}
			colorize := !noColor

			streamsEvents(client, deploymentID, colorize, fromBeginning, noStream)
			return nil
		},
	}
	eventCmd.PersistentFlags().BoolVarP(&fromBeginning, "from-beginning", "b", false, "Show events from the beginning of a deployment")
	eventCmd.PersistentFlags().BoolVarP(&noStream, "no-stream", "n", false, "Show events then exit. Do not stream events. It implies --from-beginning")
	deploymentsCmd.AddCommand(eventCmd)
}

func streamsEvents(client *janusClient, deploymentID string, colorize, fromBeginning, stop bool) {
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
			handleHTTPStatusCode(response, deploymentID, "deployment", http.StatusOK)
		} else {
			response, err = client.Head("/events")
		}
		if err != nil {
			errExit(err)
		}
		// Get last index
		idxHd := response.Header.Get(rest.JanusIndexHeader)
		if idxHd != "" {
			lastIdx, err = strconv.ParseUint(idxHd, 10, 64)
			if err != nil {
				errExit(err)
			}
			fmt.Println("Streaming new events...")
		} else {
			fmt.Fprint(os.Stderr, "Failed to get latest events index from Janus, events will appear from the beginning.")
		}
	}
	for {
		if deploymentID != "" {
			request, err = client.NewRequest("GET", fmt.Sprintf("/deployments/%s/events?index=%d", deploymentID, lastIdx), nil)
		} else {
			request, err = client.NewRequest("GET", fmt.Sprintf("/events?index=%d", lastIdx), nil)
		}
		if err != nil {
			errExit(err)
		}
		request.Header.Add("Accept", "application/json")
		response, err := client.Do(request)
		if err != nil {
			errExit(err)
		}
		if deploymentID != "" {
			handleHTTPStatusCode(response, deploymentID, "deployment", http.StatusOK)
		}

		var evts rest.EventsCollection
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			errExit(err)
		}
		err = json.Unmarshal(body, &evts)
		if err != nil {
			errExit(err)
		}
		if lastIdx == evts.LastIndex {
			continue
		}
		lastIdx = evts.LastIndex
		for _, event := range evts.Events {
			ts := event.Timestamp
			if colorize {
				ts = color.CyanString("%s", event.Timestamp)
			}
			evType, err := events.StatusUpdateTypeString(event.Type)
			if err != nil {
				if colorize {
					fmt.Printf("%s: ", color.MagentaString("Warning"))
				} else {
					fmt.Print("Warning: ")
				}
				fmt.Printf("Unknown event type: %q\n", event.Type)
			}
			switch evType {
			case events.InstanceStatusChangeType:
				fmt.Printf("%s:\t Deployment: %s\t Node: %s\t Instance: %s\t State: %s\n", ts, event.DeploymentID, event.Node, event.Instance, event.Status)
			case events.DeploymentStatusChangeType:
				fmt.Printf("%s:\t Deployment: %s\t Deployment Status: %s\n", ts, event.DeploymentID, event.Status)
			case events.CustomCommandStatusChangeType:
				fmt.Printf("%s:\t Deployment: %s\t Task %q (custom command)\t Status: %s\n", ts, event.DeploymentID, event.TaskID, event.Status)
			case events.ScalingStatusChangeType:
				fmt.Printf("%s:\t Deployment: %s\t Task %q (scaling)\t Status: %s\n", ts, event.DeploymentID, event.TaskID, event.Status)
			case events.WorkflowStatusChangeType:
				fmt.Printf("%s:\t Deployment: %s\t Task %q (workflow)\t Status: %s\n", ts, event.DeploymentID, event.TaskID, event.Status)
			}

		}

		if stop {
			return
		}
	}
}
