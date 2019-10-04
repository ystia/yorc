// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/ystia/yorc/v4/commands/httputil"
	"github.com/ystia/yorc/v4/helper/tabutil"
	"github.com/ystia/yorc/v4/rest"
	"io/ioutil"
	"net/http"
)

func init() {
	var location string
	hpLocationsCmd := &cobra.Command{
		Use:   "locations",
		Short: "List hosts pools locations",
		Long:  `Lists hosts pools locations managed by this Yorc cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			colorize := !noColor
			client, err := httputil.GetClient(clientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}
			request, err := client.NewRequest("GET", "/hosts_pool", nil)
			if err != nil {
				httputil.ErrExit(err)
			}
			q := request.URL.Query()
			request.URL.RawQuery = q.Encode()
			request.Header.Add("Accept", "application/json")
			response, err := client.Do(request)
			if err != nil {
				httputil.ErrExit(err)
			}
			defer response.Body.Close()
			httputil.HandleHTTPStatusCode(response, "", "hosts pool locations", http.StatusOK)
			var coll rest.HostsPoolLocations
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				httputil.ErrExit(err)
			}
			err = json.Unmarshal(body, &coll)
			if err != nil {
				httputil.ErrExit(err)
			}

			locationsTable := tabutil.NewTable()
			locationsTable.AddHeaders("Name")
			for _, location := range coll.Locations {
				locationsTable.AddRow(location)
			}
			if colorize {
				defer color.Unset()
			}
			fmt.Printf("Host pools for location %q:\n", location)
			fmt.Println(locationsTable.Render())
			return nil
		},
	}
	hostsPoolCmd.AddCommand(hpLocationsCmd)
}
