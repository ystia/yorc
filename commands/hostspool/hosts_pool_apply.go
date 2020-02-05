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
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ystia/yorc/v4/commands/httputil"
	"github.com/ystia/yorc/v4/helper/sliceutil"
	"github.com/ystia/yorc/v4/helper/tabutil"
	"github.com/ystia/yorc/v4/prov/hostspool"
	"github.com/ystia/yorc/v4/rest"
)

func init() {
	var location string
	var autoApprove bool
	var applyCmd = &cobra.Command{
		Use:   "apply -l <locationName> <path to Hosts Pool configuration>",
		Short: "Apply a Hosts Pool configuration for a specified location",
		Long: `Apply a Hosts Pool configuration provided in the file passed in argument for a specified location
	This file should contain a YAML or a JSON description.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := httputil.GetClient(clientConfig)
			if err != nil {
				return err
			}
			return applyHostsPoolConfig(client, args, location, autoApprove)
		},
	}
	applyCmd.Flags().StringVarP(&location, "location", "l", "", "Need to provide the specified hosts pool location name")
	applyCmd.PersistentFlags().BoolVarP(&autoApprove, "auto-approve", "", false,
		"Skip interactive approval before applying this new Hosts Pool configuration.")
	hostsPoolCmd.AddCommand(applyCmd)
}

func applyHostsPoolConfig(client httputil.HTTPClient, args []string, location string, autoApprove bool) error {
	if location == "" {
		return errors.Errorf("Expecting a hosts pool location name")
	}
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

	// Read config file, viper will accept indifferently a yaml or json
	// format
	v := viper.New()
	v.SetConfigFile(args[0])
	err = v.ReadInConfig()
	if err != nil {
		return err
	}

	var hostsPoolRequest rest.HostsPoolRequest
	err = v.Unmarshal(&hostsPoolRequest)
	if err != nil {
		return err
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
	request, err := client.NewRequest("GET", "/hosts_pool/"+location, nil)
	if err != nil {
		return err
	}
	request.Header.Add("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	httputil.HandleHTTPStatusCode(response, "", "Hosts Pool",
		http.StatusOK, http.StatusNoContent)

	// Unmarshal response content if any
	var hostsColl rest.HostsCollection
	var checkpoint uint64
	if response.StatusCode != http.StatusNoContent {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}
		err = json.Unmarshal(body, &hostsColl)
		if err != nil {
			return err
		}
		checkpoint = hostsColl.Checkpoint
	}

	// Find which hosts will be deleted, updated, created

	var deletion, update, creation bool
	var hostsImpacted []string
	hostsToDeleteTable := tabutil.NewTable()
	hostsToDeleteTable.AddHeaders(
		"Name", "Connection", "Status", "Allocations", "Message", "Labels")
	hostsToCreateTable := tabutil.NewTable()
	hostsToCreateTable.AddHeaders("Name", "Connection", "Labels")
	hostsToUpdateTable := tabutil.NewTable()
	hostsToUpdateTable.AddHeaders(
		"Version", "Name", "Connection", "Status", "Allocations", "Message", "Labels")

	for _, hostLink := range hostsColl.Hosts {
		if hostLink.Rel == rest.LinkRelHost {
			var host rest.Host

			err = httputil.GetJSONEntityFromAtomGetRequest(
				client, hostLink, &host)
			if err != nil {
				return err
			}

			if newDef, ok := newPoolMap[host.Name]; ok {
				// This is an update
				//  Check if there is any change before registering the
				// need to update
				if host.Connection != newDef.Connection ||
					!reflect.DeepEqual(host.Labels, newDef.Labels) {
					update = true
					hostsImpacted = append(hostsImpacted, host.Name)
					addUpdateRows(hostsToUpdateTable, colorize, &host, &rest.Host{Host: hostspool.Host{Connection: newDef.Connection, Labels: newDef.Labels}})
				}

				// This host is now computed, removing it from the map
				// so that in the end, only hosts to create will be left
				delete(newPoolMap, host.Name)
			} else {
				// host isn't in the new Pool, this is a deletion
				deletion = true
				addRow(hostsToDeleteTable, colorize, hostDeletion, &host, true)
			}

		}
	}

	// Hosts left in newPoolMap are hosts to create
	for _, host := range newPoolMap {
		creation = true
		hostsImpacted = append(hostsImpacted, host.Name)
		addRow(hostsToCreateTable, colorize, hostCreation, &rest.Host{Host: hostspool.Host{Name: host.Name, Connection: host.Connection, Labels: host.Labels}}, false)
	}

	if !deletion && !update && !creation {
		fmt.Println("No change needed, Hosts Pool is up to date.")
		return nil
	}

	fmt.Printf("\nThe following changes will be applied on location %q.\n", location)
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
			fmt.Printf("\nApply these settings (Warning: host resources labels can be recalculated depending of allocations) [y/N]: ")
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
	request, err = client.NewRequest("POST", "/hosts_pool/"+location,
		bytes.NewBuffer(bArray))
	if err != nil {
		return err
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
	if err != nil {
		return err
	}
	defer response.Body.Close()

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

		request, err := client.NewRequest("GET", "/hosts_pool/"+location+"/"+name, nil)
		if err != nil {
			return err
		}
		request.Header.Add("Accept", "application/json")

		response, err := client.Do(request)
		if err != nil {
			return err
		}
		defer response.Body.Close()

		httputil.HandleHTTPStatusCode(response, name, "host pool", http.StatusOK)
		var host rest.Host
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}
		err = json.Unmarshal(body, &host)
		if err != nil {
			return err
		}

		if host.Status == hostspool.HostStatusError {
			connectionFailure = true
			addHostInErrorRow(hostsTable, colorize, hostError, &host)
		}
	}
	fmt.Println("New hosts pool configuration applied successfully.")
	if connectionFailure {
		fmt.Println("Connection failures occured for the following hosts:")
		fmt.Println("")
		fmt.Println(hostsTable.Render())
	}

	return nil
}

// Returns a printable value of labels
func toPrintableLabels(labels map[string]string) string {
	var labelsList string
	for k, v := range labels {
		if labelsList != "" {
			labelsList += "|"
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
func addUpdateRows(table tabutil.Table, colorize bool, oldHost *rest.Host, newHost *rest.Host) {
	name := oldHost.Name
	message := oldHost.Message
	status := oldHost.Status

	oldConnection := oldHost.Connection
	newConnection := newHost.Connection
	oldLabels := oldHost.Labels
	newLabels := newHost.Labels

	// Padding columns in the same row
	allocationsSubRows := make([]string, 0)
	for _, alloc := range oldHost.Allocations {
		allocationsSubRows = append(allocationsSubRows, strings.Split(alloc.String(), ",")...)
	}
	oldConnectionSubRows := strings.Split(toPrintableConnection(oldConnection), ",")
	oldLabelSubRows := strings.Split(toPrintableLabels(oldLabels), ",")
	// Sorting labels for an easier comparison between old and new labels
	sort.Strings(oldLabelSubRows)
	sliceutil.PadSlices("", &allocationsSubRows, &oldConnectionSubRows, &oldLabelSubRows)

	newConnectionSubRows := strings.Split(toPrintableConnection(newConnection), ",")
	newLabelSubRows := strings.Split(toPrintableLabels(newLabels), ",")
	// Sorting labels for an easier comparison between old and new labels
	sort.Strings(newLabelSubRows)
	sliceutil.PadSlices("", &allocationsSubRows, &newConnectionSubRows, &newLabelSubRows)

	// Add rows for old values, one row for each sub-column
	colNumber := 7
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

		coloredColumns[4] = getColoredText(colorize,
			strings.TrimSpace(allocationsSubRows[i]), operation)

		coloredColumns[5] = messageValue

		operation = hostNoOperation
		if oldLabelSubRows[i] != "" {
			key := strings.Split(oldLabelSubRows[i], ":")[0]
			newValue, ok := newLabels[key]
			if !ok || newValue != oldLabels[key] {
				operation = hostUpdate
			}
		}
		coloredColumns[6] = getColoredText(colorize,
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
		coloredColumns[4] = getColoredText(colorize,
			strings.TrimSpace(allocationsSubRows[i]), operation)

		coloredColumns[5] = messageValue

		operation = hostNoOperation
		if newLabelSubRows[i] != "" {
			key := strings.Split(newLabelSubRows[i], ":")[0]
			oldValue, ok := oldLabels[key]
			if !ok || oldValue != newLabels[key] {
				operation = hostUpdate
			}
		}
		coloredColumns[6] = getColoredText(colorize,
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
