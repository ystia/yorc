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
		//colorize := !NoColor
		client, err := httputil.GetClient(ClientConfig)
		if err != nil {
			httputil.ErrExit(err)
		}
		request, err := client.NewRequest("GET", "/locations", nil)
		if err != nil {
			httputil.ErrExit(err)
		}
		request.Header.Add("Content-Type", "application/json")
		response, err := client.Do(request)
		if err != nil {
			httputil.ErrExit(err)
		}
		defer response.Body.Close()
		httputil.HandleHTTPStatusCode(response, "", "locations", http.StatusOK)
		var locs rest.LocationsCollection
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			httputil.ErrExit(err)
		}
		err = json.Unmarshal(body, &locs)
		if err != nil {
			httputil.ErrExit(err)
		}

		locsTable := tabutil.NewTable()
		locsTable.AddHeaders("Name", "Type", "Properties")
		for _, loc := range locs.Locations {
			if loc.Type != adapter.AdaptedLocationType {
				locProps := loc.Properties
				propKeys := locProps.Keys()
				for i := 0; i < len(propKeys); i++ {
					propValue := locProps.Get(propKeys[i])
					value := fmt.Sprintf("%v", propValue)
					prop := propKeys[i] + ": " + value
					if i == 0 {
						locsTable.AddRow(loc.Name, loc.Type, prop)
					} else {
						locsTable.AddRow("", "", prop)
					}
				}
			} else {
				locsTable.AddRow(loc.Name, loc.Type, "")
			}
		}
		fmt.Println("Locations:")
		fmt.Println(locsTable.Render())
		return nil
	},
}
