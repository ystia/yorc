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
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ystia/yorc/commands/httputil"
	"github.com/ystia/yorc/rest"
)

func init() {
	var applyCmd = &cobra.Command{
		Use:   "apply <path to Hosts Pool description>",
		Short: "Apply a Hosts Pool description",
		Long: `Apply a Hosts Pool description provided in the file passed in argument
	This file should contain a YAML description.`,
		RunE: func(cmd *cobra.Command, args []string) error {
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

			// Read config file, viper will accept indiferrently a yaml or json
			// format
			viper.SetConfigFile(args[0])
			err = viper.ReadInConfig()
			if err != nil {
				log.Panic(err)
			}

			var hostsPoolRequest rest.HostsPoolRequest
			err = viper.Unmarshal(&hostsPoolRequest)
			if err != nil {
				log.Panic(err)
			}
			log.Println("LOLO", hostsPoolRequest)
			client, err := httputil.GetClient()
			if err != nil {
				httputil.ErrExit(err)
			}

			bArray, err := json.Marshal(&hostsPoolRequest)
			request, err := client.NewRequest("PUT", "/hosts_pool",
				bytes.NewBuffer(bArray))
			if err != nil {
				httputil.ErrExit(err)
			}
			request.Header.Add("Content-Type", "application/json")

			response, err := client.Do(request)
			defer response.Body.Close()
			if err != nil {
				httputil.ErrExit(err)
			}

			httputil.HandleHTTPStatusCode(response, args[0], "host pool", http.StatusOK)
			fmt.Println("Command submitted.")
			return nil
		},
	}
	hostsPoolCmd.AddCommand(applyCmd)
}
