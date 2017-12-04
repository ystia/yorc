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
	var logCmd = &cobra.Command{
		Use:     "logs [<DeploymentId>]",
		Short:   "Stream logs for a deployment or all deployments",
		Long:    `Stream all the logs, or logs for a given deployment id`,
		Aliases: []string{"log"},
		RunE: func(cmd *cobra.Command, args []string) error {
			var deploymentID string
			if len(args) == 1 {
				deploymentID = args[0]
			} else {
				fmt.Println("No deployment id provided, logs for all deployments will be returned")
			}

			client, err := getClient()
			if err != nil {
				errExit(err)
			}
			colorize := !noColor

			streamsLogs(client, deploymentID, colorize, fromBeginning, noStream)
			return nil
		},
	}
	logCmd.PersistentFlags().BoolVarP(&fromBeginning, "from-beginning", "b", false, "Show logs from the beginning of deployments")
	logCmd.PersistentFlags().BoolVarP(&noStream, "no-stream", "n", false, "Show logs then exit. Do not stream logs. It implies --from-beginning")
	deploymentsCmd.AddCommand(logCmd)
}

func streamsLogs(client *janusClient, deploymentID string, colorize, fromBeginning, stop bool) {
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
			handleHTTPStatusCode(response, deploymentID, "deployment", http.StatusOK)
		} else {
			response, err = client.Head("/logs")
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
			fmt.Println("Streaming new logs...")
		} else {
			fmt.Fprint(os.Stderr, "Failed to get latest log index from Janus, logs will appear from the beginning.")
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

		var logs rest.LogsCollection
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			errExit(err)
		}
		err = json.Unmarshal(body, &logs)
		if err != nil {
			errExit(err)
		}

		lastIdx = logs.LastIndex
		for _, log := range logs.Logs {
			if colorize {
				fmt.Printf("%s\n", color.CyanString("%s", format(log)))
			} else {
				fmt.Printf("%s\n", format(log))
			}
		}
		if stop {
			return
		}
	}
}

func format(log json.RawMessage) string {
	var data map[string]interface{}
	err := json.Unmarshal(log, &data)
	if err != nil {
		errExit(err)
	}
	return events.FormatLog(data)
}
