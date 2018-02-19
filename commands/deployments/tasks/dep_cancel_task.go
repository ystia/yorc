package tasks

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"novaforge.bull.com/starlings-janus/janus/commands/httputil"
)

func init() {
	tasksCmd.AddCommand(cancelTaskCmd)
}

var cancelTaskCmd = &cobra.Command{
	Use:   "cancel <DeploymentId> <TaskId>",
	Short: "Cancel a deployment task",
	Long: `Cancel a task specifying the deployment id and the task id.
	The task should be in status "INITIAL" or "RUNNING" to be canceled.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.Errorf("Expecting a deployment id and a task id (got %d parameters)", len(args))
		}
		client, err := httputil.GetClient()
		if err != nil {
			httputil.ErrExit(err)
		}

		url := "/deployments/" + args[0] + "/tasks/" + args[1]
		request, err := client.NewRequest("DELETE", url, nil)
		if err != nil {
			httputil.ErrExit(err)
		}

		request.Header.Add("Accept", "application/json")
		response, err := client.Do(request)
		defer response.Body.Close()
		if err != nil {
			httputil.ErrExit(err)
		}
		ids := args[0] + "/" + args[1]
		httputil.HandleHTTPStatusCode(response, ids, "deployment/task", http.StatusAccepted)
		return nil
	},
}
