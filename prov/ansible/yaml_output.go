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

package ansible

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/log"
)

// Regular expression for a line like:
// PLAY [playbook name] ********************************************************
var newSectionRegExp = regexp.MustCompile("(PLAY|TASK) \\[([^\\]]*)\\] [\\*]+")

// Regular expression for a line like:
// ok: [10.0.0.1]
var taskStatusRegExp = regexp.MustCompile("(ok|changed|skipped|fatal|failed): \\[([^\\]]*)\\]")

type ansibleTaskContext struct {
	inTaskSection             bool
	inTaskDetailsSection      bool
	playSection               string
	taskName                  string
	taskSectionHeader         string
	taskSectionStatusMap      map[string]string
	taskLogLevelMap           map[string]events.LogLevel
	taskStdErrLogLevelMap     map[string]events.LogLevel
	taskSectionDetailsMap     map[string]string
	currentHost               string
	taskSectionDetailsBuilder strings.Builder
	context                   context.Context
	deploymentID              string
	nodeName                  string
	hostsConn                 map[string]*hostConnection
	playbookMode              bool
}

func getInstanceIDForHost(host string, hostConnectionMap map[string]*hostConnection) string {
	instanceID := ""
	for _, connection := range hostConnectionMap {
		if host == connection.host {
			instanceID = connection.instanceID
			break
		}
	}
	return instanceID
}

// logAnsibleTaskResult logs the result of an Ansible task
func logAnsibleTaskResult(ansibleTask *ansibleTaskContext) {
	for host, status := range ansibleTask.taskSectionStatusMap {

		var logBuilder strings.Builder
		logBuilder.WriteString("Ansible task output:\n")
		if ansibleTask.playSection != "" {
			logBuilder.WriteString(ansibleTask.playSection)
			logBuilder.WriteString(" ")
		}
		logBuilder.WriteString(ansibleTask.taskSectionHeader)
		fmt.Fprintf(&logBuilder, "\n%s\n%s", status, ansibleTask.taskSectionDetailsMap[host])

		// Add the instance ID to event context
		instanceID := host
		instanceCtx := ansibleTask.context
		instanceID = getInstanceIDForHost(host, ansibleTask.hostsConn)
		if instanceID != "" {
			instanceCtx = events.AddLogOptionalFields(ansibleTask.context,
				events.LogOptionalFields{events.InstanceID: instanceID})
		}

		events.WithContextOptionalFields(instanceCtx).
			NewLogEntry(ansibleTask.taskLogLevelMap[host], ansibleTask.deploymentID).
			RegisterAsString(logBuilder.String())
	}
}

// logScriptTaskResult logs the result of a script, filtering the fact that the
// Orchestrator is using Ansible interbally to run this script
func logScriptTaskResult(ansibleTask *ansibleTaskContext) {
	for host := range ansibleTask.taskSectionStatusMap {

		// Add the instance ID to event context
		instanceID := host
		instanceCtx := ansibleTask.context
		instanceID = getInstanceIDForHost(host, ansibleTask.hostsConn)
		if instanceID != "" {
			instanceCtx = events.AddLogOptionalFields(ansibleTask.context,
				events.LogOptionalFields{events.InstanceID: instanceID})
		}

		var taskDetailsMap map[string]interface{}
		err := yaml.Unmarshal([]byte(ansibleTask.taskSectionDetailsMap[host]), &taskDetailsMap)
		if err != nil {
			log.Fatal("Could not unmarshall task details", err)
			return
		}

		// List of attributes to extract from a task yaml output and log
		ansibleAttributes := []struct {
			name     string
			isStderr bool
		}{
			{name: "module_stderr", isStderr: true},
			{name: "module_stdout", isStderr: false},
			{name: "stderr", isStderr: true},
			{name: "stdout", isStderr: false},
			{name: "msg", isStderr: false},
		}
		for _, ansibleAttr := range ansibleAttributes {

			attrValue, found := taskDetailsMap[ansibleAttr.name]
			if found && attrValue != "" {

				var logBuilder strings.Builder
				fmt.Fprintf(&logBuilder, "Node %q, host %q, %s:\n%s",
					ansibleTask.nodeName, host, ansibleAttr.name, attrValue)

				var logLevel events.LogLevel
				if ansibleAttr.isStderr {
					logLevel = ansibleTask.taskStdErrLogLevelMap[host]
				} else {
					logLevel = ansibleTask.taskLogLevelMap[host]
				}
				events.WithContextOptionalFields(instanceCtx).
					NewLogEntry(logLevel, ansibleTask.deploymentID).
					RegisterAsString(logBuilder.String())
			}
		}
	}
}

func endTaskSectionAndLogTaskResult(ansibleTask *ansibleTaskContext) {
	if ansibleTask.inTaskSection {
		// End of previous task section
		if ansibleTask.inTaskDetailsSection {
			ansibleTask.inTaskDetailsSection = false
			ansibleTask.taskSectionDetailsMap[ansibleTask.currentHost] = ansibleTask.taskSectionDetailsBuilder.String()
			ansibleTask.taskSectionDetailsBuilder.Reset()
		}

		if ansibleTask.playbookMode {
			logAnsibleTaskResult(ansibleTask)
		} else {
			// The user did not specify a playbook but a script to run.
			// The orchestrator is using a playbook internally but this
			// internal implementation doesn't have to appear in Consul logs.
			// Logs for this script will be added in Consul.
			// Logs of other internal tasks are skipped.
			if ansibleTask.taskName == "command" || ansibleTask.taskName == "shell" {
				logScriptTaskResult(ansibleTask)
			}
		}
	}
}

// logAnsibleOutput logs the Ansible yaml output of a playbook.
// If playbookMode, the raw yaml output of tasks is provided in logs.
// Else, only some yaml attributes (stdout, stderr) are added in logs,
// this is used when running scripts to not let appear in logs that the
// Orchestrator encapsulated this script in an Ansible playbook to run it
func logAnsibleOutput(ctx context.Context, deploymentID, nodeName string,
	hostsConn map[string]*hostConnection,
	output io.ReadCloser,
	playbookMode bool) {

	ansibleTask := ansibleTaskContext{
		taskSectionStatusMap:  make(map[string]string),
		taskLogLevelMap:       make(map[string]events.LogLevel),
		taskStdErrLogLevelMap: make(map[string]events.LogLevel),
		taskSectionDetailsMap: make(map[string]string),
		context:               ctx,
		deploymentID:          deploymentID,
		nodeName:              nodeName,
		hostsConn:             hostsConn,
		playbookMode:          playbookMode,
	}

	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := scanner.Text()
		log.Debugf("%s", line)

		// Check if it is a new section (play or task) starting
		newSectionMatch := newSectionRegExp.FindStringSubmatch(line)
		if len(newSectionMatch) > 2 {
			// End of previous task section if any
			endTaskSectionAndLogTaskResult(&ansibleTask)

			ansibleTask.inTaskSection = (newSectionMatch[1] == "TASK")
			ansibleTask.taskSectionHeader = ""
			ansibleTask.taskSectionStatusMap = make(map[string]string)
			ansibleTask.taskSectionDetailsMap = make(map[string]string)
			ansibleTask.taskSectionDetailsBuilder.Reset()
			if ansibleTask.inTaskSection {
				ansibleTask.taskName = newSectionMatch[2]
				ansibleTask.taskSectionHeader = fmt.Sprintf("%s [%s]", newSectionMatch[1], newSectionMatch[2])
			} else if newSectionMatch[1] == "PLAY" {
				ansibleTask.playSection = fmt.Sprintf("%s [%s]", newSectionMatch[1], newSectionMatch[2])
			}
			continue
		}

		// Check if this line provides a task status
		taskStatusMatch := taskStatusRegExp.FindStringSubmatch(line)
		if len(taskStatusMatch) > 1 {

			// End of previous section providing details of the task execution
			// on another host
			if ansibleTask.inTaskDetailsSection {
				ansibleTask.inTaskDetailsSection = false
				ansibleTask.taskSectionDetailsMap[ansibleTask.currentHost] = ansibleTask.taskSectionDetailsBuilder.String()
				ansibleTask.taskSectionDetailsBuilder.Reset()
			}

			ansibleTask.currentHost = taskStatusMatch[2]
			ansibleTask.taskSectionStatusMap[ansibleTask.currentHost] = line
			if taskStatusMatch[1] == "failed" || taskStatusMatch[1] == "fatal" {
				ansibleTask.taskLogLevelMap[ansibleTask.currentHost] = events.LogLevelERROR
				ansibleTask.taskStdErrLogLevelMap[ansibleTask.currentHost] = events.LogLevelERROR
			} else {
				ansibleTask.taskLogLevelMap[ansibleTask.currentHost] = events.LogLevelINFO
				// If the task didn't fail, log stderr messages as warnings
				ansibleTask.taskStdErrLogLevelMap[ansibleTask.currentHost] = events.LogLevelWARN
			}
			continue
		}

		// Check if this line provides details of a task run
		if ansibleTask.inTaskSection && strings.HasPrefix(line, "  ") {
			ansibleTask.inTaskDetailsSection = true
			fmt.Fprintf(&ansibleTask.taskSectionDetailsBuilder, "%s\n", line)
			continue
		}

		// check the playbook end
		if strings.HasPrefix(line, "PLAY RECAP *") {
			// End last task section
			endTaskSectionAndLogTaskResult(&ansibleTask)
			break
		}
	} // end of loop on playbook output
}

type logAnsibleOutputInConsulFn func(context.Context, string, string, map[string]hostConnection, io.ReadCloser)

func logAnsibleOutputInConsulFromScript(ctx context.Context, deploymentID, nodeName string, hostsConn map[string]*hostConnection, output io.ReadCloser) {
	playbookMode := false
	logAnsibleOutput(ctx, deploymentID, nodeName, hostsConn, output, playbookMode)
}

func logAnsibleOutputInConsul(ctx context.Context, deploymentID, nodeName string, hostsConn map[string]*hostConnection, output io.ReadCloser) {
	playbookMode := true
	logAnsibleOutput(ctx, deploymentID, nodeName, hostsConn, output, playbookMode)
}
