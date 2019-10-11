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

package workflows

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ystia/yorc/v4/commands/deployments"
	"github.com/ystia/yorc/v4/commands/httputil"
	"github.com/ystia/yorc/v4/rest"
)

func init() {
	var wfListCmd = &cobra.Command{
		Use:     "list <id>",
		Short:   "List workflows defined on a given deployment <id>",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting an id (got %d parameters)", len(args))
			}
			client, err := httputil.GetClient(deployments.ClientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}

			request, err := client.NewRequest("GET", fmt.Sprintf("/deployments/%s/workflows", args[0]), nil)
			if err != nil {
				httputil.ErrExit(err)
			}
			request.Header.Add("Accept", "application/json")
			response, err := client.Do(request)
			if err != nil {
				httputil.ErrExit(err)
			}
			defer response.Body.Close()
			httputil.HandleHTTPStatusCode(response, args[0], "deployment", http.StatusOK)

			var wfs rest.WorkflowsCollection
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				httputil.ErrExit(err)
			}
			err = json.Unmarshal(body, &wfs)
			if err != nil {
				httputil.ErrExit(err)
			}

			for _, wfLink := range wfs.Workflows {
				if wfLink.Rel == rest.LinkRelWorkflow {
					fmt.Println(path.Base(wfLink.Href))
				}
			}
			return nil
		},
	}
	workflowsCmd.AddCommand(wfListCmd)
}
