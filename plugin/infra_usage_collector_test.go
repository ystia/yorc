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
	"github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/prov"
	"testing"
	"time"
)

type mockInfraUsageCollector struct {
	getUsageInfoCalled bool
	ctx                context.Context
	conf               config.Configuration
	taskID             string
	infraName          string
	contextCancelled   bool
}

func (m *mockInfraUsageCollector) GetUsageInfo(ctx context.Context, conf config.Configuration, taskID, infraName string) (map[string]interface{}, error) {
	m.getUsageInfoCalled = true
	m.ctx = ctx
	m.conf = conf
	m.taskID = taskID
	m.infraName = infraName

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

func TestInfraUsageCollectorGetUsageInfo(t *testing.T) {
	t.Parallel()
	mock := new(mockInfraUsageCollector)
	client, _ := plugin.TestPluginRPCConn(t, map[string]plugin.Plugin{
		InfraUsageCollectorPluginName: &InfraUsageCollectorPlugin{F: func() prov.InfraUsageCollector {
			return mock
		}},
	})
	defer client.Close()

	raw, err := client.Dispense(InfraUsageCollectorPluginName)
	require.Nil(t, err)

	plugin := raw.(prov.InfraUsageCollector)

	info, err := plugin.GetUsageInfo(context.Background(), config.Configuration{ConsulAddress: "test", ConsulDatacenter: "testdc"}, "TestTaskID", "myInfra")
	require.Nil(t, err)
	require.True(t, mock.getUsageInfoCalled)
	require.Equal(t, "test", mock.conf.ConsulAddress)
	require.Equal(t, "testdc", mock.conf.ConsulDatacenter)
	require.Equal(t, "TestTaskID", mock.taskID)
	require.Equal(t, "myInfra", mock.infraName)
	require.Equal(t, 3, len(info))

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
	t.Parallel()
	mock := new(mockInfraUsageCollector)
	client, _ := plugin.TestPluginRPCConn(t, map[string]plugin.Plugin{
		InfraUsageCollectorPluginName: &InfraUsageCollectorPlugin{F: func() prov.InfraUsageCollector {
			return mock
		}},
	})
	defer client.Close()

	raw, err := client.Dispense(InfraUsageCollectorPluginName)
	require.Nil(t, err)

	plugin := raw.(prov.InfraUsageCollector)

	_, err = plugin.GetUsageInfo(context.Background(), config.Configuration{ConsulAddress: "test", ConsulDatacenter: "testdc"}, "TestFailure", "myInfra")
	require.Error(t, err, "An error was expected during executing plugin infra usage collector")
}

func TestInfraUsageCollectorGetUsageInfoWithCancel(t *testing.T) {
	t.Parallel()
	mock := new(mockInfraUsageCollector)
	client, _ := plugin.TestPluginRPCConn(t, map[string]plugin.Plugin{
		InfraUsageCollectorPluginName: &InfraUsageCollectorPlugin{F: func() prov.InfraUsageCollector {
			return mock
		}},
	})
	defer client.Close()

	raw, err := client.Dispense(InfraUsageCollectorPluginName)
	require.Nil(t, err)

	plugin := raw.(prov.InfraUsageCollector)

	ctx := context.Background()
	ctx, cancelF := context.WithCancel(ctx)
	go func() {
		_, err = plugin.GetUsageInfo(ctx, config.Configuration{ConsulAddress: "test", ConsulDatacenter: "testdc"}, "TestCancel", "myInfra")
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
	client, _ := plugin.TestPluginRPCConn(t, map[string]plugin.Plugin{
		InfraUsageCollectorPluginName: &InfraUsageCollectorPlugin{
			F: func() prov.InfraUsageCollector {
				return mock
			},
			SupportedInfras: []string{"myInfra"},
		},
	})
	defer client.Close()

	raw, err := client.Dispense(InfraUsageCollectorPluginName)
	require.Nil(t, err)

	plugin := raw.(InfraUsageCollector)

	infras, err := plugin.GetSupportedInfras()
	require.Nil(t, err)
	require.Equal(t, 1, len(infras))
	require.Equal(t, "myInfra", infras[0])
}
