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
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/ystia/yorc/v4/commands/httputil"
	"github.com/ystia/yorc/v4/rest"
)

func init() {
	var location string
	var outputFormat string
	var filePath string
	hpExportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export hosts pool configuration for a specified location",
		Long:  `Export hosts pool configuration for a specified location as a YAML or JSON representation`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := httputil.GetClient(clientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}
			return exportHostsPool(client, args, location, outputFormat, filePath)
		},
	}
	hpExportCmd.Flags().StringVarP(&location, "location", "l", "", "Need to provide the specified hosts pool location name")
	hpExportCmd.Flags().StringVarP(&outputFormat, "output", "o", "yaml", "Output format: yaml, json")
	hpExportCmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to a file where to store the output")
	hostsPoolCmd.AddCommand(hpExportCmd)
}

func exportHostsPool(client httputil.HTTPClient, args []string, location, outputFormat, filePath string) error {
	if location == "" {
		return errors.Errorf("Expecting a hosts pool location name")
	}
	request, err := client.NewRequest("GET", "/hosts_pool/"+location, nil)
	if err != nil {
		return err
	}

	request.Header.Add("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	httputil.HandleHTTPStatusCode(response, "", "host pool", http.StatusOK)
	var hostsColl rest.HostsCollection
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, &hostsColl)
	if err != nil {
		return err
	}

	pool := rest.HostsPoolRequest{}
	for _, hostLink := range hostsColl.Hosts {
		if hostLink.Rel == rest.LinkRelHost {
			var restHost rest.Host
			err = httputil.GetJSONEntityFromAtomGetRequest(client, hostLink, &restHost)
			if err != nil {
				return err
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
		return err
	}

	if filePath != "" {
		err = ioutil.WriteFile(filePath, bSlice, 0644)
		if err != nil {
			return err
		}
	} else {
		output := string(bSlice)
		fmt.Println(output)
	}

	return nil
}
