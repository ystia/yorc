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

package server

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	ctu "github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/registry"
	"github.com/ystia/yorc/v4/vault"
)

type mockVaultSecret struct{}

func (mvs *mockVaultSecret) Raw() interface{} {
	return "mock"
}

func (mvs *mockVaultSecret) String() string {
	return fmt.Sprint(mvs.Raw())
}

type mockVaultClient struct{}

func (mvc *mockVaultClient) GetSecret(id string, options ...string) (vault.Secret, error) {
	return &mockVaultSecret{}, nil
}

func (mvc *mockVaultClient) Shutdown() error {
	return nil
}

type mockVaultClientBuilder struct{}

func (mvcb *mockVaultClientBuilder) BuildClient(cfg config.Configuration) (vault.Client, error) {
	return &mockVaultClient{}, nil
}

func Test_initVaultClient(t *testing.T) {

	registry.GetRegistry().RegisterVaultClientBuilder("my_mock_vault", &mockVaultClientBuilder{}, registry.BuiltinOrigin)

	type args struct {
		configuration config.Configuration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"NoVaultConfig", args{config.Configuration{}}, false},
		{"NoVaultProvider", args{config.Configuration{Vault: config.DynamicMap{"type": "some_vault_provider"}}}, true},
		{"MockVaultProvider", args{config.Configuration{Vault: config.DynamicMap{"type": "my_mock_vault"}}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := initVaultClient(tt.args.configuration)
			if (err != nil) != tt.wantErr {
				t.Errorf("initVaultClient() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.args.configuration.Vault.GetString("type") == "my_mock_vault" {
				assert.NotNil(t, deployments.DefaultVaultClient)
				assert.IsType(t, &mockVaultClient{}, deployments.DefaultVaultClient)
				secret, err := deployments.DefaultVaultClient.GetSecret("something")
				require.NoError(t, err)
				assert.Equal(t, "mock", secret.String())
			}
		})
	}
}

func testInitConsulClient(t *testing.T, srv *ctu.TestServer, client *api.Client) {
	type args struct {
		configuration config.Configuration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"normalsetup", args{config.Configuration{Consul: config.Consul{Address: srv.HTTPAddr}}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := initConsulClient(tt.args.configuration)
			if (err != nil) != tt.wantErr {
				t.Errorf("initConsulClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				assert.NotNil(t, got)
			}
		})
	}
}

func testInitLocationManager(t *testing.T, srv *ctu.TestServer) {
	consulConf := config.Consul{Address: srv.HTTPAddr}
	type args struct {
		configuration config.Configuration
	}
	tests := []struct {
		name          string
		args          args
		expectedlocNb int
		wantErr       bool
	}{
		{"TestNoLocations", args{config.Configuration{Consul: consulConf}}, 0, false},
		{"TestFirstTimeLocationsConfig", args{config.Configuration{Consul: consulConf, LocationsFilePath: "./testdata/locations1.yaml"}}, 1, false},
		{"TestSecondTimeLocationsConfigIgnored", args{config.Configuration{Consul: consulConf, LocationsFilePath: "./testdata/locations2.yaml"}}, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := initLocationManager(tt.args.configuration)
			if (err != nil) != tt.wantErr {
				t.Errorf("initLocationManager() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				m, err := locations.GetManager(tt.args.configuration)
				require.NoError(t, err)
				locationsConfigs, err := m.GetLocations()
				require.NoError(t, err)
				assert.Equal(t, tt.expectedlocNb, len(locationsConfigs))
			}
		})
	}
}

func testRunServer(t *testing.T, srv *ctu.TestServer, client *api.Client) {
	consulConf := config.Consul{Address: srv.HTTPAddr}
	type args struct {
		configuration config.Configuration
		shutdownCh    chan struct{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"NormalStartup", args{configuration: config.Configuration{Consul: consulConf, ServerGracefulShutdownTimeout: 1 * time.Millisecond}, shutdownCh: make(chan struct{})}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			go func() {
				select {
				case <-time.After(500 * time.Millisecond):
					close(tt.args.shutdownCh)
				}
			}()
			if err := RunServer(tt.args.configuration, tt.args.shutdownCh); (err != nil) != tt.wantErr {
				t.Errorf("RunServer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
