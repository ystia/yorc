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
	locsConfig, err := getLocationsConfig(client, true)
	if err != nil {
		httputil.ErrExit(err)
	}
	// Print out locations definitions in a table
	locsTable := tabutil.NewTable()
	locsTable.AddHeaders("Name", "Type", "Properties")
	for _, locConfig := range locsConfig.Locations {
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
	}
	fmt.Println("Locations:")
	fmt.Println(locsTable.Render())
	return nil
}

// getLocationsConfig uses the HTTP client to make a request to locations API.
// It returns a collection of existent location definitions. If no location definition exist,
// the treatement depends on retOnError value :
// - true : the command returns and status code displayed
// - false: the function returns a nil value to the caller
func getLocationsConfig(client httputil.HTTPClient, retOnError bool) (*rest.LocationsCollection, error) {
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
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if retOnError {
		httputil.HandleHTTPStatusCode(response, "", "locations", http.StatusOK)
	} else {
		// check for body content and renturn nil result if no values found
		if len(body) == 0 {
			return nil, nil
		}
	}

	var locRefs rest.LocationCollection
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
		if locConfig.Type != adapter.AdaptedLocationType {
			locs.Locations = append(locs.Locations, locConfig)
		}
	}

	return &locs, nil
}
