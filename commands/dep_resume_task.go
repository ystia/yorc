package commands

import (
	"net/http"
	"path"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	tasksCmd.AddCommand(resumeTaskCmd)
}

var resumeTaskCmd = &cobra.Command{
	Use:   "resume <DeploymentId> <TaskId>",
	Short: "Resume a deployment task",
	Long: `Resume a task specifying the deployment id and the task id.
	The task should be in status "FAILED" to be resumed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.Errorf("Expecting a deployment id and a task id (got %d parameters)", len(args))
		}
		client, err := getClient()
		if err != nil {
			errExit(err)
		}

		url := path.Join("/deployments", args[0], "tasks", args[1])
		request, err := client.NewRequest("PUT", url, nil)
		if err != nil {
			errExit(err)
		}

		//request.Header.Add("Accept", "application/json")
		response, err := client.Do(request)
		defer response.Body.Close()
		if err != nil {
			errExit(err)
		}
		ids := args[0] + "/" + args[1]
		handleHTTPStatusCode(response, ids, "deployment/task", http.StatusAccepted)
		return nil
	},
}
