package events

import (
	"testing"
	"github.com/stretchr/testify/require"
	"github.com/hashicorp/consul/api"
	"path"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"github.com/stretchr/testify/assert"
	"novaforge.bull.com/starlings-janus/janus/testutil"
)

func TestGenerateValue(t *testing.T) {
	t.Parallel()
	value := WithOptionalFields(OptionalFields{
		WorkFlowID:  "my_workflowID",
		OperationID: "my_operationID",
	}).Add(DEBUG, "my_deploymentID").generateValue()
	require.Equal(t,"{\"OperationID\":\"my_operationID\",\"WorkFlowID\":\"my_workflowID\",\"content\":null,\"deploymentID\":\"my_deploymentID\",\"level\":\"DEBUG\"}", string(value))
}

func TestNewLogEntry(t *testing.T) {
	t.Parallel()
	logEntry, err := NewLogEntry(TRACE, "my_deploymentID")
	require.Nil(t, err)
	require.Equal(t, &FormattedLogEntry{level: TRACE, deploymentID: "my_deploymentID"}, logEntry)
}

func testRegisterLogsInConsul(t *testing.T, kv *api.KV){
	t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	tests := []struct{
		name string
		args *FormattedLogEntry
		wantErr bool
		wantValue string
	}{
		{name: "TestLevelDebug", args:&FormattedLogEntry{deploymentID:deploymentID, level:DEBUG, content:[]byte("LOG ONE"), additionalInfo:nil}, wantErr: false, wantValue:"{\"content\":\"LOG ONE\",\"deploymentID\":\"TestRunConsulEventsPackageTests_groupEvents_TestRegisterLogsInConsul\",\"level\":\"DEBUG\"}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.Register(tt.args.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("TestRegister() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
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
