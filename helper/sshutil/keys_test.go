// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package sshutil

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v3/deployments"
)

var expectedKey []byte

func init() {
	var err error
	expectedKey, err = ioutil.ReadFile("./testdata/validkey.pem")
	if err != nil {
		panic(err)
	}
}

func testGetKeysFromCredentialsAttribute(t *testing.T, kv *api.KV) {
	deploymentID := loadTestYaml(t, kv, "ComputeWithCredentials.yaml")

	err := deployments.SetInstanceStateStringWithContextualLogs(context.Background(), kv, deploymentID, "Compute", "0", "started")
	require.NoError(t, err, "Failed to setup instance state")
	err = deployments.SetInstanceStateStringWithContextualLogs(context.Background(), kv, deploymentID, "ComputeMultiKeys", "0", "started")
	require.NoError(t, err, "Failed to setup instance state")
	type args struct {
		nodeName       string
		instanceID     string
		capabilityName string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]*PrivateKey
		wantErr bool
	}{
		{"GetLocalFile", args{"Compute", "0", "endpoint"}, map[string]*PrivateKey{"0": &PrivateKey{Content: expectedKey, Path: "./testdata/validkey.pem"}}, false},
		{"GetMultiKeys", args{"ComputeMultiKeys", "0", "endpoint"}, map[string]*PrivateKey{"0": &PrivateKey{Content: expectedKey, Path: "./testdata/validkey.pem"}, "anotherValidKey": &PrivateKey{Content: expectedKey}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetKeysFromCredentialsAttribute(kv, deploymentID, tt.args.nodeName, tt.args.instanceID, tt.args.capabilityName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetKeysFromCredentialsAttribute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.EqualValuesf(t, tt.want, got, "GetKeysFromCredentialsAttribute() = %v, want %v", got, tt.want)
		})
	}
}
