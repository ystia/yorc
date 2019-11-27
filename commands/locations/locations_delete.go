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
)

func init() {
	LocationsCmd.AddCommand(deleteCmd)
}

var deleteCmd = &cobra.Command{
	Use:   "delete <locationName>",
	Short: "Delete a location",
	Long:  `Delete a location.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := httputil.GetClient(ClientConfig)
		if err != nil {
			httputil.ErrExit(err)
		}
		return deleteLocation(client, args)
	},
}

func deleteLocation(client httputil.HTTPClient, args []string) error {
	if len(args) != 1 {
		return errors.Errorf("Expecting one location name (got %d parameters)", len(args))
	}
	err := deleteLocationConfig(client, args[0])
	if err != nil {
		return err
	}

	fmt.Println("Location " + args[0] + " deleted")

	return nil
}
