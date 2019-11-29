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

package locations

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/ystia/yorc/v4/commands/httputil"
	"github.com/ystia/yorc/v4/rest"
)

func init() {
	var jsonParam string

	var addCmd = &cobra.Command{
		Use:   "add ",
		Short: "Add a location definition",
		Long:  `Add a location definition by providing a JSON description.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := httputil.GetClient(ClientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}
			return addLocation(client, jsonParam)
		},
	}

	addCmd.Flags().StringVarP(&jsonParam, "data", "d", "", "Need to provide the JSON format of location definition")

	LocationsCmd.AddCommand(addCmd)
}

func addLocation(client httputil.HTTPClient, jsonParam string) error {
	if len(jsonParam) == 0 {
		return errors.Errorf("You need to provide JSON data with location definition")
	}

	var locConfig rest.LocationConfiguration
	err := json.Unmarshal([]byte(jsonParam), &locConfig)
	if err != nil {
		return err
	}

	if len(locConfig.Name) == 0 {
		return errors.Errorf("You need to provide location name in definition")
	}

	locName, err := putLocationConfig(client, locConfig)
	if err != nil {
		return err
	}
	fmt.Println("Location ", locName, " created")

	return nil
}
