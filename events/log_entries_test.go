package events

import (
	"encoding/json"
	"path"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/testutil"
)

type mockTimeProvider struct{}

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
	require.Equal(t, "_janus/logs/my_deploymentID/"+logEntry.timestamp.Format(time.RFC3339Nano), string(value))
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

	logsPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "logs")
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
