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
	"encoding/json"
	"fmt"
	"io/ioutil"

	"net/http"

	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/ystia/yorc/commands/httputil"
	"github.com/ystia/yorc/helper/tabutil"
	"github.com/ystia/yorc/rest"
)

func init() {
	var filters []string
	hpListCmd := &cobra.Command{
		Use:   "list",
		Short: "List hosts pools",
		Long:  `Lists hosts of the hosts pool managed by this Yorc cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			colorize := !noColor
			client, err := httputil.GetClient()
			if err != nil {
				httputil.ErrExit(err)
			}
			request, err := client.NewRequest("GET", "/hosts_pool", nil)
			if err != nil {
				httputil.ErrExit(err)
			}
			q := request.URL.Query()
			for i := range filters {
				q.Add("filter", filters[i])
			}
			request.URL.RawQuery = q.Encode()
			request.Header.Add("Accept", "application/json")
			response, err := client.Do(request)
			if err != nil {
				httputil.ErrExit(err)
			}
			defer response.Body.Close()
			httputil.HandleHTTPStatusCode(response, "", "host pool", http.StatusOK)
			var hostsColl rest.HostsCollection
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				httputil.ErrExit(err)
			}
			err = json.Unmarshal(body, &hostsColl)
			if err != nil {
				httputil.ErrExit(err)
			}

			hostsTable := tabutil.NewTable()
			hostsTable.AddHeaders("Name", "Connection", "Status", "Message", "Labels")
			for _, hostLink := range hostsColl.Hosts {
				if hostLink.Rel == rest.LinkRelHost {
					var host rest.Host
					var labelsList string

					err = httputil.GetJSONEntityFromAtomGetRequest(client, hostLink, &host)
					if err != nil {
						httputil.ErrExit(err)
					}

					for k, v := range host.Labels {
						if labelsList != "" {
							labelsList += ", "
						}
						labelsList += fmt.Sprintf("%s:%s", k, v)
					}

					hostsTable.AddRow(host.Name, host.Connection.String(), getColoredHostStatus(colorize, host.Status.String()), host.Message, labelsList)
				}
			}
			if colorize {
				defer color.Unset()
			}
			fmt.Println("Host pools:")
			fmt.Println(hostsTable.Render())
			return nil
		},
	}
	hpListCmd.Flags().StringSliceVarP(&filters, "filter", "f", nil, "Filter hosts based on their labels. May be specified several time, filters are joined by a logical 'and'. See the documentation for the filters grammar.")
	hostsPoolCmd.AddCommand(hpListCmd)
}

func getColoredHostStatus(colorize bool, status string) string {
	if !colorize {
		return status
	}
	switch {
	case strings.ToLower(status) == "free":
		return color.New(color.FgHiGreen, color.Bold).SprintFunc()(status)
	case strings.ToLower(status) == "allocated":
		return color.New(color.FgHiYellow, color.Bold).SprintFunc()(status)
	default:
		return color.New(color.FgHiRed, color.Bold).SprintFunc()(status)
	}
}
