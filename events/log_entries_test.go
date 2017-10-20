package events

import (
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/testutil"
	"path"
	"testing"
)

type mockTimeProvider struct{}

func TestGenerateValue(t *testing.T) {
	t.Parallel()
	t.Skip()
	getTimestamp = func() string {
		return "FAKE"
	}

	value := WithOptionalFields(LogOptionalFields{
		WorkFlowID:  "my_workflowID",
		OperationID: "my_operationID",
	}).NewLogEntry(DEBUG, "my_deploymentID").generateValue()
	require.Equal(t, "{\"Content\":\"\",\"DeploymentID\":\"my_deploymentID\",\"Level\":\"DEBUG\",\"OperationID\":\"my_operationID\",\"Timestamp\":\"FAKE\",\"WorkFlowID\":\"my_workflowID\"}", string(value))
}

func TestSimpleLogEntry(t *testing.T) {
	t.Parallel()
	logEntry := SimpleLogEntry(TRACE, "my_deploymentID")
	require.Equal(t, &FormattedLogEntry{level: TRACE, deploymentID: "my_deploymentID"}, logEntry)
}

func testRegisterLogsInConsul(t *testing.T, kv *api.KV) {
	t.Parallel()
	getTimestamp = func() string {
		return "FAKE"
	}
	deploymentID := testutil.BuildDeploymentID(t)
	tests := []struct {
		name      string
		args      *FormattedLogEntry
		wantErr   bool
		wantValue string
	}{
		{name: "TestLevelDebug", args: &FormattedLogEntry{deploymentID: deploymentID, level: DEBUG, content: []byte("LOG ONE"), additionalInfo: nil}, wantErr: false, wantValue: "{\"Content\":\"LOG ONE\",\"DeploymentID\":\"TestRunConsulEventsPackageTests_groupEvents_TestRegisterLogsInConsul\",\"Level\":\"DEBUG\",\"Timestamp\":\"FAKE\"}"},
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
		assert.Equal(t, tc.wantValue, string(kvp.Value))
	}
}
