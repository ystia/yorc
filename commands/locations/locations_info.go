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
	"github.com/ystia/yorc/v4/locations/adapter"
)

func init() {
	LocationsCmd.AddCommand(infoCmd)
}

var infoCmd = &cobra.Command{
	Use:   "info <locationName>",
	Short: "Get a location definition",
	Long:  `Get a location definition.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := httputil.GetClient(ClientConfig)
		if err != nil {
			httputil.ErrExit(err)
		}
		return getLocationInfo(client, args)
	},
}

func getLocationInfo(client httputil.HTTPClient, args []string) error {
	if len(args) != 1 {
		return errors.Errorf("Expecting one location name (got %d parameters)", len(args))
	}

	locConfig, err := getLocationConfig(client, args[0])
	if err != nil {
		return err
	}

	if locConfig.Type != adapter.AdaptedLocationType {
		fmt.Println("Location " + locConfig.Name + " has type " + locConfig.Type)
		fmt.Println("Location " + locConfig.Name + " configuration properties: ")
		locProps := locConfig.Properties
		propKeys := locProps.Keys()
		for i := 0; i < len(propKeys); i++ {
			propValue := locProps.Get(propKeys[i])
			value := fmt.Sprintf("%v", propValue)
			prop := propKeys[i] + ": " + value
			fmt.Println(prop)
		}
	}
	return nil
}
