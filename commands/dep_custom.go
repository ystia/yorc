package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"novaforge.bull.com/starlings-janus/janus/rest"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	var jsonParam string
	var nodeName string
	var customCName string
	var inputs []string
	var customCmd = &cobra.Command{
		Use:   "custom <id>",
		Short: "Use a custom command",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("Expecting an id (got %d parameters)", len(args))
			}
			janusAPI := viper.GetString("janus_api")

			if len(jsonParam) == 0 && len(nodeName) == 0 {
				return fmt.Errorf("You need to provide a JSON or complete the arguments")
			}

			if len(jsonParam) == 0 && len(nodeName) != 0 && len(customCName) != 0 {
				var InputsStruct rest.CustomCommandRequest
				InputsStruct.CustomCommandName = customCName
				InputsStruct.NodeName = nodeName

				for _, arg := range inputs {
					for _, split := range strings.Split(arg, ",") {
						tmp := strings.Split(split, "=")
						InputsStruct.Inputs[tmp[0]] = tmp[1]
					}
				}

				tmp, err := json.Marshal(InputsStruct)
				if err != nil {
					log.Panic(err)
				}

				jsonParam = string(tmp)
			}

			request, err := http.NewRequest("POST", "http://"+janusAPI+"/deployments/"+args[0]+"/custom", bytes.NewBuffer([]byte(jsonParam)))
			if err != nil {
				errExit(err)
			}
			request.Header.Add("Content-Type", "application/json")

			response, err := http.DefaultClient.Do(request)
			if err != nil {
				errExit(err)
			}
			if response.StatusCode != http.StatusAccepted {
				// Try to get the reason
				printErrors(response.Body)
				errExit(fmt.Errorf("Expecting HTTP Status code 202 got %d, reason %q", response.StatusCode, response.Status))
			}

			fmt.Println("Command submitted. path :", response.Header.Get("Location"))
			return nil
		},
	}
	customCmd.PersistentFlags().StringVarP(&jsonParam, "data", "d", "", "Need to provide the JSON format of the custom command")
	customCmd.PersistentFlags().StringVarP(&nodeName, "node", "n", "", "Provide the node name (use with flag c and i)")
	customCmd.PersistentFlags().StringVarP(&customCName, "custom", "c", "", "Provide the custom command name (use with flag n and i)")
	customCmd.PersistentFlags().StringSliceVarP(&inputs, "inputsMap", "i", make([]string, 0), "Provide the input for the custom command (use with flag c and n)")
	deploymentsCmd.AddCommand(customCmd)
}
