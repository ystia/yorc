package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/rest"
)

func init() {
	var fromBeginning bool
	var noStream bool
	var logCmd = &cobra.Command{
		Use:     "logs <DeploymentId>",
		Short:   "Stream logs for a deployment",
		Long:    `Stream logs for a given deployment id`,
		Aliases: []string{"log"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting a deployment id (got %d parameters)", len(args))
			}
			client, err := getClient()
			if err != nil {
				errExit(err)
			}
			colorize := !noColor

			streamsLogs(client, args[0], colorize, fromBeginning, noStream)
			return nil
		},
	}
	logCmd.PersistentFlags().BoolVarP(&fromBeginning, "from-beginning", "b", false, "Show logs from the beginning of a deployment")
	logCmd.PersistentFlags().BoolVarP(&noStream, "no-stream", "n", false, "Show logs then exit. Do not stream logs. It implies --from-beginning")
	deploymentsCmd.AddCommand(logCmd)
}

func streamsLogs(client *janusClient, deploymentID string, colorize, fromBeginning, stop bool) {
	if colorize {
		defer color.Unset()
	}
	var lastIdx uint64
	if !fromBeginning && !stop {
		// Get last index
		response, err := client.Head("/deployments/" + deploymentID + "/logs")
		if err != nil {
			errExit(err)
		}
		handleHTTPStatusCode(response, deploymentID, "deployment", http.StatusOK)
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
		request, err := client.NewRequest("GET", fmt.Sprintf("/deployments/%s/logs?index=%d%s", deploymentID, lastIdx, filtersParam), nil)
		if err != nil {
			errExit(err)
		}
		request.Header.Add("Accept", "application/json")
		response, err := client.Do(request)
		if err != nil {
			errExit(err)
		}

		handleHTTPStatusCode(response, deploymentID, "deployment", http.StatusOK)
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
