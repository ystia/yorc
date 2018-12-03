// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deployments

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/tasks"

	"path"

	"os"

	"bytes"
	"strconv"

	"net/http"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/ystia/yorc/commands/httputil"
	"github.com/ystia/yorc/helper/tabutil"
	"github.com/ystia/yorc/rest"
	"github.com/ystia/yorc/tosca"
)

var commErrorMsg = httputil.YorcAPIDefaultErrorMsg

func init() {
	var detailedInfo bool
	var follow bool

	var infoCmd = &cobra.Command{
		Use:   "info <DeploymentId>",
		Short: "Get Information about a deployment",
		Long: `Display information about a given deployment.
It prints the deployment status and the status of all the nodes contained in this deployment.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting a deployment id (got %d parameters)", len(args))
			}
			client, err := httputil.GetClient(ClientConfig)
			if err != nil {
				httputil.ErrExit(err)
			}

			err = DisplayInfo(client, args[0], detailedInfo, follow, 3*time.Second)
			if err != nil {
				httputil.ErrExit(err)
			}

			return err
		},
	}
	infoCmd.PersistentFlags().BoolVarP(&detailedInfo, "detailed", "d", false, "Add details to the info command making it less concise and readable.")
	infoCmd.PersistentFlags().BoolVarP(&follow, "follow", "f", false, "Follow deployment info updates (without details) until the deployment is finished.")

	DeploymentsCmd.AddCommand(infoCmd)
}

// DisplayInfo displays deployment info
func DisplayInfo(client *httputil.YorcClient, deploymentID string, detailed, follow bool, refreshTime time.Duration) error {

	colorize := !NoColor
	if colorize {
		commErrorMsg = color.New(color.FgHiRed, color.Bold).SprintFunc()(httputil.YorcAPIDefaultErrorMsg)
	}
	request, err := client.NewRequest("GET", "/deployments/"+deploymentID, nil)
	if err != nil {
		return err
	}
	finished := false
	lastStatus := ""
	for !finished {
		request.Header.Add("Accept", "application/json")
		response, err := client.Do(request)
		if err != nil {
			if lastStatus != "" {
				// Following a deployment that was purged, ending wihtout error
				return nil
			}
			return err
		}
		defer response.Body.Close()
		if follow && response.StatusCode == http.StatusNotFound {
			// Undeployment done, not exiting on error when
			// the CLI is folliwng the undeployment steps
			fmt.Printf("%s undeployed.\n", deploymentID)
			return nil
		}

		httputil.HandleHTTPStatusCode(response, deploymentID, "deployment", http.StatusOK)
		var dep rest.Deployment
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}
		err = json.Unmarshal(body, &dep)
		if err != nil {
			return err
		}
		if follow {
			// Set the cursor to row 0, column 0
			fmt.Printf("\033[0;0H")
			if lastStatus != dep.Status {
				// Clear screen to remove end of line when new status is shorter
				print("\033[H\033[2J")
				lastStatus = dep.Status
			}
		}
		fmt.Println("Deployment: ", dep.ID)

		fmt.Println("Global status:", getColoredDeploymentStatus(colorize, dep.Status))
		if colorize {
			defer color.Unset()
		}
		var errs []error
		if !detailed || follow {
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

		finished = !follow ||
			dep.Status == deployments.DEPLOYED.String() ||
			dep.Status == deployments.UNDEPLOYED.String() ||
			dep.Status == deployments.DEPLOYMENT_FAILED.String() ||
			dep.Status == deployments.UNDEPLOYMENT_FAILED.String()
		if !finished {
			time.Sleep(refreshTime)
		}

	}
	return nil
}
func tableBasedDeploymentRendering(client *httputil.YorcClient, dep rest.Deployment, colorize bool) []error {
	errs := make([]error, 0)
	nodesTable := tabutil.NewTable()
	nodesTable.AddHeaders("Node", "Status (instance/total)")

	tasksTable := tabutil.NewTable()
	tasksTable.AddHeaders("Id", "Type", "Status")

	outputsTable := tabutil.NewTable()
	outputsTable.AddHeaders("Output Name", "Value")
	var err error
	for _, atomLink := range dep.Links {

		if atomLink.Rel == rest.LinkRelNode {
			var node rest.Node

			err = httputil.GetJSONEntityFromAtomGetRequest(client, atomLink, &node)
			if err != nil {
				errs = append(errs, err)
				nodesTable.AddRow(path.Base(atomLink.Href), commErrorMsg)
				continue
			}
			statusesMap := make(map[string]int)
			totalNbInstances := 0
			for _, nodeLink := range node.Links {
				if nodeLink.Rel == rest.LinkRelInstance {
					var instance rest.NodeInstance
					err = httputil.GetJSONEntityFromAtomGetRequest(client, nodeLink, &instance)
					if err != nil {
						errs = append(errs, err)
						nodesTable.AddRow(path.Base(atomLink.Href), commErrorMsg)
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
			err = httputil.GetJSONEntityFromAtomGetRequest(client, atomLink, &task)
			if err != nil {
				errs = append(errs, err)
				tasksTable.AddRow(path.Base(atomLink.Href), commErrorMsg)
				continue
			}
			// Ignore TaskTypeAction
			if tasks.TaskTypeAction.String() != task.Type {
				tasksTable.AddRow(task.ID, task.Type, GetColoredTaskStatus(colorize, task.Status))
			}
		} else if atomLink.Rel == rest.LinkRelOutput {
			var output rest.Output

			err = httputil.GetJSONEntityFromAtomGetRequest(client, atomLink, &output)
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

func detailedDeploymentRendering(client *httputil.YorcClient, dep rest.Deployment, colorize bool) []error {
	errs := make([]error, 0)
	var err error
	nodesList := []string{"Nodes:"}
	tasksList := []string{"Tasks:"}
	outputsList := []string{"Outputs:"}
	for _, atomLink := range dep.Links {

		if atomLink.Rel == rest.LinkRelNode {
			var node rest.Node

			err = httputil.GetJSONEntityFromAtomGetRequest(client, atomLink, &node)
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
					err = httputil.GetJSONEntityFromAtomGetRequest(client, nodeLink, &inst)
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
							err = httputil.GetJSONEntityFromAtomGetRequest(client, instanceLink, &attr)
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
			err = httputil.GetJSONEntityFromAtomGetRequest(client, atomLink, &task)
			if err != nil {
				errs = append(errs, err)
				tasksList = append(tasksList, fmt.Sprintf("  - %s: %s", path.Base(atomLink.Href), commErrorMsg))
				continue
			}
			tasksList = append(tasksList, fmt.Sprintf("  - %s:", task.ID))
			tasksList = append(tasksList, fmt.Sprintf("      type: %s", task.Type))
			tasksList = append(tasksList, fmt.Sprintf("      status: %s", GetColoredTaskStatus(colorize, task.Status)))
		} else if atomLink.Rel == rest.LinkRelOutput {
			var output rest.Output

			err = httputil.GetJSONEntityFromAtomGetRequest(client, atomLink, &output)
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

// GetColoredTaskStatus allows to color status on CLI
func GetColoredTaskStatus(colorize bool, status string) string {
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
