package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"path"

	"os"

	"bytes"
	"strconv"

	"net/http"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"novaforge.bull.com/starlings-janus/janus/helper/tabutil"
	"novaforge.bull.com/starlings-janus/janus/rest"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

var commErrorMsg = "Janus API communication error"

func init() {
	var detailedInfo bool

	var infoCmd = &cobra.Command{
		Use:   "info <DeploymentId>",
		Short: "Get Information about a deployment",
		Long: `Display information about a given deployment.
It prints the deployment status and the status of all the nodes contained in this deployment.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting a deployment id (got %d parameters)", len(args))
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
			defer response.Body.Close()
			if err != nil {
				errExit(err)
			}
			handleHTTPStatusCode(response, args[0], "deployment", http.StatusOK)
			var dep rest.Deployment
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				errExit(err)
			}
			err = json.Unmarshal(body, &dep)
			if err != nil {
				errExit(err)
			}
			fmt.Println("Deployment: ", dep.ID)

			fmt.Println("Global status:", getColoredDeploymentStatus(colorize, dep.Status))
			if colorize {
				defer color.Unset()
			}
			var errs []error
			if !detailedInfo {
				errs = tableBasedDeploymentRendering(client, dep, colorize)
			} else {
				errs = detailedDeploymentRendering(client, dep, colorize)
			}
			if len(errs) > 0 {
				fmt.Fprintln(os.Stderr, "\n\nErrors encountered:")
				for _, err := range errs {
					fmt.Fprintln(os.Stderr, "###################\n", err)
				}
			}
			return nil
		},
	}
	infoCmd.PersistentFlags().BoolVarP(&detailedInfo, "detailed", "d", false, "Add details to the info command making it less concise and readable.")

	deploymentsCmd.AddCommand(infoCmd)
}

func tableBasedDeploymentRendering(client *janusClient, dep rest.Deployment, colorize bool) []error {
	errs := make([]error, 0)
	nodesTable := tabutil.NewTable()
	nodesTable.AddHeaders("Node", "Statuses")

	tasksTable := tabutil.NewTable()
	tasksTable.AddHeaders("Id", "Type", "Status")

	outputsTable := tabutil.NewTable()
	outputsTable.AddHeaders("Output Name", "Value")
	var err error
	for _, atomLink := range dep.Links {

		if atomLink.Rel == rest.LinkRelNode {
			var node rest.Node

			err = getJSONEntityFromAtomGetRequest(client, atomLink, &node)
			if err != nil {
				errs = append(errs, err)
				nodesTable.AddRow(path.Base(atomLink.Href), commErrorMsg, "")
				continue
			}
			statusesMap := make(map[string]int)
			totalNbInstances := 0
			for _, nodeLink := range node.Links {
				if nodeLink.Rel == rest.LinkRelInstance {
					var instance rest.NodeInstance
					err = getJSONEntityFromAtomGetRequest(client, nodeLink, &instance)
					if err != nil {
						errs = append(errs, err)
						nodesTable.AddRow(path.Base(atomLink.Href), commErrorMsg, "")
						continue
					}
					statusesMap[instance.Status]++
					totalNbInstances++
				}
			}
			var buffer bytes.Buffer
			for status, nbInstances := range statusesMap {
				buffer.WriteString(getColoredNodeStatus(colorize, status))
				buffer.WriteString(" (")
				buffer.WriteString(strconv.Itoa(nbInstances))
				buffer.WriteString("/")
				buffer.WriteString(strconv.Itoa(totalNbInstances))
				buffer.WriteString(") ")
			}
			nodesTable.AddRow(node.Name, buffer.String())
		} else if atomLink.Rel == rest.LinkRelTask {
			var task rest.Task
			err = getJSONEntityFromAtomGetRequest(client, atomLink, &task)
			if err != nil {
				errs = append(errs, err)
				tasksTable.AddRow(path.Base(atomLink.Href), "", commErrorMsg)
				continue
			}
			tasksTable.AddRow(task.ID, task.Type, getColoredTaskStatus(colorize, task.Status))
		} else if atomLink.Rel == rest.LinkRelOutput {
			var output rest.Output

			err = getJSONEntityFromAtomGetRequest(client, atomLink, &output)
			if err != nil {
				errs = append(errs, err)
				outputsTable.AddRow(path.Base(atomLink.Href), commErrorMsg)
				continue
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
	return errs
}

func detailedDeploymentRendering(client *janusClient, dep rest.Deployment, colorize bool) []error {
	errs := make([]error, 0)
	var err error
	nodesList := []string{"Nodes:"}
	tasksList := []string{"Tasks:"}
	outputsList := []string{"Outputs:"}
	for _, atomLink := range dep.Links {

		if atomLink.Rel == rest.LinkRelNode {
			var node rest.Node

			err = getJSONEntityFromAtomGetRequest(client, atomLink, &node)
			if err != nil {
				errs = append(errs, err)
				nodesList = append(nodesList, fmt.Sprintf("  - %s: %s", path.Base(atomLink.Href), commErrorMsg))
				continue
			}
			nodesList = append(nodesList, fmt.Sprintf("  - %s:", node.Name))
			nodesList = append(nodesList, fmt.Sprintf("    Instances:"))
			for _, nodeLink := range node.Links {
				if nodeLink.Rel == rest.LinkRelInstance {
					var inst rest.NodeInstance
					err = getJSONEntityFromAtomGetRequest(client, nodeLink, &inst)
					if err != nil {
						errs = append(errs, err)
						nodesList = append(nodesList, fmt.Sprintf("      - %s: %s", path.Base(nodeLink.Href), commErrorMsg))
						continue
					}
					nodesList = append(nodesList, fmt.Sprintf("      - %s: %s", inst.ID, getColoredNodeStatus(colorize, inst.Status)))
					nodesList = append(nodesList, fmt.Sprintf("        Attributes:"))
					for _, instanceLink := range inst.Links {
						if instanceLink.Rel == rest.LinkRelAttribute {
							var attr rest.Attribute
							err = getJSONEntityFromAtomGetRequest(client, instanceLink, &attr)
							if err != nil {
								errs = append(errs, err)
								nodesList = append(nodesList, fmt.Sprintf("          - %s: %s", path.Base(instanceLink.Href), commErrorMsg))
								continue
							}
							nodesList = append(nodesList, fmt.Sprintf("          - %s: %s", attr.Name, attr.Value))
						}
					}
				}
			}
		} else if atomLink.Rel == rest.LinkRelTask {
			var task rest.Task
			err = getJSONEntityFromAtomGetRequest(client, atomLink, &task)
			if err != nil {
				errs = append(errs, err)
				tasksList = append(tasksList, fmt.Sprintf("  - %s: %s", path.Base(atomLink.Href), commErrorMsg))
				continue
			}
			tasksList = append(tasksList, fmt.Sprintf("  - %s:", task.ID))
			tasksList = append(tasksList, fmt.Sprintf("      type: %s", task.Type))
			tasksList = append(tasksList, fmt.Sprintf("      status: %s", getColoredTaskStatus(colorize, task.Status)))
		} else if atomLink.Rel == rest.LinkRelOutput {
			var output rest.Output

			err = getJSONEntityFromAtomGetRequest(client, atomLink, &output)
			if err != nil {
				errs = append(errs, err)
				outputsList = append(outputsList, fmt.Sprintf("  - %s: %s", path.Base(atomLink.Href), commErrorMsg))
				continue
			}
			outputsList = append(outputsList, fmt.Sprintf("  - %s: %s", output.Name, output.Value))
		}
	}
	fmt.Println()
	for _, line := range nodesList {
		fmt.Println(line)
	}
	fmt.Println()
	for _, line := range tasksList {
		fmt.Println(line)
	}
	fmt.Println()
	for _, line := range outputsList {
		fmt.Println(line)
	}
	return errs
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
	case strings.Contains(strings.ToLower(status), tosca.NodeStateError.String()):
		return color.New(color.FgHiRed, color.Bold).SprintFunc()(status)
	case strings.HasSuffix(strings.ToLower(status), tosca.NodeStateStarted.String()):
		return color.New(color.FgHiGreen, color.Bold).SprintFunc()(status)
	case strings.Contains(strings.ToLower(status), tosca.NodeStateInitial.String()), strings.Contains(strings.ToLower(status), tosca.NodeStateDeleted.String()):
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
