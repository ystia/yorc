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

package hashivault

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/vault"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/config"
	v "github.com/ystia/yorc/v4/vault"
)

// Sample from https://www.vaultproject.io/api/secret/kv/kv-v1.html#sample-response
// and https://www.vaultproject.io/api/secret/kv/kv-v2.html#sample-response-1
// and https://www.vaultproject.io/api-docs/secret/gcp/#sample-response-4
var KVv1ReadSampleResponse = `
{
	"auth": null,
	"data": {
		"foo": "bar",
		"baz": "qux",
		"ttl": "1h"
	},
	"lease_duration": 3600,
	"lease_id": "",
	"renewable": false
}
`
var KVv2ReadsSampleResponse = `
{
	"data": {
		"data": {
		  "baz": "qux",
		  "foo": "bar",
		  "ttl": "1h"
		},
		"metadata": {
		  "created_time": "2018-03-22T02:24:06.945319214Z",
		  "deletion_time": "",
		  "destroyed": false,
		  "version": 1
		}
	  }
}
`

// "<private-key-data>" encoded in base64 -> "PHByaXZhdGUta2V5LWRhdGE+Cg=="
var gcpReadSampleResponse = `
{
	"request_id": "12345",
	"lease_id": "gcp/key/my-key-roleset/9876",
	"renewable": true,
	"lease_duration": 3600,
	"data": {
	  "private_key_data": "PHByaXZhdGUta2V5LWRhdGE+",
	  "key_algorithm": "TYPE_GOOGLE_CREDENTIALS_FILE",
	  "key_type": "KEY_ALG_RSA_2048"
	},
	"wrap_info": null,
	"warnings": null,
	"auth": null
  }
`

func Test_vaultSecret_String(t *testing.T) {
	tests := []struct {
		sampleResponse string
		secret         v.Secret
		expected       string
	}{
		{
			KVv1ReadSampleResponse,
			&kvV1Secret{nil, map[string]string{"data": "foo"}},
			"bar",
		}, {
			KVv2ReadsSampleResponse,
			&kvV2Secret{nil, map[string]string{"data": "foo"}},
			"bar",
		}, {
			gcpReadSampleResponse,
			&gcpSecret{nil, map[string]string{"data": "private_key_data"}},
			"<private-key-data>",
		},
	}
	for _, testCase := range tests {
		err := json.Unmarshal([]byte(testCase.sampleResponse), testCase.secret)
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, testCase.expected, testCase.secret.String(), "String() should retrieve the good value of the data map !")
	}

}

func createTestVaultServer(t *testing.T) (ln net.Listener, addr string, rootToken string) {
	t.Helper()
	core, _, rootToken := vault.TestCoreUnsealed(t)

	ln, addr = http.TestServer(t, core)
	return ln, addr, rootToken
}

func createTestVaultClient(t *testing.T) (net.Listener, v.Client) {
	ln, addr, rootToken := createTestVaultServer(t)
	b := clientBuilder{}
	client, err := b.BuildClient(config.Configuration{Vault: config.DynamicMap{
		"address":         addr,
		"token":           rootToken,
		"tls_skip_verify": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	return ln, client
}

func connectToLocalVaultClient(t *testing.T) v.Client {
	b := clientBuilder{}
	client, err := b.BuildClient(config.Configuration{Vault: config.DynamicMap{
		"address":         "http://127.0.0.1:8200",
		"token":           "root",
		"tls_skip_verify": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func Test_vaultClient_GetSecret(t *testing.T) {
	ln, vc := createTestVaultClient(t)
	defer ln.Close()
	c, ok := vc.(*vaultClient)
	if !ok {
		t.Fatal("Conversion error")
	}
	client := c.vClient.Logical()

	secretPath := "secret/data/my-secret"
	key := "password"
	myStrongPassword := "MySr0ngP4ssw0rd"
	client.Write(secretPath, map[string]interface{}{
		key: myStrongPassword,
	})

	secret, err := vc.GetSecret(secretPath, "data="+key)
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, myStrongPassword, secret.String(), "Secret get should be equals !")
}
