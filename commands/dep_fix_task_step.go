package commands

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"log"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"strings"
)

func init() {
	tasksCmd.AddCommand(updateTaskStepCmd)
}

var updateTaskStepCmd = &cobra.Command{
	Use:   "fix <DeploymentId> <TaskId> <StepName>",
	Short: "Fix a deployment task step in error",
	Long: `Fix a task step specifying the deployment id, the task id and the step name.
	The task step must be in error to be fixed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 3 {
			return errors.Errorf("Expecting a deployment id, a task id and a step name(got %d parameters)", len(args))
		}
		client, err := getClient()
		if err != nil {
			errExit(err)
		}

		// The task step status is set to "done"
		step := &tasks.TaskStep{Name: args[2], Status: strings.ToLower(tasks.TaskStepStatusDONE.String())}
		body, err := json.Marshal(step)
		if err != nil {
			log.Panic(err)
		}

		url := "/deployments/" + args[0] + "/tasks/" + args[1] + "/steps/" + args[2]
		request, err := client.NewRequest("PUT", url, bytes.NewBuffer(body))
		if err != nil {
			errExit(err)
		}

		request.Header.Add("Accept", "application/json")
		response, err := client.Do(request)
		if err != nil {
			errExit(err)
		}
		ids := args[0] + "/" + args[1]
		handleHTTPStatusCode(response, ids, "deployment/task/step", http.StatusOK)
		return nil
	},
}
