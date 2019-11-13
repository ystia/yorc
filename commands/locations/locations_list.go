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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/ystia/yorc/v4/commands/httputil"
	"github.com/ystia/yorc/v4/helper/tabutil"
	"github.com/ystia/yorc/v4/locations/adapter"
	"github.com/ystia/yorc/v4/rest"
)

func init() {
	LocationsCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List locations",
	Long:  `List locations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := httputil.GetClient(ClientConfig)
		if err != nil {
			httputil.ErrExit(err)
		}
		return listLocations(client, args)
	},
}

func listLocations(client httputil.HTTPClient, args []string) error {
	// Get locations definitions
	locsConfig, err := getLocationsConfig(client)
	if err != nil {
		httputil.ErrExit(err)
	}
	// Print out locations definitions in a table
	locsTable := tabutil.NewTable()
	locsTable.AddHeaders("Name", "Type", "Properties")
	for _, locConfig := range locsConfig.Locations {
		if locConfig.Type != adapter.AdaptedLocationType {
			locProps := locConfig.Properties
			propKeys := locProps.Keys()
			for i := 0; i < len(propKeys); i++ {
				propValue := locProps.Get(propKeys[i])
				value := fmt.Sprintf("%v", propValue)
				prop := propKeys[i] + ": " + value
				if i == 0 {
					locsTable.AddRow(locConfig.Name, locConfig.Type, prop)
				} else {
					locsTable.AddRow("", "", prop)
				}
			}
		} else {
			locsTable.AddRow(locConfig.Name, locConfig.Type, "")
		}
	}
	fmt.Println("Locations:")
	fmt.Println(locsTable.Render())
	return nil
}

// getLocationsConfig makes a GET request to locations API and returns the existent location definitions
func getLocationsConfig(client httputil.HTTPClient) (*rest.LocationsCollection, error) {
	request, err := client.NewRequest("GET", "/locations", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	httputil.HandleHTTPStatusCode(response, "", "locations", http.StatusOK)
	var locRefs rest.LocationCollection

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &locRefs)
	if err != nil {
		return nil, err
	}

	var locs rest.LocationsCollection

	for _, locLink := range locRefs.Locations {
		var locConfig rest.LocationConfiguration
		err = httputil.GetJSONEntityFromAtomGetRequest(client, locLink, &locConfig)
		if err != nil {
			return nil, err
		}
		locs.Locations = append(locs.Locations, locConfig)
	}

	return &locs, nil
}
