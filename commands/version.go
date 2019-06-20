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

package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ystia/yorc/v4/helper/consulutil"
)

var version = "unknown phantom version"

var gitCommit = "dev version"

func init() {
	var quiet bool
	versionCmd := &cobra.Command{

		Use:   "version",
		Short: "Print the version",
		Long:  `The version of Yorc`,
		Run: func(cmd *cobra.Command, args []string) {
			if quiet {
				fmt.Println(version)
			} else {
				fmt.Println("Yorc Server", version)
				fmt.Printf("Database schema version: %s\n", consulutil.YorcSchemaVersion)
				fmt.Printf("Revision: %q\n", gitCommit)
			}

		},
	}
	versionCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, fmt.Sprintf("Print just the release number in machine readable format (ie: %s)", version))

	RootCmd.AddCommand(versionCmd)
	RootCmd.Version = version
}
