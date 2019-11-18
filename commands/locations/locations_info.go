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
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/ystia/yorc/v4/commands/httputil"
	"github.com/ystia/yorc/v4/helper/tabutil"
)

func init() {
	var infoCmd = &cobra.Command{
		Use:   "info <locationName>",
		Short: "Gets a location definition",
		Long:  `Gets the type and the properties of a location.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := httputil.GetClient(ClientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}
			return getLocationInfo(client, args)
		},
	}
	LocationsCmd.AddCommand(infoCmd)
}

func getLocationInfo(client httputil.HTTPClient, args []string) error {
	if len(args) != 1 {
		return errors.Errorf("Expecting one parameter for location name (got %d parameters)", len(args))
	}
	locConfig, err := getLocationConfig(client, args[0])
	if err != nil {
		return err
	}

	locationTable := tabutil.NewTable()
	locationTable.AddHeaders("Name", "Type", "Properties")
	addRow(locationTable, false, locationShow, locConfig)
	fmt.Println(locationTable.Render())
	return nil
}
