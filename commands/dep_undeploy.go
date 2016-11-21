package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net/http"
)

func init() {
	deploymentsCmd.AddCommand(undeployCmd)
	setConfigUndeploy()
}

var undeployCmd = &cobra.Command{
	Use:   "undeploy <DeploymentId>",
	Short: "Undeploy an application",
	Long:  `Undeploy an application specifying the deployment ID.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("Expecting a deployment id (got %d parameters)", len(args))
		}
		janusApi := viper.GetString("janus_api")

		url :=  "http://"+janusApi+"/deployments/"+args[0]
		if getConfigUndeploy() {
			url = url+"?purge=true"
		}

		request, err := http.NewRequest("DELETE",url, nil)
		if err != nil {
			errExit(err)
		}

		request.Header.Add("Accept", "application/json")
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			errExit(err)
		}
		if response.StatusCode != 202 {
			// Try to get the reason
			printErrors(response.Body)
			errExit(fmt.Errorf("Expecting HTTP Status code 202 got %d, reason %q", response.StatusCode, response.Status))
		}

		fmt.Println("Undeployment submited. In progress...")
		return nil
	},
}

func setConfigUndeploy() {
	//Flags definition for purge
	undeployCmd.PersistentFlags().BoolP("purge", "p", false, "To use if you want to purge instead of undeploy")

}

func getConfigUndeploy() bool {
	//Flags for purge
	return viper.GetBool("purge")
}
