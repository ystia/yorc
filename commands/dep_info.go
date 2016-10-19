package commands

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/helper/tabutil"
	"novaforge.bull.com/starlings-janus/janus/rest"
	"strings"
)

func init() {
	deploymentsCmd.AddCommand(infoCmd)
}

var infoCmd = &cobra.Command{
	Use:   "info <DeploymentId>",
	Short: "Get Information about a deployment",
	Long: `Display information about a given deployment.
It prints the deployment status and the status of all the nodes contained in this deployment.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("Expecting a deployment id (got %d parameters)", len(args))
		}
		janusApi := viper.GetString("janus_api")
		colorize := !noColor

		request, err := http.NewRequest("GET", "http://"+janusApi+"/deployments/"+args[0], nil)
		if err != nil {
			errExit(err)
		}
		request.Header.Add("Accept", "application/json")
		response, err := http.DefaultClient.Do(request)
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
		fmt.Println("Deployment: ", dep.Id)

		fmt.Println("Global status:", getColoredDeploymentStatus(colorize, dep.Status))
		if colorize {
			defer color.Unset()
		}
		nodesTable := tabutil.NewTable()
		nodesTable.AddHeaders("Node", "Status", "Instances number")

		tasksTable := tabutil.NewTable()
		tasksTable.AddHeaders("Id", "Type", "Status")

		outputsTable := tabutil.NewTable()
		outputsTable.AddHeaders("Output Name", "Value")
		for _, atomLink := range dep.Links {

			if atomLink.Rel == rest.LINK_REL_NODE {
				response = doAtomGetRequest(janusApi, atomLink)
				var node rest.Node
				body, err := ioutil.ReadAll(response.Body)
				if err != nil {
					errExit(err)
				}
				err = json.Unmarshal(body, &node)
				if err != nil {
					errExit(err)
				}
				nbInstances := 0
				for _, nodeLink := range node.Links {
					if nodeLink.Rel == rest.LINK_REL_INSTANCE {
						nbInstances++
					}
				}
				if nbInstances == 0 {
					nbInstances = 1
				}
				nodesTable.AddRow(node.Name, getColoredNodeStatus(colorize, node.Status), nbInstances)
			} else if atomLink.Rel == rest.LINK_REL_TASK {
				response = doAtomGetRequest(janusApi, atomLink)
				var task rest.Task
				body, err := ioutil.ReadAll(response.Body)
				if err != nil {
					errExit(err)
				}
				err = json.Unmarshal(body, &task)
				if err != nil {
					errExit(err)
				}
				tasksTable.AddRow(task.Id, task.Type, getColoredTaskStatus(colorize, task.Status))
			} else if atomLink.Rel == rest.LINK_REL_OUTPUT {
				response = doAtomGetRequest(janusApi, atomLink)
				var output rest.Output
				body, err := ioutil.ReadAll(response.Body)
				if err != nil {
					errExit(err)
				}
				err = json.Unmarshal(body, &output)
				if err != nil {
					errExit(err)
				}
				outputsTable.AddRow(output.Name, output.Value)
			}

		}
		fmt.Println()
		fmt.Println("Nodes:")
		fmt.Println(nodesTable.Render())
		fmt.Println()
		fmt.Println("Tasks:")
		fmt.Println(tasksTable.Render())
		fmt.Println()
		fmt.Println("Outputs:")
		fmt.Println(outputsTable.Render())
		return nil
	},
}

func doAtomGetRequest(janusApi string, atomLink rest.AtomLink) *http.Response {
	request, err := http.NewRequest("GET", "http://"+janusApi+atomLink.Href, nil)
	if err != nil {
		errExit(err)
	}
	request.Header.Add("Accept", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		errExit(err)
	}
	if response.StatusCode != 200 {
		// Try to get the reason
		printErrors(response.Body)
		errExit(fmt.Errorf("Expecting HTTP Status code 200 got %d, reason %q", response.StatusCode, response.Status))
	}
	return response
}

func getColoredDeploymentStatus(colorize bool, status string) string {
	if !colorize {
		return status
	}
	switch {
	case strings.Contains(strings.ToLower(status), "failed"):
		return color.New(color.FgHiRed, color.Bold).SprintFunc()(status)
	case strings.HasSuffix(strings.ToLower(status), "ed"):
		return color.New(color.FgHiGreen, color.Bold).SprintFunc()(status)
	case strings.HasSuffix(strings.ToLower(status), "progress"):
		return color.New(color.FgHiYellow, color.Bold).SprintFunc()(status)
	default:
		return color.New(color.Bold).SprintFunc()(status)
	}
}

func getColoredNodeStatus(colorize bool, status string) string {
	if !colorize {
		return status
	}
	switch {
	case strings.Contains(strings.ToLower(status), "failed"):
		return color.New(color.FgHiRed, color.Bold).SprintFunc()(status)
	case strings.HasSuffix(strings.ToLower(status), "started"):
		return color.New(color.FgHiGreen, color.Bold).SprintFunc()(status)
	case strings.Contains(strings.ToLower(status), "canceled"):
		return color.New(color.FgHiWhite, color.Bold).SprintFunc()(status)
	default:
		return color.New(color.FgHiYellow, color.Bold).SprintFunc()(status)
	}
}

func getColoredTaskStatus(colorize bool, status string) string {
	if !colorize {
		return status
	}
	switch {
	case strings.ToLower(status) == "failed", strings.ToLower(status) == "canceled":
		return color.New(color.FgHiRed, color.Bold).SprintFunc()(status)
	case strings.ToLower(status) == "done":
		return color.New(color.FgHiGreen, color.Bold).SprintFunc()(status)
	case strings.ToLower(status) == "initial":
		return color.New(color.Bold).SprintFunc()(status)
	default:
		return color.New(color.FgHiYellow, color.Bold).SprintFunc()(status)
	}
}
