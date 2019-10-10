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
	"fmt"
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
	var labels []string

	var addCmd = &cobra.Command{
		Use:   "add -l <locationName> <hostname>",
		Short: "Add host to the pool of a specified location",
		Long:  `Adds a host to the hosts pool of a specified location managed by this Yorc cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := httputil.GetClient(clientConfig)
			if err != nil {
				return err
			}
			return addHost(client, args, location, jsonParam, privateKey, password, user, host, port, labels)
		},
	}
	addCmd.Flags().StringVarP(&location, "location", "l", "", "Need to provide the specified hosts pool location name")
	addCmd.Flags().StringVarP(&jsonParam, "data", "d", "", "Need to provide the JSON format of the host pool")
	addCmd.Flags().StringVarP(&user, "user", "", "root", "User used to connect to the host")
	addCmd.Flags().StringVarP(&host, "host", "", "", "Hostname or ip address used to connect to the host. (defaults to the hostname in the hosts pool)")
	addCmd.Flags().Uint64VarP(&port, "port", "", 22, "Port used to connect to the host.")
	addCmd.Flags().StringVarP(&privateKey, "key", "k", "", "Need to provide a private key or a password for the host pool")
	addCmd.Flags().StringVarP(&password, "password", "p", "", "Need to provide a private key or a password for the host pool")
	addCmd.Flags().StringSliceVarP(&labels, "label", "", nil, "Label in form 'key=value' to add to the host. May be specified several time.")

	hostsPoolCmd.AddCommand(addCmd)
}

func addHost(client httputil.HTTPClient, args []string, location, jsonParam, privateKey, password, user, host string, port uint64, labels []string) error {
	if len(args) != 1 {
		return errors.Errorf("Expecting a hostname (got %d parameters)", len(args))
	}
	if location == "" {
		return errors.Errorf("Expecting a hosts pool location name")
	}
	if len(jsonParam) == 0 && len(privateKey) == 0 && len(password) == 0 {
		return errors.Errorf("You need to provide either JSON with connection information or private key or password for the host pool")
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
		for _, l := range labels {
			parts := strings.SplitN(l, "=", 2)
			me := rest.MapEntry{Name: parts[0]}
			if len(parts) == 2 {
				me.Value = parts[1]
			}
			hostRequest.Labels = append(hostRequest.Labels, me)
		}
		tmp, err := json.Marshal(hostRequest)
		if err != nil {
			return errors.Wrapf(err, "failed to unmarshall json body")
		}

		jsonParam = string(tmp)
	}

	request, err := client.NewRequest("PUT", "/hosts_pool/"+location+"/"+args[0], bytes.NewBuffer([]byte(jsonParam)))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	httputil.HandleHTTPStatusCode(response, args[0], "host pool", http.StatusCreated)
	fmt.Println("Command submitted. path :", response.Header.Get("Location"))
	return nil
}
