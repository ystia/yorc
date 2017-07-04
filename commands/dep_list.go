package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"novaforge.bull.com/starlings-janus/janus/helper/tabutil"
	"novaforge.bull.com/starlings-janus/janus/rest"
)

func init() {
	deploymentsCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List deployments",
	Long:  `List active deployments. Giving their id and status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		colorize := !noColor
		client, err := getClient()
		if err != nil {
			errExit(err)
		}
		request, err := client.NewRequest("GET", "/deployments", nil)
		if err != nil {
			errExit(err)
		}
		request.Header.Add("Accept", "application/json")
		response, err := client.Do(request)
		if err != nil {
			errExit(err)
		}
		handleHttpStatusCode(response)
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

				err = getJSONEntityFromAtomGetRequest(client, depLink, &dep)
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
