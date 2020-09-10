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
	"github.com/spf13/cobra"
	"github.com/ystia/yorc/v4/commands/httputil"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/tabutil"
	"github.com/ystia/yorc/v4/rest"
	"io/ioutil"
	"net/http"
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
		return listLocations(client)
	},
}

func listLocations(client httputil.HTTPClient) error {
	// Get locations definitions
	locsConfig, err := getLocationsConfig(client, true)
	if err != nil {
		httputil.ErrExit(err)
	}
	// Print out locations definitions in a table
	locationsTable := tabutil.NewTable()
	locationsTable.AddHeaders("Name", "Type", "Properties")
	for _, locConfig := range locsConfig.Locations {
		displayLocationInfo(locationsTable, locConfig.Name, locConfig.Type, locConfig.Properties, false, 0)
	}
	fmt.Println("Locations:")
	fmt.Println(locationsTable.Render())
	return nil
}

func displayLocationInfo(table tabutil.Table, locationName, locationType string, properties config.DynamicMap, colorize bool, operation int) {
	propKeys := properties.Keys()
	if len(propKeys) > 0 {
		for i := 0; i < len(propKeys); i++ {
			propValue := properties.Get(propKeys[i])
			displayProperties(table, locationName, locationType, i, propKeys[i], propValue, 0, colorize, operation)
		}
	} else {
		displayRow(table, locationName, locationType, 0, "", "", 0, false, 0)
	}
}

func displayRow(table tabutil.Table, locationName, locationType string, index int, propKey string, propValue interface{}, nbTabs int, colorize bool, operation int) {
	value := fmt.Sprintf("%v", propValue)
	var str string

	// Add tabulations
	for i := 0; i <= nbTabs; i++ {
		str += "  "
	}

	if propKey != "" {
		str += propKey + ": " + value
	} else {
		str += value
	}

	if index == 0 {
		table.AddRow(getColoredText(colorize, locationName, operation), getColoredText(colorize, locationType, operation), getColoredText(colorize, str, operation))
	} else {
		table.AddRow("", "", getColoredText(colorize, str, operation))
	}
}

func displayProperties(table tabutil.Table, locationName, locationType string, index int, propKey string, propValue interface{}, nbTabs int, colorize bool, operation int) {
	switch v := propValue.(type) {
	case []interface{}:
		displayRow(table, locationName, locationType, index, propKey, "", nbTabs, colorize, operation)
		index++
		nbTabs++
		for i := 0; i < len(v); i++ {
			displayProperties(table, locationName, locationType, index, "", v[i], nbTabs, colorize, operation)
		}
	case map[string]interface{}:
		if propKey != "" {
			displayRow(table, locationName, locationType, index, propKey, "", nbTabs, colorize, operation)
			nbTabs++
		}
		index++

		for k, val := range v {
			displayProperties(table, locationName, locationType, index, k, val, nbTabs, colorize, operation)
		}
	default:
		displayRow(table, locationName, locationType, index, propKey, propValue, nbTabs, colorize, operation)
	}
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
		locs.Locations = append(locs.Locations, locConfig)
	}

	return &locs, nil
}
