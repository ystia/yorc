package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/helper/tabutil"
	"novaforge.bull.com/starlings-janus/janus/rest"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"os"
)

func init() {
	var withSteps bool
	var infoTaskCmd = &cobra.Command{
		Use:   "info <DeploymentId> <TaskId>",
		Short: "Get information about a deployment task",
		Long:  `Display information about a given task specifying the deployment id and the task id.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return errors.Errorf("Expecting a deployment id and a task id (got %d parameters)", len(args))
			}
			client, err := getClient()
			if err != nil {
				errExit(err)
			}

			url := "/deployments/" + args[0] + "/tasks/" + args[1]
			request, err := client.NewRequest("GET", url, nil)
			if err != nil {
				errExit(err)
			}

			request.Header.Add("Accept", "application/json")
			response, err := client.Do(request)
			if err != nil {
				errExit(err)
			}
			ids := args[0] + "/" + args[1]
			handleHTTPStatusCode(response, ids, "deployment/task", http.StatusOK)
			var task rest.Task
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				errExit(err)
			}
			err = json.Unmarshal(body, &task)
			if err != nil {
				errExit(err)
			}
			fmt.Println("Task: ", task.ID)
			fmt.Println("Task status:", task.Status)
			fmt.Println("Task type:", task.Type)

			if withSteps {
				displayStepTables(client, args)
			}

			return nil
		},
	}

	infoTaskCmd.PersistentFlags().BoolVarP(&withSteps, "steps", "w", false, "Show steps of the related workflow associated to the task")
	tasksCmd.AddCommand(infoTaskCmd)
}

func displayStepTables(client *janusClient, args []string) {
	colorize := !noColor
	if colorize {
		commErrorMsg = color.New(color.FgHiRed, color.Bold).SprintFunc()(commErrorMsg)
	}
	request, err := client.NewRequest("GET", "/deployments/"+args[0]+"/tasks/"+args[1]+"/steps", nil)
	if err != nil {
		errExit(err)
	}
	request.Header.Add("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		errExit(err)
	}
	handleHTTPStatusCode(response, args[0], "step", http.StatusOK)
	var steps []tasks.TaskStep
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		errExit(err)
	}
	err = json.Unmarshal(body, &steps)
	if err != nil {
		errExit(err)
	}
	if colorize {
		defer color.Unset()
	}
	fmt.Println("Steps:")
	tasksTable := tabutil.NewTable()
	tasksTable.AddHeaders("Name", "Status")
	errs := make([]error, 0)
	for _, step := range steps {
		tasksTable.AddRow(step.Name, getColoredTaskStatus(colorize, step.Status))
	}
	fmt.Println(tasksTable.Render())
	if len(errs) > 0 {
		fmt.Fprintln(os.Stderr, "\n\nErrors encountered:")
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, "###################\n", err)
		}
	}
}
