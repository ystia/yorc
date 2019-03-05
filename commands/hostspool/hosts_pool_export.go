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

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/ystia/yorc/v3/commands/httputil"
	"github.com/ystia/yorc/v3/rest"
)

func init() {
	var outputFormat string
	var filePath string
	hpExportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export hosts pool configuration",
		Long:  `Export hosts pool configuration as a YAML or JSON representation`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := httputil.GetClient(clientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}
			request, err := client.NewRequest("GET", "/hosts_pool", nil)
			if err != nil {
				httputil.ErrExit(err)
			}

			request.Header.Add("Accept", "application/json")
			response, err := client.Do(request)
			defer response.Body.Close()
			if err != nil {
				httputil.ErrExit(err)
			}
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

			pool := rest.HostsPoolRequest{}
			for _, hostLink := range hostsColl.Hosts {
				if hostLink.Rel == rest.LinkRelHost {
					var restHost rest.Host
					err = httputil.GetJSONEntityFromAtomGetRequest(client, hostLink, &restHost)
					if err != nil {
						httputil.ErrExit(err)
					}

					host := rest.HostConfig{
						Name:       restHost.Name,
						Connection: restHost.Connection,
						Labels:     restHost.Labels,
					}
					pool.Hosts = append(pool.Hosts, host)
				}
			}

			// Marshal according to the specified output format
			outputFormat = strings.ToLower(strings.TrimSpace(outputFormat))
			var bSlice []byte
			if outputFormat == "json" {
				bSlice, err = json.MarshalIndent(pool, "", "    ")
			} else {
				bSlice, err = yaml.Marshal(pool)
			}
			if err != nil {
				httputil.ErrExit(err)
			}

			if filePath != "" {
				err = ioutil.WriteFile(filePath, bSlice, 0644)
				if err != nil {
					httputil.ErrExit(err)
				}
			} else {
				output := string(bSlice)
				fmt.Println(output)
			}

			return nil
		},
	}
	hpExportCmd.Flags().StringVarP(&outputFormat, "output", "o", "yaml", "Output format: yaml, json")
	hpExportCmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to a file where to store the output")
	hostsPoolCmd.AddCommand(hpExportCmd)
}
