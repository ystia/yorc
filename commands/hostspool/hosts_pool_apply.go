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
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ystia/yorc/commands/httputil"
	"github.com/ystia/yorc/helper/tabutil"
	"github.com/ystia/yorc/rest"
)

// Internal constants for operations on hosts pool
const (
	hostDeletion = iota
	hostUpdate
	hostCreation
)

func init() {
	var autoApprove bool
	var applyCmd = &cobra.Command{
		Use:   "apply <path to Hosts Pool description>",
		Short: "Apply a Hosts Pool description",
		Long: `Apply a Hosts Pool description provided in the file passed in argument
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
			newPoolMap := make(map[string]rest.HostInPool)
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
			httputil.HandleHTTPStatusCode(response, "", "Hosts Pool", http.StatusOK)
			var hostsColl rest.HostsCollection
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				httputil.ErrExit(err)
			}
			err = json.Unmarshal(body, &hostsColl)
			if err != nil {
				httputil.ErrExit(err)
			}

			// Find which hosts will be deleted, updated, created
			var deletion, update, creation bool
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
							addRow(hostsToUpdateTable, colorize, hostUpdate,
								"old",
								host.Name,
								host.Connection.String(),
								host.Status.String(),
								host.Message,
								toPrintableLabels(host.Labels))

							addRow(hostsToUpdateTable, colorize, hostUpdate,
								"new",
								newDef.Name,
								newDef.Connection.String(),
								host.Status.String(),
								host.Message,
								toPrintableLabels(newDef.Labels))
						}

						// This host is now computed, removing it from the map
						// so that in the end, only hosts to create will be left
						delete(newPoolMap, host.Name)
					} else {
						// host isn't in the new Pool, this is a deletion
						deletion = true
						addRow(hostsToDeleteTable, colorize, hostDeletion,
							host.Name,
							host.Connection.String(),
							host.Status.String(),
							host.Message,
							toPrintableLabels(host.Labels))
					}

				}
			}

			// Hosts left in newPoolMap are hosts to create
			for _, host := range newPoolMap {
				creation = true
				addRow(hostsToCreateTable, colorize, hostCreation,
					host.Name,
					host.Connection.String(),
					toPrintableLabels(host.Labels))
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
			request, err = client.NewRequest("PUT", "/hosts_pool",
				bytes.NewBuffer(bArray))
			if err != nil {
				httputil.ErrExit(err)
			}
			request.Header.Add("Content-Type", "application/json")

			response, err = client.Do(request)
			defer response.Body.Close()
			if err != nil {
				httputil.ErrExit(err)
			}

			httputil.HandleHTTPStatusCode(
				response, args[0], "host pool", http.StatusOK)
			return nil
		},
	}
	applyCmd.PersistentFlags().BoolVarP(&autoApprove, "auto-approve", "", false,
		"Skip interactive approval before applying this new Hosts Pool description.")
	hostsPoolCmd.AddCommand(applyCmd)
}

// Returns a printable value of labels
func toPrintableLabels(labels map[string]string) string {
	var labelsList string
	for k, v := range labels {
		if labelsList != "" {
			labelsList += ", "
		}
		labelsList += fmt.Sprintf("%s:%s", k, v)
	}

	return labelsList
}

// getColoredText returns a text colored accroding to the operation in
// argument :
// - red for a deletion
// - yellow for an update (bold for the new version, regular for the old one)
// - green for a creation
func getColoredText(colorize bool, text string, operation int) string {
	if !colorize {
		return text
	}
	switch operation {
	case hostCreation:
		return color.New(color.FgHiGreen, color.Bold).SprintFunc()(text)
	case hostUpdate:
		return color.New(color.FgHiYellow, color.Bold).SprintFunc()(text)
	case hostDeletion:
		return color.New(color.FgHiRed, color.Bold).SprintFunc()(text)
	default:
		return text
	}
}

// Add a row to a table, with text colored according to the operation
func addRow(table tabutil.Table, colorize bool, operation int, columns ...string) {

	coloredColumns := make([]interface{}, len(columns))
	for i, text := range columns {
		coloredColumns[i] = getColoredText(colorize, text, operation)
	}

	table.AddRow(coloredColumns...)
}
