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
	"github.com/ystia/yorc/commands"
	"github.com/ystia/yorc/helper/tabutil"
	"github.com/ystia/yorc/prov/hostspool"
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
	setHostsPoolConfig()
}

var noColor bool

var hostsPoolCmd = &cobra.Command{
	Use:           "hostspool",
	Aliases:       []string{"hostpool", "hostsp", "hpool", "hp"},
	Short:         "Perform commands on hosts pool",
	Long:          `Allow to add, update and delete hosts pool`,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Help()
		if err != nil {
			fmt.Print(err)
		}
	},
}

func setHostsPoolConfig() {
	hostsPoolCmd.PersistentFlags().StringP("yorc-api", "j", "localhost:8800", "specify the host and port used to join the Yorc' REST API")
	hostsPoolCmd.PersistentFlags().StringP("ca-file", "", "", "This provides a file path to a PEM-encoded certificate authority. This implies the use of HTTPS to connect to the Yorc REST API.")
	hostsPoolCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable coloring output")
	hostsPoolCmd.PersistentFlags().BoolP("secured", "s", false, "Use HTTPS to connect to the Yorc REST API")
	hostsPoolCmd.PersistentFlags().BoolP("skip-tls-verify", "", false, "skip-tls-verify controls whether a client verifies the server's certificate chain and host name. If set to true, TLS accepts any certificate presented by the server and any host name in that certificate. In this mode, TLS is susceptible to man-in-the-middle attacks. This should be used only for testing. This implies the use of HTTPS to connect to the Yorc REST API.")

	viper.BindPFlag("yorc_api", hostsPoolCmd.PersistentFlags().Lookup("yorc-api"))
	viper.BindPFlag("secured", hostsPoolCmd.PersistentFlags().Lookup("secured"))
	viper.BindPFlag("ca_file", hostsPoolCmd.PersistentFlags().Lookup("ca-file"))
	viper.BindPFlag("skip_tls_verify", hostsPoolCmd.PersistentFlags().Lookup("skip-tls-verify"))
	viper.SetEnvPrefix("yorc")
	viper.BindEnv("yorc_api", "YORC_API")
	viper.BindEnv("secured")
	viper.BindEnv("ca_file")
	viper.BindEnv("skip_tls_verify")
	viper.SetDefault("yorc_api", "localhost:8800")
	viper.SetDefault("secured", false)
	viper.SetDefault("skip_tls_verify", false)
}

// getColoredText returns a text colored accroding to the operation in
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

// AaddRow adds a row to a table, with text colored according to the operation
func addRow(table tabutil.Table, colorize bool, operation int,
	name string,
	connection hostspool.Connection,
	status *hostspool.HostStatus,
	message *string,
	labels map[string]string) {

	colNumber := 2
	statusString := ""
	if status != nil {
		colNumber++
		statusString = status.String()
	}
	messageString := ""
	if message != nil {
		colNumber++
		messageString = *message
	}
	if labels != nil {
		colNumber++
	}

	connectionSubRows := strings.Split(connection.String(), ",")
	var labelSubRows []string
	if labels != nil {
		labelSubRows = strings.Split(toPrintableLabels(labels), ",")
		sort.Strings(labelSubRows)
	}

	connectionSubRows, labelSubRows = padSlices(connectionSubRows, labelSubRows)
	subRowsNumber := len(connectionSubRows)

	// Add rows, one for each sub-column
	for i := 0; i < subRowsNumber; i++ {
		coloredColumns := make([]interface{}, colNumber)
		coloredColumns[0] = getColoredText(colorize, name, operation)
		coloredColumns[1] = getColoredText(colorize,
			strings.TrimSpace(connectionSubRows[i]), operation)
		j := 2
		if status != nil {
			// For a list operation, color the status according to its value
			if operation == hostList {
				coloredColumns[j] = getColoredHostStatus(colorize, statusString)
			} else {
				coloredColumns[j] = getColoredText(colorize, statusString, operation)
			}

			j++
		}
		if message != nil {
			coloredColumns[j] = getColoredText(colorize, messageString, operation)
			j++
		}
		if labels != nil {
			coloredColumns[j] = getColoredText(colorize,
				strings.TrimSpace(labelSubRows[i]), operation)
		}

		table.AddRow(coloredColumns...)
		if i == 0 {
			// Don't repeat single column values in sub-columns
			name = ""
			statusString = ""
			messageString = ""
		}
	}
}
