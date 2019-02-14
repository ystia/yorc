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
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ystia/yorc/v3/commands"
	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/helper/sliceutil"
	"github.com/ystia/yorc/v3/helper/tabutil"
	"github.com/ystia/yorc/v3/rest"
)

// Internal constants for operations on hosts pool
const (
	hostDeletion = iota
	hostUpdate
	hostCreation
	hostError
	hostList
	hostNoOperation
)

func init() {
	commands.RootCmd.AddCommand(hostsPoolCmd)
	commands.ConfigureYorcClientCommand(hostsPoolCmd, hpViper, &cfgFile, &noColor)
}

var hpViper = viper.New()
var clientConfig config.Client

var noColor bool
var cfgFile string

var hostsPoolCmd = &cobra.Command{
	Use:           "hostspool",
	Aliases:       []string{"hostpool", "hostsp", "hpool", "hp"},
	Short:         "Perform commands on hosts pool",
	Long:          `Allow to add, update and delete hosts pool`,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		clientConfig = commands.GetYorcClientConfig(hpViper, cfgFile)
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
	case hostCreation:
		return color.New(color.FgHiGreen, color.Bold).SprintFunc()(text)
	case hostUpdate:
		return color.New(color.FgHiYellow, color.Bold).SprintFunc()(text)
	case hostDeletion:
		return color.New(color.FgHiRed, color.Bold).SprintFunc()(text)
	case hostError:
		return color.New(color.FgHiRed, color.Bold).SprintFunc()(text)
	default:
		return text
	}
}

// getColoredHostStatus returns a status colored according to its value,
// used by 'hosts list' operations
func getColoredHostStatus(colorize bool, status string) string {
	if !colorize {
		return status
	}
	switch {
	case strings.ToLower(status) == "free":
		return color.New(color.FgHiGreen, color.Bold).SprintFunc()(status)
	case strings.ToLower(status) == "allocated":
		return color.New(color.FgHiYellow, color.Bold).SprintFunc()(status)
	default:
		return color.New(color.FgHiRed, color.Bold).SprintFunc()(status)
	}
}

// padSlices is padding as necessary slices in arguments so that both slices
// have the same number of elements
func padSlices(slice1 []string, slice2 []string) ([]string, []string) {
	slice1Size := len(slice1)
	slice2Size := len(slice2)

	if slice1Size > slice2Size {
		for i := 0; i < slice1Size-slice2Size; i++ {
			slice2 = append(slice2, "")
		}
	} else {
		for i := 0; i < slice2Size-slice1Size; i++ {
			slice1 = append(slice1, "")
		}
	}

	return slice1, slice2
}

// AddRow adds a row to a table, with text colored according to the operation
// longTable specifies table with all headers
func addRow(table tabutil.Table, colorize bool, operation int, host *rest.Host, fullTable bool) {
	colNumber := 3
	if fullTable {
		colNumber = 6
	}

	statusString := ""
	if &host.Status != nil {
		statusString = host.Status.String()
	}

	allocationsSubRows := make([]string, 0)
	for _, alloc := range host.Allocations {
		allocationsSubRows = append(allocationsSubRows, strings.Split(alloc.String(), ",")...)
	}

	connectionSubRows := strings.Split(host.Connection.String(), ",")
	var labelSubRows []string
	if host.Labels != nil {
		labelSubRows = strings.Split(toPrintableLabels(host.Labels), ",")
		sort.Strings(labelSubRows)
	}

	sliceutil.PadSlices("", &allocationsSubRows, &connectionSubRows, &labelSubRows)
	subRowsNumber := len(connectionSubRows)

	// Add rows, one for each sub-column
	for i := 0; i < subRowsNumber; i++ {
		coloredColumns := make([]interface{}, colNumber)
		coloredColumns[0] = getColoredText(colorize, host.Name, operation)
		coloredColumns[1] = getColoredText(colorize,
			strings.TrimSpace(connectionSubRows[i]), operation)
		j := 2
		if fullTable {
			// For a list operation, color the status according to its value
			if operation == hostList {
				coloredColumns[j] = getColoredHostStatus(colorize, statusString)
			} else {
				coloredColumns[j] = getColoredText(colorize, statusString, operation)
			}
			j++
		}

		if fullTable {
			coloredColumns[j] = getColoredText(colorize,
				strings.TrimSpace(allocationsSubRows[i]), operation)
			j++
		}

		if fullTable {
			coloredColumns[j] = getColoredText(colorize, host.Message, operation)
			j++
		}
		coloredColumns[j] = getColoredText(colorize,
			strings.TrimSpace(labelSubRows[i]), operation)
		table.AddRow(coloredColumns...)
		if i == 0 {
			// Don't repeat single column values in sub-columns
			host.Name = ""
			statusString = ""
			host.Message = ""
		}
	}
}

func addHostInErrorRow(table tabutil.Table, colorize bool, operation int, host *rest.Host) {
	colNumber := 4

	statusString := ""
	if &host.Status != nil {
		statusString = host.Status.String()
	}

	connectionSubRows := strings.Split(host.Connection.String(), ",")
	var labelSubRows []string
	if host.Labels != nil {
		labelSubRows = strings.Split(toPrintableLabels(host.Labels), ",")
		sort.Strings(labelSubRows)
	}

	subRowsNumber := len(connectionSubRows)
	// Add rows, one for each sub-column
	for i := 0; i < subRowsNumber; i++ {
		coloredColumns := make([]interface{}, colNumber)
		coloredColumns[0] = getColoredText(colorize, host.Name, operation)
		coloredColumns[1] = getColoredText(colorize,
			strings.TrimSpace(connectionSubRows[i]), operation)

		// For a list operation, color the status according to its value
		if operation == hostList {
			coloredColumns[2] = getColoredHostStatus(colorize, statusString)
		} else {
			coloredColumns[2] = getColoredText(colorize, statusString, operation)
		}

		coloredColumns[3] = getColoredText(colorize, host.Message, operation)
		table.AddRow(coloredColumns...)
		if i == 0 {
			// Don't repeat single column values in sub-columns
			host.Name = ""
			statusString = ""
			host.Message = ""
		}
	}
}
