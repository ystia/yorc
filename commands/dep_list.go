package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"novaforge.bull.com/starlings-janus/janus/helper/tabutil"
	"novaforge.bull.com/starlings-janus/janus/rest"
)

func init() {
	deploymentsCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List active deployments",
	Long:  `List active deployments. Giving there ids and statuses.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		janusAPI := viper.GetString("janus_api")
		colorize := !noColor

		request, err := http.NewRequest("GET", "http://"+janusAPI+"/deployments", nil)
		if err != nil {
			errExit(err)
		}
		request.Header.Add("Accept", "application/json")
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			errExit(err)
		}
		if response.StatusCode != 200 {
			// Try to get the reason
			printErrors(response.Body)
			errExit(fmt.Errorf("Expecting HTTP Status code 200 got %d, reason %q", response.StatusCode, response.Status))
		}
		var deps rest.DeploymentsCollection
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			errExit(err)
		}
		err = json.Unmarshal(body, &deps)
		if err != nil {
			errExit(err)
		}

		depsTable := tabutil.NewTable()
		depsTable.AddHeaders("Id", "Status")
		for _, depLink := range deps.Deployments {
			if depLink.Rel == rest.LinkRelDeployment {
				var dep rest.Deployment

				err = getJSONEntityFromAtomGetRequest(janusAPI, depLink, &dep)
				if err != nil {
					errExit(err)
				}
				depsTable.AddRow(dep.ID, getColoredDeploymentStatus(colorize, dep.Status))
			}
		}
		if colorize {
			defer color.Unset()
		}
		fmt.Println("Deployments:")
		fmt.Println(depsTable.Render())
		return nil
	},
}
