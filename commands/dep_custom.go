package commands

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net/http"
)

func init() {
	var jsonParam string
	var customCmd = &cobra.Command{
		Use:   "custom <id>",
		Short: "Use a custom command",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("Expecting an id (got %d parameters)", len(args))
			}
			janusApi := viper.GetString("janus_api")

			if len(jsonParam) == 0 {
				return fmt.Errorf("You need to provide a JSON in parameter")
			}

			request, err := http.NewRequest("POST", "http://"+janusApi+"/deployments/"+args[0]+"/custom", bytes.NewBuffer([]byte(jsonParam)))
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

			fmt.Println("Command submited. path :", response.Header.Get("Location"))
			return nil
		},
	}
	customCmd.PersistentFlags().StringVarP(&jsonParam, "data", "d", "", "Need to provide the JSON format of the custom command")
	deploymentsCmd.AddCommand(customCmd)
}
