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

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/vault"
	"github.com/ystia/yorc/v4/config"
	v "github.com/ystia/yorc/v4/vault"
)

func createTestVaultServer(t *testing.T) (net.Listener, *api.Client) {
	t.Helper()
	return nil, nil
	core, _, rootToken := vault.TestCoreUnsealed(t)

	ln, addr := http.TestServer(t, core)

	conf := api.DefaultConfig()
	conf.Address = addr

	client, err := api.NewClient(conf)
	if err != nil {
		t.Fatal(err)
	}
	client.SetToken(rootToken)
	return ln, client
}

func createTestVaultClient(t *testing.T) v.Client {
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
	ln, client := createTestVaultServer(t)
	defer ln.Close()
	c := client.Logical()

	c.Write("secret/data/my-secret", map[string]interface{}{
		"secret": "bar",
	})

	vc := createTestVaultClient(t)
	secret, err := vc.GetSecret("secret/data/my-secret")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(secret)
}
