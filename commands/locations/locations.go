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

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ystia/yorc/v4/commands"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/tabutil"
	"github.com/ystia/yorc/v4/rest"
)

func init() {
	commands.RootCmd.AddCommand(LocationsCmd)
	commands.ConfigureYorcClientCommand(LocationsCmd, DepViper, &cfgFile, &noColor)
}

// DepViper is the viper configuration for the locations command and its children
var DepViper = viper.New()

// ClientConfig is the Yorc client configuration resolved by cobra/viper
var ClientConfig config.Client
var cfgFile string

// noColor returns true if no-color option is set
var noColor bool

// Internal constants for operations on locations
const (
	locationDeletion = iota
	locationUpdate
	locationCreation
	locationList
)

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

// getColoredText returns a text colored according to the operation in
// argument :
// - red for a deletion
// - yellow for an update (bold for the new version, regular for the old one)
// - green for a creation
func getColoredText(colorize bool, text string, operation int) string {
	if !colorize {
		return text
	}
	switch operation {
	case locationCreation:
		return color.New(color.FgHiGreen, color.Bold).SprintFunc()(text)
	case locationUpdate:
		return color.New(color.FgHiYellow, color.Bold).SprintFunc()(text)
	case locationDeletion:
		return color.New(color.FgHiRed, color.Bold).SprintFunc()(text)
	default:
		return text
	}
}

// AddRow adds a row to a table, with text colored according to the operation
// longTable specifies table with all headers
func addRow(table tabutil.Table, colorize bool, operation int, lConfig rest.LocationConfiguration) {
	colNumber := 3

	coloredColumns := make([]interface{}, colNumber)
	locProps := lConfig.Properties
	propKeys := locProps.Keys()
	for i := 0; i < len(propKeys); i++ {
		propValue := locProps.Get(propKeys[i])
		value := fmt.Sprintf("%v", propValue)
		prop := propKeys[i] + ": " + value
		if i == 0 {
			coloredColumns[0] = getColoredText(colorize, lConfig.Name, operation)
			coloredColumns[1] = getColoredText(colorize, lConfig.Type, operation)
			coloredColumns[2] = getColoredText(colorize, prop, operation)
		} else {
			coloredColumns[0] = getColoredText(colorize, "", operation)
			coloredColumns[1] = getColoredText(colorize, "", operation)
			coloredColumns[2] = getColoredText(colorize, prop, operation)

		}
		table.AddRow(coloredColumns...)
	}

}
