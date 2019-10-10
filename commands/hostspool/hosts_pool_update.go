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
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ystia/yorc/v4/commands/httputil"
	"github.com/ystia/yorc/v4/prov/hostspool"
	"github.com/ystia/yorc/v4/rest"
)

func init() {
	var location string
	var jsonParam string
	var privateKey string
	var password string
	var user string
	var host string
	var port uint64
	var labelsAdd []string
	var labelsRemove []string

	var updCmd = &cobra.Command{
		Use:   "update -l <locationName> <hostname>",
		Short: "Update host pool of a specified location",
		Long:  `Update labels list or connection of a host of the hosts pool of a specified location managed by this Yorc cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := httputil.GetClient(clientConfig)
			if err != nil {
				return err
			}
			return updateHost(client, args, location, jsonParam, privateKey, password, user, host, port, labelsAdd, labelsRemove)
		},
	}
	updCmd.Flags().StringVarP(&location, "location", "l", "", "Need to provide the specified hosts pool location name")
	updCmd.Flags().StringVarP(&jsonParam, "data", "d", "", "Need to provide the JSON format of the updated host pool")
	updCmd.Flags().StringVarP(&user, "user", "", "", "User used to connect to the host")
	updCmd.Flags().StringVarP(&host, "host", "", "", "Hostname or ip address used to connect to the host. (defaults to the hostname in the hosts pool)")
	updCmd.Flags().Uint64VarP(&port, "port", "", 0, "Port used to connect to the host.")
	updCmd.Flags().StringVarP(&privateKey, "key", "k", "", `At any time a host of the pool should have at least one of private key or password. To delete a registered password use the "-" character.`)
	updCmd.Flags().StringVarP(&password, "password", "p", "", `At any time a host of the pool should have at least one of private key or password. To delete a registered private key use the "-" character.`)
	updCmd.Flags().StringSliceVarP(&labelsAdd, "add-label", "", nil, "Add a label in form 'key=value' to the host. May be specified several time.")
	updCmd.Flags().StringSliceVarP(&labelsRemove, "remove-label", "", nil, "Remove a label from the host. May be specified several time.")

	hostsPoolCmd.AddCommand(updCmd)
}

func updateHost(client httputil.HTTPClient, args []string, location, jsonParam, privateKey, password, user, host string, port uint64, labelsAdd, labelsRemove []string) error {
	if len(args) != 1 {
		return errors.Errorf("Expecting a hostname (got %d parameters)", len(args))
	}
	if location == "" {
		return errors.Errorf("Expecting a hosts pool location name")
	}
	if len(jsonParam) == 0 {
		var hostRequest rest.HostRequest
		hostRequest.Connection = &hostspool.Connection{
			User:       user,
			Host:       host,
			Port:       port,
			Password:   password,
			PrivateKey: privateKey,
		}
		for _, l := range labelsAdd {
			parts := strings.SplitN(l, "=", 2)
			me := rest.MapEntry{Op: rest.MapEntryOperationAdd, Name: parts[0]}
			if len(parts) == 2 {
				me.Value = parts[1]
			}
			hostRequest.Labels = append(hostRequest.Labels, me)
		}
		for _, l := range labelsRemove {
			hostRequest.Labels = append(hostRequest.Labels, rest.MapEntry{Op: rest.MapEntryOperationRemove, Name: l})
		}
		tmp, err := json.Marshal(hostRequest)
		if err != nil {
			return err
		}

		jsonParam = string(tmp)
	}

	request, err := client.NewRequest("PATCH", "/hosts_pool/"+location+"/"+args[0], bytes.NewBuffer([]byte(jsonParam)))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	httputil.HandleHTTPStatusCode(response, args[0], "host pool", http.StatusOK)
	return nil
}
