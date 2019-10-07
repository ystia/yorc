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
	"net/http"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ystia/yorc/v4/commands/httputil"
)

func init() {
	var location string
	var delCmd = &cobra.Command{
		Use:   "delete <hostname> [hostname...]",
		Short: "Delete hosts of a specified location",
		Long:  `Delete hosts of the hosts pool of a specified location managed by this Yorc cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := httputil.GetClient(clientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}
			return deleteHost(client, args, location)
		},
	}
	delCmd.Flags().StringVarP(&location, "location", "l", "", "Need to provide the specified hosts pool location name")
	hostsPoolCmd.AddCommand(delCmd)
}

func deleteHost(client httputil.HTTPClient, args []string, location string) error {
	if len(args) < 1 {
		return errors.Errorf("Expecting at least one hostname (got %d parameters)", len(args))
	}
	if location == "" {
		return errors.Errorf("Expecting a hosts pool location name")
	}
	for i := range args {
		sendDeleteHostRequest(client, args[i], location)
	}
	return nil
}

func sendDeleteHostRequest(client httputil.HTTPClient, hostname, location string) {
	request, err := client.NewRequest("DELETE", "/hosts_pool/"+location+"/"+hostname, nil)
	if err != nil {
		httputil.ErrExit(err)
	}
	response, err := client.Do(request)
	if err != nil {
		httputil.ErrExit(err)
	}
	defer response.Body.Close()
	httputil.HandleHTTPStatusCode(response, hostname, "host pool", http.StatusOK)
}
