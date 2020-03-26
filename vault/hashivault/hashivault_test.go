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
	"net"
	"testing"

	"github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/vault"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/config"
	v "github.com/ystia/yorc/v4/vault"
)

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
