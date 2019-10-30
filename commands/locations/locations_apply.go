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
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ystia/yorc/v4/commands/httputil"
	"github.com/ystia/yorc/v4/rest"
)

func init() {
	var autoApprove bool
	var applyCmd = &cobra.Command{
		Use:   "apply <path to locations configuration file>",
		Short: "Apply a locations configuration file",
		Long: `Apply a locations configuration provided in the file passed in argument. 
		This file should contain a YAML or a JSON description.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := httputil.GetClient(ClientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}
			return applyLocationsConfig(client, args, autoApprove)
		},
	}
	applyCmd.PersistentFlags().BoolVarP(&autoApprove, "auto-approve", "", false,
		"Skip interactive approval before applying this new locations configuration.")
	LocationsCmd.AddCommand(applyCmd)
}

func applyLocationsConfig(client httputil.HTTPClient, args []string, autoApprove bool) error {
	//colorize := !noColor
	if len(args) != 1 {
		return errors.Errorf("Expecting a path to a file (got %d parameters)", len(args))
	}
	fileInfo, err := os.Stat(args[0])
	if err != nil {
		return err
	}
	if fileInfo.IsDir() {
		return errors.Errorf("Expecting a path to a file (%s is a directory)", args[0])
	}

	// Read config file, viper will accept indifferently a yaml or json
	// format
	v := viper.New()
	v.SetConfigFile(args[0])
	err = v.ReadInConfig()
	if err != nil {
		return err
	}

	var locationsRequest rest.LocationsCollection
	err = v.Unmarshal(&locationsRequest)
	if err != nil {
		return err
	}

	return errors.Errorf("Got new LocationsCollection with %v locations", len(locationsRequest.Locations))
	//return nil
}
