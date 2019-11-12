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

package plugin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/prov"
)

type mockInfraUsageCollector struct {
	getUsageInfoCalled bool
	ctx                context.Context
	conf               config.Configuration
	taskID             string
	infraName          string
	locationName       string
	params             map[string]string
	contextCancelled   bool
	lof                events.LogOptionalFields
}

func (m *mockInfraUsageCollector) GetUsageInfo(ctx context.Context, conf config.Configuration, taskID, infraName, locationName string,
	params map[string]string) (map[string]interface{}, error) {
	m.getUsageInfoCalled = true
	m.ctx = ctx
	m.conf = conf
	m.taskID = taskID
	m.infraName = infraName
	m.locationName = locationName
	m.params = params
	m.lof, _ = events.FromContext(ctx)

	go func() {
		<-m.ctx.Done()
		m.contextCancelled = true
	}()
	if m.taskID == "TestCancel" {
		<-m.ctx.Done()
	}
	if m.taskID == "TestFailure" {
		return nil, NewRPCError(errors.New("a failure occurred during plugin infra usage collector"))
	}
	res := make(map[string]interface{})
	res["keyOne"] = "valueOne"
	res["keyTwo"] = "valueTwo"
	res["keyThree"] = "valueThree"
	return res, nil
}

func setupInfraUsageCollectorTestEnv(t *testing.T) (*mockInfraUsageCollector, *plugin.RPCClient,
	prov.InfraUsageCollector, events.LogOptionalFields, context.Context) {

	t.Parallel()
	mock := new(mockInfraUsageCollector)
	client, _ := plugin.TestPluginRPCConn(
		t,
		map[string]plugin.Plugin{
			InfraUsageCollectorPluginName: &InfraUsageCollectorPlugin{F: func() prov.InfraUsageCollector {
				return mock
			}},
		},
		nil)

	raw, err := client.Dispense(InfraUsageCollectorPluginName)
	require.Nil(t, err)

	plugin := raw.(prov.InfraUsageCollector)

	lof := events.LogOptionalFields{
		events.WorkFlowID:    "testWF",
		events.InterfaceName: "delegate",
		events.OperationName: "myTest",
	}
	ctx := events.NewContext(context.Background(), lof)

	return mock, client, plugin, lof, ctx

}
func TestInfraUsageCollectorGetUsageInfo(t *testing.T) {
	mock, client, plugin, lof, ctx := setupInfraUsageCollectorTestEnv(t)
	defer client.Close()
	params := map[string]string{"param1": "value1"}
	info, err := plugin.GetUsageInfo(
		ctx,
		config.Configuration{Consul: config.Consul{Address: "test", Datacenter: "testdc"}},
		"TestTaskID", "myInfra", "myLocation", params)
	require.Nil(t, err)
	require.True(t, mock.getUsageInfoCalled)
	require.Equal(t, "test", mock.conf.Consul.Address)
	require.Equal(t, "testdc", mock.conf.Consul.Datacenter)
	require.Equal(t, "TestTaskID", mock.taskID)
	require.Equal(t, "myInfra", mock.infraName)
	require.Equal(t, "myLocation", mock.locationName)
	require.Equal(t, params, mock.params)
	require.Equal(t, 3, len(info))
	assert.Equal(t, lof, mock.lof)

	val, exist := info["keyOne"]
	require.True(t, exist)
	require.Equal(t, "valueOne", val)

	val, exist = info["keyTwo"]
	require.True(t, exist)
	require.Equal(t, "valueTwo", val)

	val, exist = info["keyThree"]
	require.True(t, exist)
	require.Equal(t, "valueThree", val)
}

func TestInfraUsageCollectorGetUsageInfoWithFailure(t *testing.T) {
	_, client, plugin, _, ctx := setupInfraUsageCollectorTestEnv(t)
	defer client.Close()
	_, err := plugin.GetUsageInfo(
		ctx,
		config.Configuration{Consul: config.Consul{Address: "test", Datacenter: "testdc"}},
		"TestFailure", "myInfra", "myLocation", map[string]string{})
	require.Error(t, err, "An error was expected during executing plugin infra usage collector")
	require.EqualError(t, err, "a failure occurred during plugin infra usage collector")
}

func TestInfraUsageCollectorGetUsageInfoWithCancel(t *testing.T) {
	mock, client, plugin, _, ctx := setupInfraUsageCollectorTestEnv(t)
	defer client.Close()
	ctx, cancelF := context.WithCancel(ctx)
	go func() {
		_, err := plugin.GetUsageInfo(
			ctx,
			config.Configuration{Consul: config.Consul{Address: "test", Datacenter: "testdc"}},
			"TestCancel", "myInfra", "myLocation", map[string]string{})
		require.Nil(t, err)
	}()
	cancelF()
	// Wait for cancellation signal to be dispatched
	time.Sleep(50 * time.Millisecond)
	require.True(t, mock.contextCancelled, "Context should be cancelled")
}

func TestGetSupportedInfra(t *testing.T) {
	t.Parallel()
	mock := new(mockInfraUsageCollector)
	client, _ := plugin.TestPluginRPCConn(
		t,
		map[string]plugin.Plugin{
			InfraUsageCollectorPluginName: &InfraUsageCollectorPlugin{
				F: func() prov.InfraUsageCollector {
					return mock
				},
				SupportedInfras: []string{"myInfra"},
			},
		},
		nil)
	defer client.Close()

	raw, err := client.Dispense(InfraUsageCollectorPluginName)
	require.Nil(t, err)

	plugin := raw.(InfraUsageCollector)

	infras, err := plugin.GetSupportedInfras()
	require.Nil(t, err)
	require.Equal(t, 1, len(infras))
	require.Equal(t, "myInfra", infras[0])
}
