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

package locations

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ystia/yorc/v4/commands/httputil"
	"github.com/ystia/yorc/v4/helper/tabutil"
	"github.com/ystia/yorc/v4/rest"
)

func init() {
	var autoApprove bool
	var applyCmd = &cobra.Command{
		Use:   "apply <path to locations configuration file>",
		Short: "Apply a locations configuration file",
		Long: `Apply a locations configuration provided in the file passed in argument. 
		This file should contain a YAML or a JSON description.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := httputil.GetClient(ClientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}
			return applyLocationsConfig(client, args, autoApprove)
		},
	}
	applyCmd.PersistentFlags().BoolVarP(&autoApprove, "auto-approve", "", false,
		"Skip interactive approval before applying this new locations configuration.")
	LocationsCmd.AddCommand(applyCmd)
}

// readConfigFile reads the configuration file and returns the provided locations configurations
func readConfigFile(client httputil.HTTPClient, path string) (*rest.LocationsCollection, error) {
	var locationsApplied *rest.LocationsCollection

	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fileInfo.IsDir() {
		return nil, errors.Errorf("Expecting a path to a file (%s is a directory)", path)
	}

	// Read config file, viper will accept indifferently a yaml or json
	// format
	v := viper.New()
	v.SetConfigFile(path)
	err = v.ReadInConfig()
	if err != nil {
		return nil, err
	}
	err = v.Unmarshal(&locationsApplied)
	if err != nil {
		return nil, err
	}
	return locationsApplied, nil
}

func applyLocationsConfig(client httputil.HTTPClient, args []string, autoApprove bool) error {
	colorize := !noColor
	if len(args) != 1 {
		return errors.Errorf("Expecting a path to a file (got %d parameters)", len(args))
	}

	// Get locations definitions proposed by the client
	locationsApplied, err := readConfigFile(client, args[0])
	if err != nil {
		return err
	}

	// Put all these desfinitions for the momemnt in a map of locations to create
	newLocationsMap := make(map[string]rest.LocationConfiguration)
	for _, newLocation := range locationsApplied.Locations {
		newLocationsMap[newLocation.Name] = newLocation
	}
	// Prepare a map for locations to update
	updateLocationsMap := make(map[string]rest.LocationConfiguration)

	// Prepare a map for locations to delete
	deleteLocationsMap := make(map[string]rest.LocationConfiguration)

	// Get existent locations configuration
	locsConfig, err := getLocationsConfig(client, false)
	if err != nil {
		return err
	}
	// Use an array for existent locations configuration
	// to avoid nesting a for that contains if/the/else, in an if
	var existentLocsConfig []rest.LocationConfiguration
	if locsConfig != nil {
		existentLocsConfig = locsConfig.Locations
	} else {
		existentLocsConfig = make([]rest.LocationConfiguration, 0)
	}
	// update newLocationsMap, updateLocationsMap and deleteLocationsMap
	// based on already existent locations configurations
	for _, locConfig := range existentLocsConfig {
		newLocConfig, ok := newLocationsMap[locConfig.Name]
		checkUpdate := false
		if ok {
			// newLocConfig corresponds to an already defined location
			// Delete newLocConfig from the map of locations to create
			delete(newLocationsMap, locConfig.Name)
			checkUpdate = true
		} else {
			// locConfig is not in the new locations specifications, delete it from consul
			deleteLocationsMap[locConfig.Name] = locConfig
		}
		// Check if there is any change before registering the need to update
		if checkUpdate && (locConfig.Type != newLocConfig.Type || !reflect.DeepEqual(locConfig.Properties, newLocConfig.Properties)) {
			// Add newLocConfig to the map for locations to update
			updateLocationsMap[locConfig.Name] = newLocConfig
		}
	}

	// Present locations to be created
	presentLocationsMap(&newLocationsMap, colorize, locationCreation)

	// Present locations to be updated
	presentLocationsMap(&updateLocationsMap, colorize, locationUpdate)

	// Present locations to be deleted
	presentLocationsMap(&deleteLocationsMap, colorize, locationDeletion)

	if (len(newLocationsMap) + len(updateLocationsMap) + len(deleteLocationsMap)) > 0 {
		// Changes to do. Let's see what user decides
		if !approveToApply(autoApprove) {
			return nil
		}
	}

	// Proceed to apply changes
	err = doApply(client, &newLocationsMap, &updateLocationsMap, &deleteLocationsMap)
	if err != nil {
		return err
	}

	return nil
}

func doApply(client httputil.HTTPClient, createMap, updateMap, deleteMap *map[string]rest.LocationConfiguration) error {
	// Proceed to the create
	for _, newLocation := range *createMap {
		locConfig := rest.LocationConfiguration{
			Name:       newLocation.Name,
			Type:       newLocation.Type,
			Properties: newLocation.Properties,
		}

		locName, err := putLocationConfig(client, locConfig)
		if err != nil {
			return err
		}

		fmt.Printf("Location %s created", locName)
		fmt.Println("")
	}

	// Proceed to update
	for _, updateLocation := range *updateMap {
		locConfig := rest.LocationConfiguration{
			Name:       updateLocation.Name,
			Type:       updateLocation.Type,
			Properties: updateLocation.Properties,
		}

		locName, err := patchLocationConfig(client, locConfig)
		if err != nil {
			return err
		}

		fmt.Printf("Location %s updated", locName)
		fmt.Println("")
	}

	// Proceed to delete
	for locNameToDelete := range *deleteMap {
		err := deleteLocationConfig(client, locNameToDelete)
		if err != nil {
			return err
		}

		fmt.Printf("Location %s deleted", locNameToDelete)
		fmt.Println("")
	}

	return nil
}

func presentLocationsMap(locationsMap *map[string]rest.LocationConfiguration, colorize bool, op int) {
	if len(*locationsMap) > 0 {
		locationsTable := tabutil.NewTable()
		locationsTable.AddHeaders("Name", "Type", "Properties")
		for _, locConfig := range *locationsMap {
			addRow(locationsTable, colorize, op, locConfig)
		}
		fmt.Printf("\n- Locations to %s:", opNames[op])
		fmt.Println("")
		fmt.Println(locationsTable.Render())
		fmt.Printf("Number of locations to %s : %v ", opNames[op], len(*locationsMap))
		fmt.Println("")
	}
}

func approveToApply(autoApprove bool) bool {
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
		// Check answer
		if answer != "y" && answer != "yes" {
			fmt.Println("Changes not applied.")
			return false
		}
	}
	return true
}
