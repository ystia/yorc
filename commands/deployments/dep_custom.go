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

package deployments

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/ystia/yorc/rest"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/ystia/yorc/commands/httputil"
	"github.com/ystia/yorc/tosca"
)

func init() {
	var jsonParam string
	var nodeName string
	var customCName string
	var interfaceName string
	var inputs []string
	var customCmd = &cobra.Command{
		Use:   "custom <id>",
		Short: "Execute a custom command",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting an id (got %d parameters)", len(args))
			}
			client, err := httputil.GetClient(ClientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}
			if len(jsonParam) == 0 && (len(nodeName) == 0 || len(interfaceName) == 0 || len(customCName) == 0) {
				return errors.Errorf("You need to provide a JSON or complete the mandatory flags (\"custom\", \"node\", \"interface\")")
			}

			if len(jsonParam) == 0 && len(nodeName) != 0 && len(customCName) != 0 {
				var InputsStruct rest.CustomCommandRequest
				InputsStruct.CustomCommandName = customCName
				InputsStruct.NodeName = nodeName
				InputsStruct.InterfaceName = interfaceName
				InputsStruct.Inputs = make(map[string]*tosca.ValueAssignment)
				for _, arg := range inputs {
					keyValue := strings.Split(arg, "=")
					var value tosca.ValueAssignment
					err := json.Unmarshal([]byte(strings.TrimSpace(keyValue[1])), &value)
					if err != nil {
						return err
					}
					InputsStruct.Inputs[strings.TrimSpace(keyValue[0])] = &value
				}

				tmp, err := json.Marshal(InputsStruct)
				if err != nil {
					log.Panic(err)
				}

				jsonParam = string(tmp)
			}

			request, err := client.NewRequest("POST", "/deployments/"+args[0]+"/custom", bytes.NewBuffer([]byte(jsonParam)))
			if err != nil {
				httputil.ErrExit(err)
			}
			request.Header.Add("Content-Type", "application/json")

			response, err := client.Do(request)
			if err != nil {
				httputil.ErrExit(err)
			}
			defer response.Body.Close()

			httputil.HandleHTTPStatusCode(response, args[0], "deployment", http.StatusAccepted)
			fmt.Println("Command submitted. path :", response.Header.Get("Location"))
			return nil
		},
	}
	customCmd.PersistentFlags().StringVarP(&jsonParam, "data", "d", "", "Provide the JSON format of the custom command with node, interface, custom and inputs data")
	customCmd.PersistentFlags().StringVarP(&nodeName, "node", "n", "", "Provide the node name (mandatory)")
	customCmd.PersistentFlags().StringVarP(&interfaceName, "interface", "", "", "Provide the interface name (mandatory)")
	customCmd.PersistentFlags().StringVarP(&customCName, "custom", "", "", "Provide the custom command name (mandatory)")
	customCmd.PersistentFlags().StringArrayVarP(&inputs, "input", "i", make([]string, 0), "Provide the input for the custom command")
	DeploymentsCmd.AddCommand(customCmd)
}
