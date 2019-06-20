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

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ystia/yorc/v4/commands/httputil"
	"github.com/ystia/yorc/v4/helper/tabutil"
	"github.com/ystia/yorc/v4/rest"
)

func init() {
	var getCmd = &cobra.Command{
		Use:   "info <hostname>",
		Short: "Get host pool info",
		Long:  `Gets the description of a host of the hosts pool managed by this Yorc cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			colorize := !noColor
			if len(args) != 1 {
				return errors.Errorf("Expecting a hostname (got %d parameters)", len(args))
			}
			client, err := httputil.GetClient(clientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}

			request, err := client.NewRequest("GET", "/hosts_pool/"+args[0], nil)
			request.Header.Add("Accept", "application/json")
			if err != nil {
				httputil.ErrExit(err)
			}

			response, err := client.Do(request)
			if err != nil {
				httputil.ErrExit(err)
			}
			defer response.Body.Close()

			httputil.HandleHTTPStatusCode(response, args[0], "host pool", http.StatusOK)
			var host rest.Host
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				httputil.ErrExit(err)
			}
			err = json.Unmarshal(body, &host)
			if err != nil {
				httputil.ErrExit(err)
			}

			hostsTable := tabutil.NewTable()
			hostsTable.AddHeaders("Name", "Connection", "Status", "Allocations", "Message", "Labels")
			addRow(hostsTable, colorize, hostList, &host, true)
			if colorize {
				defer color.Unset()
			}
			fmt.Println("Host pool:")
			fmt.Println(hostsTable.Render())
			return nil
		},
	}
	hostsPoolCmd.AddCommand(getCmd)
}
