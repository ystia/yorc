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
	"io/ioutil"
	"net/http"
	"path"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ystia/yorc/v4/commands/httputil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/rest"
)

func init() {
	var force bool
	var purgeCmd = &cobra.Command{
		Use:   "purge <id>",
		Short: "purge a deployment",
		Long: `Purge a deployment <id>. This deployment should be in UNDEPLOYED status.
If an error is encountered the purge process is stopped and the deployment status is set
to PURGE_FAILED.

A purge may be run in force mode. In this mode Yorc does not check if the deployment is in
UNDEPLOYED status or even if the deployment exist. Moreover, in force mode the purge process
doesn't fail-fast and try to delete as much as it can. An report with encountered errors is
produced at the end of the process.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting a deployment id (got %d parameters)", len(args))
			}

			client, err := httputil.GetClient(ClientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}
			deploymentID := args[0]

			return postPurgeRequest(client, deploymentID, force)

		},
	}
	purgeCmd.Flags().BoolVarP(&force, "force", "f", false, "Force purge of a deployment ignoring states checks and any errors. This should be use with extrem caution to cleanup environment.")
	DeploymentsCmd.AddCommand(purgeCmd)
}

func postPurgeRequest(client httputil.HTTPClient, deploymentID string, force bool) error {
	request, err := client.NewRequest("POST", path.Join("/deployments", deploymentID, "purge"), nil)
	if err != nil {
		httputil.ErrExit(errors.Wrap(err, httputil.YorcAPIDefaultErrorMsg))
	}

	query := request.URL.Query()
	if force {
		query.Set("force", strconv.FormatBool(force))
	}
	request.URL.RawQuery = query.Encode()
	request.Header.Add("Accept", "application/json")
	log.Debugf("POST: %s", request.URL.String())

	response, err := client.Do(request)
	if err != nil {
		httputil.ErrExit(errors.Wrap(err, httputil.YorcAPIDefaultErrorMsg))
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		ioutil.ReadAll(response.Body)
		return nil
	}
	var errs rest.Errors
	bodyContent, _ := ioutil.ReadAll(response.Body)
	json.Unmarshal(bodyContent, &errs)

	return nil
}
