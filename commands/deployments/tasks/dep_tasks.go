package tasks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"novaforge.bull.com/starlings-janus/janus/helper/tabutil"
	"novaforge.bull.com/starlings-janus/janus/rest"

	"net/http"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"novaforge.bull.com/starlings-janus/janus/commands/deployments"
	"novaforge.bull.com/starlings-janus/janus/commands/httputil"
)

func init() {
	deployments.DeploymentsCmd.AddCommand(tasksCmd)
}

var commErrorMsg = httputil.JanusAPIDefaultErrorMsg
var tasksCmd = &cobra.Command{
	Use:   "tasks <DeploymentId>",
	Short: "List tasks of a deployment",
	Long: `Display info about the tasks related to a given deployment.
    It prints the tasks ID, type and status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.Errorf("Expecting a deployment id (got %d parameters)", len(args))
		}
		client, err := httputil.GetClient()
		if err != nil {
			httputil.ErrExit(err)
		}
		colorize := !deployments.NoColor
		if colorize {
			commErrorMsg = color.New(color.FgHiRed, color.Bold).SprintFunc()(commErrorMsg)
		}
		request, err := client.NewRequest("GET", "/deployments/"+args[0], nil)
		if err != nil {
			httputil.ErrExit(err)
		}
		request.Header.Add("Accept", "application/json")
		response, err := client.Do(request)
		defer response.Body.Close()
		if err != nil {
			httputil.ErrExit(err)
		}
		httputil.HandleHTTPStatusCode(response, args[0], "deployment", http.StatusOK)
		var dep rest.Deployment
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			httputil.ErrExit(err)
		}
		err = json.Unmarshal(body, &dep)
		if err != nil {
			httputil.ErrExit(err)
		}
		if colorize {
			defer color.Unset()
		}
		fmt.Println("Tasks:")
		tasksTable := tabutil.NewTable()
		tasksTable.AddHeaders("Id", "Type", "Status")
		errs := make([]error, 0)
		for _, atomLink := range dep.Links {
			if atomLink.Rel == rest.LinkRelTask {
				var task rest.Task
				err = httputil.GetJSONEntityFromAtomGetRequest(client, atomLink, &task)
				if err != nil {
					errs = append(errs, err)
					tasksTable.AddRow(path.Base(atomLink.Href), commErrorMsg)
					continue
				}
				tasksTable.AddRow(task.ID, task.Type, deployments.GetColoredTaskStatus(colorize, task.Status))
			}
		}
		fmt.Println(tasksTable.Render())
		if len(errs) > 0 {
			fmt.Fprintln(os.Stderr, "\n\nErrors encountered:")
			for _, err := range errs {
				fmt.Fprintln(os.Stderr, "###################\n", err)
			}
		}
		return nil
	},
}
