package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"novaforge.bull.com/starlings-janus/janus/rest"
)

func init() {
	var fromBeginning bool
	var noStream bool
	var filters []string
	var logCmd = &cobra.Command{
		Use:     "logs <DeploymentId>",
		Short:   "Streams logs for a given deployment id",
		Long:    `Streams logs for a given deployment id`,
		Aliases: []string{"log"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("Expecting a deployment id (got %d parameters)", len(args))
			}
			janusAPI := viper.GetString("janus_api")
			colorize := !noColor

			streamsLogs(janusAPI, args[0], colorize, fromBeginning, noStream, filters...)
			return nil
		},
	}
	logCmd.PersistentFlags().BoolVarP(&fromBeginning, "from-beginning", "b", false, "Show logs from the beginning of a deployment")
	logCmd.PersistentFlags().BoolVarP(&noStream, "no-stream", "n", false, "Show logs then exit. Do not stream logs. It implies --from-beginning")
	logCmd.PersistentFlags().StringSliceVarP(&filters, "filter", "f", []string{}, "Allows to filters logs by type. Accepted filters are \"engine\" for Janus logs, \"infrastructure\" for infrastructure provisioning logs or \"software\" for software provisioning. This flag may appear several time and may contain a coma separated list of filters. If not specified logs are not filtered.")
	deploymentsCmd.AddCommand(logCmd)
}

func streamsLogs(janusAPI, deploymentID string, colorize, fromBeginning, stop bool, filters ...string) {
	if colorize {
		defer color.Unset()
	}
	var lastIdx uint64
	if !fromBeginning && !stop {
		// Get last index
		response, err := http.Head("http://" + janusAPI + "/deployments/" + deploymentID + "/logs")
		if err != nil {
			errExit(err)
		}
		if response.StatusCode != 200 {
			// Try to get the reason
			printErrors(response.Body)
			errExit(fmt.Errorf("Expecting HTTP Status code 200 got %d, reason %q", response.StatusCode, response.Status))
		}
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
	if len(filters) > 0 {
		filtersParam = fmt.Sprintf("&filter=%s", strings.Join(filters, ","))
	}
	for {
		request, err := http.NewRequest("GET", fmt.Sprintf("http://%s/deployments/%s/logs?index=%d%s", janusAPI, deploymentID, lastIdx, filtersParam), nil)
		if err != nil {
			errExit(err)
		}
		request.Header.Add("Accept", "application/json")
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			errExit(err)
		}
		if response.StatusCode != 200 {
			// Try to get the reason
			printErrors(response.Body)
			errExit(fmt.Errorf("Expecting HTTP Status code 200 got %d, reason %q", response.StatusCode, response.Status))
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
				fmt.Printf("%s: %s\n", color.CyanString("%s", log.Timestamp), log.Logs)
			} else {
				fmt.Printf("%s: %s\n", log.Timestamp, log.Logs)
			}
		}

		if stop {
			return
		}
	}
}
