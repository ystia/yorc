package workflows

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ystia/yorc/rest"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tmc/dot"
	"github.com/ystia/yorc/commands/httputil"
)

func init() {
	var workflowName string
	var horizontal bool
	var wfGraphCmd = &cobra.Command{
		Use:   "graph <id>",
		Long:  "Generate a GraphViz Dot format representation of a given workflow. The output can be easily converted to an image by making use of the dot command provided by GraphViz:\n\tyorc deployments workflows graph <id> | dot -Tpng > graph.png",
		Short: "Generate a GraphViz Dot format representation of a given workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting an id (got %d parameters)", len(args))
			}
			client, err := httputil.GetClient()
			if err != nil {
				httputil.ErrExit(err)
			}
			if workflowName == "" {
				return errors.New("Missing mandatory \"workflow-name\" parameter")
			}
			request, err := client.NewRequest("GET", fmt.Sprintf("/deployments/%s/workflows/%s", args[0], workflowName), nil)
			if err != nil {
				httputil.ErrExit(err)
			}
			request.Header.Add("Accept", "application/json")
			response, err := client.Do(request)
			defer response.Body.Close()
			if err != nil {
				httputil.ErrExit(err)
			}
			ids := args[0] + "/" + workflowName
			httputil.HandleHTTPStatusCode(response, ids, "deployment/workflow", http.StatusOK)

			var wf rest.Workflow
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				httputil.ErrExit(err)
			}
			err = json.Unmarshal(body, &wf)
			if err != nil {
				httputil.ErrExit(err)
			}

			graph := dot.NewGraph("Workflow " + workflowName)
			graph.SetType(dot.DIGRAPH)
			graph.Set("label", workflowName)
			graph.Set("labelloc", "t")
			if horizontal {
				graph.Set("rankdir", "LR")
			}
			nodes := make(map[string]*dot.Node, len(wf.Steps))
			for stepName, step := range wf.Steps {
				if _, ok := nodes[stepName]; !ok {
					nodes[stepName] = dot.NewNode(stepName)
				}
				stepNode := nodes[stepName]
				graph.AddNode(stepNode)

				for _, next := range step.OnSuccess {
					nextNode := dot.NewNode(next)
					nodes[next] = nextNode
					graph.AddEdge(dot.NewEdge(stepNode, nextNode))
				}
			}

			fmt.Println(graph)
			return nil
		},
	}
	wfGraphCmd.PersistentFlags().StringVarP(&workflowName, "workflow-name", "w", "", "The workflows name")
	wfGraphCmd.PersistentFlags().BoolVarP(&horizontal, "horizontal", "", false, "Draw graph with an horizontal layout.")
	workflowsCmd.AddCommand(wfGraphCmd)
}
