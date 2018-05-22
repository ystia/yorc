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
	"log"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/ystia/yorc/commands/httputil"
	"github.com/ystia/yorc/prov/hostspool"
	"github.com/ystia/yorc/rest"
)

func init() {
	var jsonParam string
	var privateKey string
	var password string
	var user string
	var host string
	var port uint64
	var labelsAdd []string
	var labelsRemove []string

	var updCmd = &cobra.Command{
		Use:   "update <hostname>",
		Short: "Update host pool",
		Long:  `Update labels list or connection of a host of the hosts pool managed by this Yorc cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting a hostname (got %d parameters)", len(args))
			}
			client, err := httputil.GetClient(clientConfig)
			if err != nil {
				httputil.ErrExit(err)
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
					log.Panic(err)
				}

				jsonParam = string(tmp)
			}

			request, err := client.NewRequest("PATCH", "/hosts_pool/"+args[0], bytes.NewBuffer([]byte(jsonParam)))
			if err != nil {
				httputil.ErrExit(err)
			}
			request.Header.Add("Content-Type", "application/json")

			response, err := client.Do(request)
			if err != nil {
				httputil.ErrExit(err)
			}
			defer response.Body.Close()

			httputil.HandleHTTPStatusCode(response, args[0], "host pool", http.StatusOK)
			return nil
		},
	}
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
