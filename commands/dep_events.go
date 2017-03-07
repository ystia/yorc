package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"novaforge.bull.com/starlings-janus/janus/rest"
)

func init() {
	var fromBeginning bool
	var noStream bool
	var eventCmd = &cobra.Command{
		Use:     "events <DeploymentId>",
		Short:   "Streams events for a given deployment id",
		Long:    `Streams events for a given deployment id`,
		Aliases: []string{"events"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("Expecting a deployment id (got %d parameters)", len(args))
			}
			janusAPI := viper.GetString("janus_api")
			colorize := !noColor

			streamsEvents(janusAPI, args[0], colorize, fromBeginning, noStream)
			return nil
		},
	}
	eventCmd.PersistentFlags().BoolVarP(&fromBeginning, "from-beginning", "b", false, "Show events from the beginning of a deployment")
	eventCmd.PersistentFlags().BoolVarP(&noStream, "no-stream", "n", false, "Show events then exit. Do not stream events. It implies --from-beginning")
	deploymentsCmd.AddCommand(eventCmd)
}

func streamsEvents(janusAPI, deploymentID string, colorize, fromBeginning, stop bool) {
	if colorize {
		defer color.Unset()
	}
	var lastIdx uint64
	if !fromBeginning && !stop {
		// Get last index
		response, err := http.Head("http://" + janusAPI + "/deployments/" + deploymentID + "/events")
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
			fmt.Println("Streaming new events...")
		} else {
			fmt.Fprint(os.Stderr, "Failed to get latest events index from Janus, events will appear from the beginning.")
		}
	}
	for {

		request, err := http.NewRequest("GET", fmt.Sprintf("http://%s/deployments/%s/events?index=%d", janusAPI, deploymentID, lastIdx), nil)
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
		var events rest.EventsCollection
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			errExit(err)
		}
		err = json.Unmarshal(body, &events)
		if err != nil {
			errExit(err)
		}
		if lastIdx == events.LastIndex {
			continue
		}
		lastIdx = events.LastIndex
		for _, event := range events.Events {
			if colorize {
				fmt.Printf("%s:\t Node: %s\t Instance: %s\t State: %s\n", color.CyanString("%s", event.Timestamp), event.Node, event.Instance, event.Status)
			} else {
				fmt.Printf("%s:\t Node: %s\t Instance: %s\t State: %s\n", event.Timestamp, event.Node, event.Instance, event.Status)
			}
		}

		if stop {
			return
		}
	}
}
