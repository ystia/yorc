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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ystia/yorc/v4/commands"
	"github.com/ystia/yorc/v4/config"
)

func init() {
	commands.RootCmd.AddCommand(LocationsCmd)
	commands.ConfigureYorcClientCommand(LocationsCmd, DepViper, &cfgFile, &NoColor)
}

// DepViper is the viper configuration for the locations command and its children
var DepViper = viper.New()

// ClientConfig is the Yorc client configuration resolved by cobra/viper
var ClientConfig config.Client
var cfgFile string

// NoColor returns true if no-color option is set
var NoColor bool

// LocationsCmd is the locations-based command
var LocationsCmd = &cobra.Command{
	Use:           "locations",
	Aliases:       []string{"locs", "loc", "l"},
	Short:         "Perform commands on locations",
	Long:          `Perform different commands on locations`,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ClientConfig = commands.GetYorcClientConfig(DepViper, cfgFile)
	},
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Help()
		if err != nil {
			fmt.Print(err)
		}
	},
}
