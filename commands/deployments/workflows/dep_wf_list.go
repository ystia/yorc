package workflows

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"novaforge.bull.com/starlings-janus/janus/rest"

	"path"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"novaforge.bull.com/starlings-janus/janus/commands/httputil"
)

func init() {
	var wfListCmd = &cobra.Command{
		Use:     "list <id>",
		Short:   "List workflows defined on a given deployment <id>",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting an id (got %d parameters)", len(args))
			}
			client, err := httputil.GetClient()
			if err != nil {
				httputil.ErrExit(err)
			}

			request, err := client.NewRequest("GET", fmt.Sprintf("/deployments/%s/workflows", args[0]), nil)
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

			var wfs rest.WorkflowsCollection
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				httputil.ErrExit(err)
			}
			err = json.Unmarshal(body, &wfs)
			if err != nil {
				httputil.ErrExit(err)
			}

			for _, wfLink := range wfs.Workflows {
				if wfLink.Rel == rest.LinkRelWorkflow {
					fmt.Println(path.Base(wfLink.Href))
				}
			}
			return nil
		},
	}
	workflowsCmd.AddCommand(wfListCmd)
}
