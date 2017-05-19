package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"novaforge.bull.com/starlings-janus/janus/helper/tabutil"
	"novaforge.bull.com/starlings-janus/janus/rest"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func init() {
	deploymentsCmd.AddCommand(tasksCmd)
}

var tasksCmd = &cobra.Command{
	Use:   "tasks <DeploymentId>",
	Short: "List tasks of a deployment",
	Long: `Display infos about the tasks related to a given deployment.
        It prints the tasks ID, type and status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("Expecting a deployment id (got %d parameters)", len(args))
		}
		client, err := getClient()
		if err != nil {
			errExit(err)
		}
		colorize := !noColor
		if colorize {
			commErrorMsg = color.New(color.FgHiRed, color.Bold).SprintFunc()(commErrorMsg)
		}
		request, err := client.NewRequest("GET", "/deployments/"+args[0], nil)
		if err != nil {
			errExit(err)
		}
		request.Header.Add("Accept", "application/json")
		response, err := client.Do(request)
		if err != nil {
			errExit(err)
		}
		if response.StatusCode != 200 {
			// Try to get the reason
			printErrors(response.Body)
			errExit(fmt.Errorf("Expecting HTTP Status code 200 got %d, reason %q", response.StatusCode, response.Status))
		}
		var dep rest.Deployment
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			errExit(err)
		}
		err = json.Unmarshal(body, &dep)
		if err != nil {
			errExit(err)
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
				err = getJSONEntityFromAtomGetRequest(client, atomLink, &task)
				if err != nil {
					errs = append(errs, err)
					tasksTable.AddRow(path.Base(atomLink.Href), "", commErrorMsg)
					continue
				}
				tasksTable.AddRow(task.ID, task.Type, getColoredTaskStatus(colorize, task.Status))
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
