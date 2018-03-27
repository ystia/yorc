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
	"github.com/ystia/yorc/commands/httputil"
)

func init() {
	var delCmd = &cobra.Command{
		Use:   "delete <hostname> [hostname...]",
		Short: "Delete hosts from hosts pool",
		Long:  `Delete hosts of the hosts pool managed by this Yorc cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.Errorf("Expecting a hostname (got %d parameters)", len(args))
			}
			client, err := httputil.GetClient()
			if err != nil {
				httputil.ErrExit(err)
			}
			for i := range args {
				request, err := client.NewRequest("DELETE", "/hosts_pool/"+args[i], nil)
				if err != nil {
					httputil.ErrExit(err)
				}

				response, err := client.Do(request)
				if err != nil {
					httputil.ErrExit(err)
				}
				defer response.Body.Close()

				httputil.HandleHTTPStatusCode(response, args[0], "host pool", http.StatusOK)
			}
			return nil
		},
	}
	hostsPoolCmd.AddCommand(delCmd)
}
