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

package deployments

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"net/http"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/ystia/yorc/commands/httputil"
	"github.com/ystia/yorc/helper/tabutil"
	"github.com/ystia/yorc/rest"
)

func init() {
	DeploymentsCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List deployments",
	Long:  `List active deployments. Giving their id and status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		colorize := !NoColor
		client, err := httputil.GetClient()
		if err != nil {
			httputil.ErrExit(err)
		}
		request, err := client.NewRequest("GET", "/deployments", nil)
		if err != nil {
			httputil.ErrExit(err)
		}
		request.Header.Add("Accept", "application/json")
		response, err := client.Do(request)
		defer response.Body.Close()
		if err != nil {
			httputil.ErrExit(err)
		}
		httputil.HandleHTTPStatusCode(response, "", "deployment", http.StatusOK)
		var deps rest.DeploymentsCollection
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			httputil.ErrExit(err)
		}
		err = json.Unmarshal(body, &deps)
		if err != nil {
			httputil.ErrExit(err)
		}

		depsTable := tabutil.NewTable()
		depsTable.AddHeaders("Id", "Status")
		for _, depLink := range deps.Deployments {
			if depLink.Rel == rest.LinkRelDeployment {
				var dep rest.Deployment

				err = httputil.GetJSONEntityFromAtomGetRequest(client, depLink, &dep)
				if err != nil {
					httputil.ErrExit(err)
				}
				depsTable.AddRow(dep.ID, getColoredDeploymentStatus(colorize, dep.Status))
			}
		}
		if colorize {
			defer color.Unset()
		}
		fmt.Println("Deployments:")
		fmt.Println(depsTable.Render())
		return nil
	},
}
