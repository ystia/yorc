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

package hostspool

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ystia/yorc/commands/httputil"
	"github.com/ystia/yorc/helper/tabutil"
	"github.com/ystia/yorc/prov/hostspool"
	"github.com/ystia/yorc/rest"
)

func init() {
	var autoApprove bool
	var applyCmd = &cobra.Command{
		Use:   "apply <path to Hosts Pool configuration>",
		Short: "Apply a Hosts Pool configuration",
		Long: `Apply a Hosts Pool configuration provided in the file passed in argument
	This file should contain a YAML or a JSON description.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			colorize := !noColor
			if len(args) != 1 {
				return errors.Errorf("Expecting a path to a file (got %d parameters)", len(args))
			}
			fileInfo, err := os.Stat(args[0])
			if err != nil {
				return err
			}
			if fileInfo.IsDir() {
				return errors.Errorf("Expecting a path to a file (%s is a directory)", args[0])
			}

			// Read config file, viper will accept indiferrently a yaml or json
			// format
			viper.SetConfigFile(args[0])
			err = viper.ReadInConfig()
			if err != nil {
				log.Panic(err)
			}

			var hostsPoolRequest rest.HostsPoolRequest
			err = viper.Unmarshal(&hostsPoolRequest)
			if err != nil {
				log.Panic(err)
			}

			// Organizing data per host name for easier use below
			newPoolMap := make(map[string]rest.HostConfig)
			for i, host := range hostsPoolRequest.Hosts {
				if host.Name == "" {
					return errors.Errorf("A non-empty Name should be provided for Host number %d, defined with connection %q",
						i+1, host.Connection)
				}

				newPoolMap[host.Name] = host
			}

			// Get current Hosts Pool to output diffs and ask for user
			// confirmation before proceeding to the change
			client, err := httputil.GetClient()
			if err != nil {
				httputil.ErrExit(err)
			}

			request, err := client.NewRequest("GET", "/hosts_pool", nil)
			if err != nil {
				httputil.ErrExit(err)
			}
			request.Header.Add("Accept", "application/json")
			response, err := client.Do(request)
			defer response.Body.Close()
			if err != nil {
				httputil.ErrExit(err)
			}
			httputil.HandleHTTPStatusCode(response, "", "Hosts Pool",
				http.StatusOK, http.StatusNoContent)

			// Unmarshal response content if any
			var hostsColl rest.HostsCollection
			var checkpoint uint64
			if response.StatusCode != http.StatusNoContent {
				body, err := ioutil.ReadAll(response.Body)
				if err != nil {
					httputil.ErrExit(err)
				}
				err = json.Unmarshal(body, &hostsColl)
				if err != nil {
					httputil.ErrExit(err)
				}
				checkpoint = hostsColl.Checkpoint
			}

			// Find which hosts will be deleted, updated, created

			var deletion, update, creation bool
			var hostsImpacted []string
			hostsToDeleteTable := tabutil.NewTable()
			hostsToDeleteTable.AddHeaders(
				"Name", "Connection", "Status", "Message", "Labels")
			hostsToCreateTable := tabutil.NewTable()
			hostsToCreateTable.AddHeaders("Name", "Connection", "Labels")
			hostsToUpdateTable := tabutil.NewTable()
			hostsToUpdateTable.AddHeaders(
				"Version", "Name", "Connection", "Status", "Message", "Labels")

			for _, hostLink := range hostsColl.Hosts {
				if hostLink.Rel == rest.LinkRelHost {
					var host rest.Host

					err = httputil.GetJSONEntityFromAtomGetRequest(
						client, hostLink, &host)
					if err != nil {
						httputil.ErrExit(err)
					}

					if newDef, ok := newPoolMap[host.Name]; ok {
						// This is an update
						//  Check if there is any change before registering the
						// need to update
						if host.Connection != newDef.Connection ||
							!reflect.DeepEqual(host.Labels, newDef.Labels) {
							update = true
							hostsImpacted = append(hostsImpacted, host.Name)
							addUpdateRows(hostsToUpdateTable, colorize,
								host.Name,
								host.Connection,
								host.Status,
								host.Message,
								host.Labels,
								newDef.Connection,
								newDef.Labels)
						}

						// This host is now computed, removing it from the map
						// so that in the end, only hosts to create will be left
						delete(newPoolMap, host.Name)
					} else {
						// host isn't in the new Pool, this is a deletion
						deletion = true
						addRow(hostsToDeleteTable, colorize, hostDeletion,
							host.Name,
							host.Connection,
							&host.Status,
							&host.Message,
							host.Labels)
					}

				}
			}

			// Hosts left in newPoolMap are hosts to create
			for _, host := range newPoolMap {
				creation = true
				hostsImpacted = append(hostsImpacted, host.Name)
				addRow(hostsToCreateTable, colorize, hostCreation,
					host.Name, host.Connection, nil, nil, host.Labels)
			}

			if !deletion && !update && !creation {
				fmt.Println("No change needed, Hosts Pool is up to date.")
				return nil
			}

			fmt.Println("\nThe following changes will be applied.")
			if creation {
				fmt.Println("\n- Hosts to be created :")
				fmt.Println("")
				fmt.Println(hostsToCreateTable.Render())
			}
			if update {
				fmt.Println("\n- Hosts to be updated :")
				fmt.Println("")
				fmt.Println(hostsToUpdateTable.Render())
			}
			if deletion {
				fmt.Println("\n- Hosts to be deleted :")
				fmt.Println("")
				fmt.Println(hostsToDeleteTable.Render())
			}

			if !autoApprove {

				// Ask for confirmation
				badAnswer := true
				var answer string
				for badAnswer {
					fmt.Printf("\nApply these settings [y/N]: ")
					var inputText string
					reader := bufio.NewReader(os.Stdin)
					inputText, err := reader.ReadString('\n')
					badAnswer = err != nil
					if !badAnswer {
						answer = strings.ToLower(strings.TrimSpace(inputText))
						badAnswer = answer != "" &&
							answer != "n" && answer != "no" &&
							answer != "y" && answer != "yes"
					}

					if badAnswer {
						fmt.Println("Unexpected input. Please enter y or n.")
					}
				}

				if answer != "y" && answer != "yes" {
					fmt.Println("Changes not applied.")
					return nil
				}
			}

			// Proceed to the change

			bArray, err := json.Marshal(&hostsPoolRequest)
			request, err = client.NewRequest("POST", "/hosts_pool",
				bytes.NewBuffer(bArray))
			if err != nil {
				httputil.ErrExit(err)
			}
			request.Header.Add("Content-Type", "application/json")

			// Specify the checkpoint that was returned by the 'hosts pool list'
			// request above, to ensure there was no change between the
			// Hosts Pool that was returned by this request
			// and the Hosts Pool to which changes will be applied
			query := request.URL.Query()
			query.Set("checkpoint", strconv.FormatUint(checkpoint, 10))
			request.URL.RawQuery = query.Encode()

			response, err = client.Do(request)
			defer response.Body.Close()
			if err != nil {
				httputil.ErrExit(err)
			}

			// Handle the response
			// This is a generic response management, except from the case where
			// a checkpoint issue was detected.
			// In this case, the REST API error message is overriden by a
			// user-friendly message
			customizedErrorMessages := map[string]string{
				hostspool.CheckpointError: "New Hosts Pool configuration not applied, as a change occured since the above diff. Please re-apply your configuration to see actual changes.",
			}

			httputil.HandleHTTPStatusCodeWithCustomizedErrorMessage(
				response, args[0], "host pool", customizedErrorMessages, http.StatusOK, http.StatusCreated)

			// Verify the status of each updated/new host and log
			// connection failures
			connectionFailure := false
			hostsTable := tabutil.NewTable()
			hostsTable.AddHeaders(
				"Name", "Connection", "Status", "Message")
			for _, name := range hostsImpacted {

				request, err := client.NewRequest("GET", "/hosts_pool/"+name, nil)
				request.Header.Add("Accept", "application/json")
				if err != nil {
					httputil.ErrExit(err)
				}

				response, err := client.Do(request)
				defer response.Body.Close()
				if err != nil {
					httputil.ErrExit(err)
				}

				httputil.HandleHTTPStatusCode(response, name, "host pool", http.StatusOK)
				var host rest.Host
				body, err := ioutil.ReadAll(response.Body)
				if err != nil {
					httputil.ErrExit(err)
				}
				err = json.Unmarshal(body, &host)
				if err != nil {
					httputil.ErrExit(err)
				}

				if host.Status == hostspool.HostStatusError {
					connectionFailure = true
					addRow(hostsTable, colorize, hostError,
						host.Name,
						host.Connection,
						&host.Status,
						&host.Message,
						nil)
				}
			}
			fmt.Println("New hosts pool configuration applied successfully.")
			if connectionFailure {
				fmt.Println("Connection failures occured for the following hosts:")
				fmt.Println("")
				fmt.Println(hostsTable.Render())
			}

			return nil
		},
	}
	applyCmd.PersistentFlags().BoolVarP(&autoApprove, "auto-approve", "", false,
		"Skip interactive approval before applying this new Hosts Pool configuration.")
	hostsPoolCmd.AddCommand(applyCmd)
}

// Returns a printable value of labels
func toPrintableLabels(labels map[string]string) string {
	var labelsList string
	for k, v := range labels {
		if labelsList != "" {
			labelsList += ","
		}
		labelsList += fmt.Sprintf("%s: %s", k, v)
	}

	return labelsList
}

// Returns a printable value of a connection, including empty fields
func toPrintableConnection(connection hostspool.Connection) string {

	return "user: " + connection.User + ",password: " + connection.Password +
		",private key:" + connection.PrivateKey + ",host: " +
		connection.Host + ",port: " + strconv.FormatUint(connection.Port, 10)
}

// Add rows to a table, for both old and new values
// with colored text for changed values
func addUpdateRows(table tabutil.Table, colorize bool,
	name string,
	oldConnection hostspool.Connection,
	status hostspool.HostStatus,
	message string,
	oldLabels map[string]string,
	newConnection hostspool.Connection,
	newLabels map[string]string) {

	// Sorting lables for an easier comparison between old and new labels
	oldLabelsSlice := strings.Split(toPrintableLabels(oldLabels), ",")
	newLabelsSlice := strings.Split(toPrintableLabels(newLabels), ",")
	sort.Strings(oldLabelsSlice)
	sort.Strings(newLabelsSlice)

	// Padding columns in the same row
	oldConnectionSubRows, oldLabelSubRows := padSlices(
		strings.Split(toPrintableConnection(oldConnection), ","),
		oldLabelsSlice)
	newConnectionSubRows, newLabelSubRows := padSlices(
		strings.Split(toPrintableConnection(newConnection), ","),
		newLabelsSlice)

	// Add rows for old values, one row for each sub-column
	colNumber := 6
	oldSubRowsNumber := len(oldLabelSubRows)
	newSubRowsNumber := len(newLabelSubRows)
	version, nameValue, statusValue, messageValue :=
		"old", name, status.String(), message
	for i := 0; i < oldSubRowsNumber; i++ {
		coloredColumns := make([]interface{}, colNumber)
		coloredColumns[0] = getColoredText(colorize, version, hostUpdate)
		coloredColumns[1] = nameValue

		// Connection, a field with no change has no specific color
		operation := hostNoOperation
		if i >= newSubRowsNumber ||
			oldConnectionSubRows[i] != newConnectionSubRows[i] {
			operation = hostUpdate
		}
		coloredColumns[2] = getColoredText(colorize,
			strings.TrimSpace(oldConnectionSubRows[i]), operation)

		coloredColumns[3] = statusValue
		coloredColumns[4] = messageValue

		operation = hostNoOperation
		if oldLabelSubRows[i] != "" {
			key := strings.Split(oldLabelSubRows[i], ":")[0]
			newValue, ok := newLabels[key]
			if !ok || newValue != oldLabels[key] {
				operation = hostUpdate
			}
		}
		coloredColumns[5] = getColoredText(colorize,
			strings.TrimSpace(oldLabelSubRows[i]), operation)

		table.AddRow(coloredColumns...)
		if i == 0 {
			// Don't repeat single column values in sub-columns
			version = ""
			nameValue = ""
			statusValue = ""
			messageValue = ""
		}
	}

	// Add rows for new values, one row for each sub-column
	version, nameValue, statusValue, messageValue =
		"new", name, status.String(), message
	for i := 0; i < newSubRowsNumber; i++ {
		coloredColumns := make([]interface{}, colNumber)
		coloredColumns[0] = getColoredText(colorize, version, hostUpdate)
		coloredColumns[1] = nameValue

		// Connection, a field with no change has no specific color
		operation := hostNoOperation
		if i >= oldSubRowsNumber ||
			oldConnectionSubRows[i] != newConnectionSubRows[i] {
			operation = hostUpdate
		}
		coloredColumns[2] = getColoredText(colorize,
			strings.TrimSpace(newConnectionSubRows[i]), operation)

		coloredColumns[3] = statusValue
		coloredColumns[4] = messageValue

		operation = hostNoOperation
		if newLabelSubRows[i] != "" {
			key := strings.Split(newLabelSubRows[i], ":")[0]
			oldValue, ok := oldLabels[key]
			if !ok || oldValue != newLabels[key] {
				operation = hostUpdate
			}
		}
		coloredColumns[5] = getColoredText(colorize,
			strings.TrimSpace(newLabelSubRows[i]), operation)

		table.AddRow(coloredColumns...)
		if i == 0 {
			// Don't repeat single column values in sub-columns
			version = ""
			nameValue = ""
			statusValue = ""
			messageValue = ""
		}
	}

}
