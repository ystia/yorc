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

package commons

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/helper/sshutil"
)

var expectedKey string

func init() {
	b, err := ioutil.ReadFile("./testdata/validkey.pem")
	if err != nil {
		panic(err)
	}
	expectedKey = string(b)
}

func testAddConnectionCheckResource(t *testing.T, kv *api.KV) {
	deploymentID := loadTestYaml(t, kv)
	infra := &Infrastructure{}
	env := &[]string{}

	ctx := context.Background()
	testPk := sshutil.PrivateKey{Content: []byte("secretkey")}

	err := AddConnectionCheckResource(ctx, kv, deploymentID, "Compute", infra, "user",
		&testPk, "10.0.0.1", "Compute", env)
	assert.Nil(t, err)
	check := requireRemoteExec(t, infra, "Compute")
	assert.Equal(t, "user", check.Connection.User)
	assert.Equal(t, "", check.Connection.BastionHost)
	assert.Equal(t, "${var.private_key}", check.Connection.PrivateKey, "should be set to variable passed by environment")
	assert.Contains(t, *env, "TF_VAR_private_key="+string(testPk.Content), "private key environment variable should be set")
	assert.NotNil(t, infra.Variable["private_key"])

	err = AddConnectionCheckResource(ctx, kv, deploymentID, "ComputeBastionPassword", infra, "user",
		&testPk, "10.0.0.1", "ComputeBastionPassword", env)
	assert.Nil(t, err)
	check = requireRemoteExec(t, infra, "ComputeBastionPassword")
	assert.Equal(t, "10.0.0.2", check.Connection.BastionHost)
	assert.Equal(t, "22", check.Connection.BastionPort)
	assert.Equal(t, "ubuntu", check.Connection.BastionUser)
	assert.Equal(t, "secret", check.Connection.BastionPassword)
	assert.Equal(t, "${var.bastion_private_key}", check.Connection.BastionPrivateKey, "should be set to variable passed by environment")
	assert.Contains(t, *env, "TF_VAR_bastion_private_key="+string(testPk.Content))
	assert.NotNil(t, infra.Variable["bastion_private_key"])

	err = AddConnectionCheckResource(ctx, kv, deploymentID, "ComputeBastionKey", infra, "user",
		&testPk, "10.0.0.1", "ComputeBastionKey", env)
	assert.Nil(t, err)
	check = requireRemoteExec(t, infra, "ComputeBastionKey")
	assert.Equal(t, "10.0.0.2", check.Connection.BastionHost)
	assert.Equal(t, "8022", check.Connection.BastionPort)
	assert.Equal(t, "ubuntu", check.Connection.BastionUser)
	assert.Equal(t, "", check.Connection.BastionPassword)
	assert.Equal(t, "${var.bastion_private_key}", check.Connection.BastionPrivateKey, "should be set to variable passed by environment")
	assert.Contains(t, *env, "TF_VAR_bastion_private_key="+expectedKey)
}

func requireRemoteExec(t *testing.T, infra *Infrastructure, name string) RemoteExec {
	require.NotNil(t, infra.Resource["null_resource"])
	res, ok := infra.Resource["null_resource"].(map[string]interface{})
	require.True(t, ok, "should be a map[string]interface{}")
	require.NotNil(t, res[name+"-ConnectionCheck"])
	resource, ok := res[name+"-ConnectionCheck"].(*Resource)
	require.True(t, ok, "should be a *Resource")
	require.NotNil(t, resource.Provisioners)
	require.Len(t, resource.Provisioners, 1)
	require.NotNil(t, resource.Provisioners[0]["remote-exec"])
	check, ok := resource.Provisioners[0]["remote-exec"].(RemoteExec)
	require.True(t, ok, "should be a RemoteExec")
	return check
}
