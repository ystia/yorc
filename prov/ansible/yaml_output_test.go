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
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/testutil"
)

func testLogAnsibleOutputInConsul(t *testing.T, kv *api.KV) {

	testFileName := "testdata/playbook_yaml_output.txt"
	file, err := os.Open(testFileName)
	require.NoError(t, err, "Unexpected error opening testdata file %s", testFileName)
	defer file.Close()

	log.SetDebug(true)
	deploymentID := testutil.BuildDeploymentID(t)
	nodeName := "node"

	hosts := map[string]*hostConnection{
		"Instance_0": {host: "10.0.0.132", instanceID: "0"},
		"Instance_1": {host: "10.0.0.133", instanceID: "1"},
	}

	logOptFields := events.LogOptionalFields{
		events.WorkFlowID:    "wfID",
		events.NodeID:        nodeName,
		events.OperationName: "create",
		events.InterfaceName: "standard",
	}
	ctx := events.NewContext(context.Background(), logOptFields)
	logAnsibleOutputInConsul(ctx, deploymentID, nodeName, hosts, file)

	logs, _, err := events.LogsEvents(kv, deploymentID, 0, 5*time.Second)
	require.NoError(t, err, "Could not retrieve logs")

	// Check the content of a task output INFO log on instance 1

	logFilters := map[string]string{
		"instanceId": "1",
		"level":      "INFO",
	}
	expectedLogContent := "Long startup component iteration 10"

	found := false
	var logMap map[string]string
	for _, logEntry := range logs {
		err := json.Unmarshal(logEntry, &logMap)
		require.NoError(t, err, "Error unmarshalling log entry %v", logEntry)
		next := false
		for k, v := range logFilters {
			if logMap[k] != v {
				next = true
				break
			}
		}
		if next {
			continue
		}

		found = strings.Contains(logMap["content"], expectedLogContent)
		if found {
			break
		}
	}

	assert.True(t, found, "Did not find expected task log output %q for instance 1 in\n%+v",
		expectedLogContent, logMap)

}

// Find a log in log entries according to filters and a content
func getLogMap(logs []json.RawMessage, logFilters map[string]string, content string) (map[string]string, error) {
	found := false
	var result map[string]string
	for _, logEntry := range logs {
		var logMap map[string]string
		if err := json.Unmarshal(logEntry, &logMap); err != nil {
			return nil, err
		}
		next := false
		for k, v := range logFilters {
			if logMap[k] != v {
				next = true
				break
			}
		}
		if next {
			continue
		}
		found = strings.Contains(logMap["content"], content)
		if found {
			result = logMap
			break
		}
	}
	return result, nil
}

func testLogAnsibleOutputInConsulFromScript(t *testing.T, kv *api.KV) {

	testFileName := "testdata/script_yaml_output.txt"
	file, err := os.Open(testFileName)
	require.NoError(t, err, "Unexpected error opening testdata file %s", testFileName)
	defer file.Close()

	log.SetDebug(true)
	deploymentID := testutil.BuildDeploymentID(t)
	nodeName := "node"

	hosts := map[string]*hostConnection{
		"Instance_0": {host: "10.0.0.132", instanceID: "0"},
		"Instance_1": {host: "10.0.0.133", instanceID: "1"},
	}

	logOptFields := events.LogOptionalFields{
		events.WorkFlowID:    "wfID",
		events.NodeID:        nodeName,
		events.OperationName: "create",
		events.InterfaceName: "standard",
	}
	ctx := events.NewContext(context.Background(), logOptFields)
	logAnsibleOutputInConsulFromScript(ctx, deploymentID, nodeName, hosts, file)

	logs, _, err := events.LogsEvents(kv, deploymentID, 0, 5*time.Second)
	require.NoError(t, err, "Could not retrieve logs")

	// Check the content of a task output INFO log on instance 0
	logFilters := map[string]string{
		"instanceId": "0",
		"level":      "INFO",
	}
	expectedLogContent := "Long startup component iteration 10"
	logMap, err := getLogMap(logs, logFilters, expectedLogContent)
	require.NoError(t, err, "Error unmarshalling log entries")
	assert.NotNil(t, logMap, "Did not find expected task log output %q for instance 0 in logs",
		expectedLogContent)

	// Check the content of a task output WARN log on instance 1
	// (an output on stderr of a task that was successful)

	logFilters = map[string]string{
		"instanceId": "1",
		"level":      "WARN",
	}

	expectedLogContent = "Writing to stderr line 2"
	logMap, err = getLogMap(logs, logFilters, expectedLogContent)
	require.NoError(t, err, "Error unmarshalling log entries")
	assert.NotNil(t, logMap, "Did not find expected task log output %q for instance 1 in logs",
		expectedLogContent)

}

func testLogAnsibleOutputInConsulFromScriptFailure(t *testing.T, kv *api.KV) {

	testFileName := "testdata/script_yaml_output_fatal.txt"
	file, err := os.Open(testFileName)
	require.NoError(t, err, "Unexpected error opening testdata file %s", testFileName)
	defer file.Close()

	log.SetDebug(true)
	deploymentID := testutil.BuildDeploymentID(t)
	nodeName := "node"

	hosts := map[string]*hostConnection{
		"Instance_0": {host: "192.168.2.11", instanceID: "0"},
	}

	logOptFields := events.LogOptionalFields{
		events.WorkFlowID:    "wfID",
		events.NodeID:        nodeName,
		events.OperationName: "create",
		events.InterfaceName: "standard",
	}
	ctx := events.NewContext(context.Background(), logOptFields)
	logAnsibleOutputInConsulFromScript(ctx, deploymentID, nodeName, hosts, file)

	logs, _, err := events.LogsEvents(kv, deploymentID, 0, 5*time.Second)
	require.NoError(t, err, "Could not retrieve logs")

	// Check the content of a task output ERROR log on instance 1
	// (an output on stdout or stderr of a task that failed)

	logFilters := map[string]string{
		"instanceId": "0",
		"level":      "ERROR",
	}
	expectedLogContent := "Short startup component iteration"
	logMap, err := getLogMap(logs, logFilters, expectedLogContent)
	require.NoError(t, err, "Error unmarshalling log entries")
	assert.NotNil(t, logMap, "Did not find expected task log output %q for instance 0 in logs",
		expectedLogContent)

	expectedLogContent = "Message on stderr"
	logMap, err = getLogMap(logs, logFilters, expectedLogContent)
	require.NoError(t, err, "Error unmarshalling log entries")
	assert.NotNil(t, logMap, "Did not find expected task log output %q for instance 0 in logs",
		expectedLogContent)

}
