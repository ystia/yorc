package tasks

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"novaforge.bull.com/starlings-janus/janus/commands/httputil"
	"novaforge.bull.com/starlings-janus/janus/tasks"
)

func init() {
	tasksCmd.AddCommand(updateTaskStepCmd)
}

var updateTaskStepCmd = &cobra.Command{
	Use:   "fix <DeploymentId> <TaskId> <StepName>",
	Short: "Fix a deployment task step on error",
	Long: `Fix a task step specifying the deployment id, the task id and the step name.
	The task step must be on error to be fixed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 3 {
			return errors.Errorf("Expecting a deployment id, a task id and a step name(got %d parameters)", len(args))
		}
		client, err := httputil.GetClient()
		if err != nil {
			httputil.ErrExit(err)
		}

		// The task step status is set to "done"
		step := &tasks.TaskStep{Name: args[2], Status: strings.ToLower(tasks.TaskStepStatusDONE.String())}
		body, err := json.Marshal(step)
		if err != nil {
			log.Panic(err)
		}

		url := path.Join("/deployments", args[0], "tasks", args[1], "steps", args[2])
		request, err := client.NewRequest("PUT", url, bytes.NewBuffer(body))
		if err != nil {
			httputil.ErrExit(err)
		}

		request.Header.Add("Content-Type", "application/json")
		response, err := client.Do(request)
		defer response.Body.Close()
		if err != nil {
			httputil.ErrExit(err)
		}
		ids := args[0] + "/" + args[1]
		httputil.HandleHTTPStatusCode(response, ids, "deployment/task/step", http.StatusOK)
		return nil
	},
}
