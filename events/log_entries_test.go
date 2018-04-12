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

package events

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/testutil"
)

func TestGenerateValue(t *testing.T) {
	t.Parallel()
	logEntry := WithOptionalFields(LogOptionalFields{
		WorkFlowID:    "my_workflowID",
		OperationName: "my_operationID",
	}).NewLogEntry(DEBUG, "my_deploymentID")

	logEntry.timestamp = time.Now()

	value, _ := logEntry.generateValue()
	require.Equal(t, "{\"content\":\"\",\"deploymentId\":\"my_deploymentID\",\"level\":\"DEBUG\",\"operationName\":\"my_operationID\",\"timestamp\":\"\",\"workflowId\":\"my_workflowID\"}", getLogEntryExceptTimestamp(t, string(value)))
}

func TestGenerateKey(t *testing.T) {
	t.Parallel()
	logEntry := WithOptionalFields(LogOptionalFields{
		WorkFlowID:    "my_workflowID",
		OperationName: "my_operationID",
	}).NewLogEntry(DEBUG, "my_deploymentID")

	logEntry.timestamp = time.Now()

	value := logEntry.generateKey()
	require.Equal(t, "_yorc/logs/my_deploymentID/"+logEntry.timestamp.Format(time.RFC3339Nano), string(value))
}

func TestSimpleLogEntry(t *testing.T) {
	t.Parallel()
	logEntry := SimpleLogEntry(INFO, "my_deploymentID")
	require.Equal(t, &LogEntry{level: INFO, deploymentID: "my_deploymentID"}, logEntry)
}

func testRegisterLogsInConsul(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	tests := []struct {
		name      string
		args      *LogEntry
		wantErr   bool
		wantValue string
	}{
		{name: "TestLevelDebug", args: &LogEntry{deploymentID: deploymentID, level: DEBUG, content: []byte("LOG ONE"), additionalInfo: nil}, wantErr: false, wantValue: "{\"content\":\"LOG ONE\",\"deploymentId\":\"TestRunConsulEventsPackageTests_groupEvents_TestRegisterLogsInConsul\",\"level\":\"DEBUG\",\"timestamp\":\"\"}"},
		{name: "TestLevelDebug", args: &LogEntry{deploymentID: deploymentID, level: INFO, content: []byte("LOG TWO"), additionalInfo: nil}, wantErr: false, wantValue: "{\"content\":\"LOG TWO\",\"deploymentId\":\"TestRunConsulEventsPackageTests_groupEvents_TestRegisterLogsInConsul\",\"level\":\"INFO\",\"timestamp\":\"\"}"},
		{name: "TestLevelDebug", args: &LogEntry{deploymentID: deploymentID, level: ERROR, content: []byte("LOG THREE"), additionalInfo: nil}, wantErr: false, wantValue: "{\"content\":\"LOG THREE\",\"deploymentId\":\"TestRunConsulEventsPackageTests_groupEvents_TestRegisterLogsInConsul\",\"level\":\"ERROR\",\"timestamp\":\"\"}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.Register(tt.args.content)
		})
	}

	logsPrefix := path.Join(consulutil.LogsPrefix, deploymentID)
	kvps, _, err := kv.List(logsPrefix, nil)
	assert.Nil(t, err)
	assert.Len(t, kvps, len(tests))

	for index, kvp := range kvps {
		tc := tests[index]
		assert.Equal(t, tc.wantValue, getLogEntryExceptTimestamp(t, string(kvp.Value)))
	}
}

// Do not handle timestamp value in assertions
func getLogEntryExceptTimestamp(t *testing.T, log string) string {
	var data map[string]string
	err := json.Unmarshal([]byte(log), &data)
	require.Nil(t, err)

	// Set timestamp/id to ""
	data["timestamp"] = ""
	b, err := json.Marshal(data)
	require.Nil(t, err)
	return string(b)
}

// Check sorting by timestamp preserves the order of log entries stored in Consul
func testLogsSortedByTimestamp(t *testing.T, kv *api.KV) {
	t.Parallel()

	// Register log entries in Consul
	const NumberOfLogs = 100
	const ContentFormat = "Log id %d"
	deploymentID := testutil.BuildDeploymentID(t)
	logEntry := SimpleLogEntry(INFO, deploymentID)
	for i := 0; i < NumberOfLogs; i++ {
		logEntry.Registerf(ContentFormat, i)
	}

	// Retrieve key/value pairs stored in Consul
	logEntries := make([]map[string]string, NumberOfLogs)
	logsPrefix := path.Join(consulutil.LogsPrefix, deploymentID)
	kvps, _, err := kv.List(logsPrefix, nil)
	require.NoError(t, err, "Failure getting log entries from consul")
	require.Len(t, kvps, NumberOfLogs, "Got unexpected number of log entries from Consul")
	for i, kvp := range kvps {
		err := json.Unmarshal(kvp.Value, &logEntries[i])
		require.NoError(t, err, "Failure unmarshalling value from consul")
	}

	// Sort entries by timestamp
	sort.Slice(logEntries, func(i, j int) bool {
		firstTime, err := time.Parse(time.RFC3339Nano, logEntries[i]["timestamp"])
		require.NoError(t, err, "Failure parsing timestamp %s", logEntries[i]["timestamp"])
		secondTime, err := time.Parse(time.RFC3339Nano, logEntries[j]["timestamp"])
		require.NoError(t, err, "Failure parsing timestamp %s", logEntries[j]["timestamp"])
		return firstTime.Before(secondTime)
	})

	// Check log entries were sorted in the order they were inserted
	for i := 0; i < NumberOfLogs; i++ {
		value := -1
		fmt.Sscanf(logEntries[i]["content"], ContentFormat, &value)
		require.NoError(t, err, "Failure scanning integer in string %s", logEntries[i]["content"])
		assert.Equal(t, i, value, "Unexpected log entry at index %d: %s", i, logEntries[i]["content"])
	}

}

func TestContext(t *testing.T) {
	rootLogOpts := LogOptionalFields{WorkFlowID: "wf"}
	require.Len(t, rootLogOpts, 1)
	rootCtx := NewContext(context.Background(), rootLogOpts)
	require.NotNil(t, rootCtx)

	lof1, ok := FromContext(rootCtx)
	assert.True(t, ok)
	assert.NotNil(t, lof1)

	lof1[NodeID] = "node"
	assert.Len(t, lof1, 2)
	assert.Len(t, rootLogOpts, 1)

	lof2, ok := FromContext(rootCtx)
	assert.True(t, ok)
	assert.NotNil(t, lof2)

	lof2[InstanceID] = "1"
	assert.Len(t, lof2, 2)
	assert.Len(t, rootLogOpts, 1)

}
