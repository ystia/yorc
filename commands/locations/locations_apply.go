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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ystia/yorc/v4/commands/httputil"
	"github.com/ystia/yorc/v4/locations/adapter"
	"github.com/ystia/yorc/v4/rest"
)

func init() {
	//var autoApprove bool
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
			return applyLocationsConfig(client, args /*, autoApprove*/)
		},
	}
	// TODO implement approval
	//applyCmd.PersistentFlags().BoolVarP(&autoApprove, "auto-approve", "", false,
	//	"Skip interactive approval before applying this new locations configuration.")
	LocationsCmd.AddCommand(applyCmd)
}

func applyLocationsConfig(client httputil.HTTPClient, args []string /*, autoApprove bool*/) error {
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

	// Get locations definitions proposed by the client
	var locationsApplied rest.LocationsCollection
	err = v.Unmarshal(&locationsApplied)
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

	// Prepare a slice for locations to delete
	var locationsToDelete []string

	// Get existent locations configuration
	locsConfig, err := getLocationsConfig(client)
	for _, locConfig := range locsConfig.Locations {

		if locConfig.Type != adapter.AdaptedLocationType {
			if newLocConfig, ok := newLocationsMap[locConfig.Name]; ok {
				// newLocConfig corresponds to an already defined location
				// Check if there is any change before registering the need to update
				if locConfig.Type != newLocConfig.Type ||
					!reflect.DeepEqual(locConfig.Properties, newLocConfig.Properties) {
					// Add newLocConfig to the map for locations to update
					updateLocationsMap[locConfig.Name] = newLocConfig
				}
				// Delete newLocConfig from the map of locations to create
				delete(newLocationsMap, locConfig.Name)
			} else {
				// locConfig is not in the new locations specifications, delete it from consul
				fmt.Println("Location " + locConfig.Name + " is not defined in the new file to apply. It will be deleted")
				locationsToDelete = append(locationsToDelete, locConfig.Name)
			}
		}
	}

	fmt.Printf("New Locations to create : %v ", len(newLocationsMap))
	fmt.Printf("Total number of locations to update : %v", len(updateLocationsMap))
	fmt.Printf("Total number of locations to delete : %v", len(locationsToDelete))
	fmt.Println("")

	// Proceed to the create

	for _, newLocation := range newLocationsMap {
		locationName := newLocation.Name
		locationRequest := rest.LocationRequest{
			Type:       newLocation.Type,
			Properties: newLocation.Properties,
		}
		bArray, err := json.Marshal(locationRequest)
		request, err := client.NewRequest("PUT", "/locations/"+locationName, bytes.NewBuffer(bArray))
		if err != nil {
			return err
		}
		request.Header.Add("Content-Type", "application/json")
		_, err = client.Do(request)
		if err != nil {
			return err
		}

		fmt.Printf("New Locations %s created !!!", newLocation.Name)
		fmt.Println("")

	}

	// Proceed to update
	for _, updateLocation := range updateLocationsMap {
		locationName := updateLocation.Name
		locationRequest := rest.LocationRequest{
			Type:       updateLocation.Type,
			Properties: updateLocation.Properties,
		}
		bArray, err := json.Marshal(locationRequest)
		request, err := client.NewRequest("PATCH", "/locations/"+locationName, bytes.NewBuffer(bArray))
		if err != nil {
			return err
		}
		request.Header.Add("Content-Type", "application/json")

		_, err = client.Do(request)
		if err != nil {
			return err
		}

		fmt.Printf("Locations %s updated !!!", updateLocation.Name)
		fmt.Println("")

	}

	// Proceed to delete

	for _, locNameToDelete := range locationsToDelete {
		request, err := client.NewRequest("DELETE", "/locations/"+locNameToDelete, nil)
		if err != nil {
			return err
		}
		_, err = client.Do(request)
		if err != nil {
			return err
		}
		fmt.Printf("Locations %s deleted !!!", locNameToDelete)
		fmt.Println("")
	}

	return nil
}
