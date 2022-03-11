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
	"fmt"
	"strings"

	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/vault"
)

type clientBuilder struct {
}

func (b *clientBuilder) BuildClient(cfg config.Configuration) (vault.Client, error) {
	log.Debug("Setting up HashiCorp Vault Client connection")
	vConf := api.DefaultConfig()

	if cfg.Vault.IsSet("address") {
		if a := cfg.Vault.GetString("address"); a != "" {
			vConf.Address = a
		}
	}

	if cfg.Vault.IsSet("max_retries") {
		if a := cfg.Vault.GetInt("max_retries"); a != 0 {
			vConf.MaxRetries = a
		}
	}

	if cfg.Vault.IsSet("timeout") {
		if a := cfg.Vault.GetDuration("timeout"); a != 0 {
			vConf.Timeout = a
		}
	}
	var caCert string
	var caPath string
	var clientCert string
	var clientKey string
	var tlsServerName string
	var insecure bool

	if cfg.Vault.IsSet("ca_cert") {
		caCert = cfg.Vault.GetString("ca_cert")
	}
	if cfg.Vault.IsSet("ca_path") {
		caPath = cfg.Vault.GetString("ca_path")
	}
	if cfg.Vault.IsSet("client_cert") {
		clientCert = cfg.Vault.GetString("client_cert")
	}
	if cfg.Vault.IsSet("client_key") {
		clientKey = cfg.Vault.GetString("client_key")
	}
	if cfg.Vault.IsSet("tls_server_name") {
		tlsServerName = cfg.Vault.GetString("tls_server_name")
	}
	if cfg.Vault.IsSet("tls_skip_verify") {
		insecure = cfg.Vault.GetBool("tls_skip_verify")
	}

	// Configure the HTTP clients TLS configuration.
	t := &api.TLSConfig{
		CACert:        caCert,
		CAPath:        caPath,
		ClientCert:    clientCert,
		ClientKey:     clientKey,
		TLSServerName: tlsServerName,
		Insecure:      insecure,
	}

	if err := vConf.ConfigureTLS(t); err != nil {
		return nil, errors.Wrap(err, "failed to create HashiCorp Vault client due to a TLS configuration error")
	}

	client, err := api.NewClient(vConf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HashiCorp Vault client")
	}

	if cfg.Vault.IsSet("token") {
		if t := cfg.Vault.GetString("token"); t != "" {
			client.SetToken(t)
		}
	}

	token, err := client.Auth().Token().Lookup(client.Token())
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HashiCorp Vault client, retrieving client token failed")

	}
	// log.Debugf("token: %+v", token)
	renewable := cast.ToBool(token.Data["renewable"])
	if renewable {
		// From https://github.com/hashicorp/vault-service-broker/blob/036b95152e081eea1e4e39cb2ad534e98abea7dd/broker.go#L653
		// Use renew-self instead of lookup here because we want the freshest renew
		// and we can find out if it's renewable or not.
		token, err = client.Auth().Token().RenewSelf(0)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create HashiCorp Vault client, retrieving client token failed")

		}
		if token.Auth == nil {
			return nil, errors.New("renew self on vault token returned an empty auth")
		}
	}
	vc := &vaultClient{vClient: client, token: token, shutdownCh: make(chan struct{})}
	if renewable {
		vc.startRenewing()
	}
	return vc, nil
}

type vaultClient struct {
	vClient    *api.Client
	token      *api.Secret
	shutdownCh chan struct{}
}

func (vc *vaultClient) GetSecret(id string, options ...string) (vault.Secret, error) {
	// log.Debugf("Getting secret: %q", id)
	opts := make(map[string]string)
	for _, o := range options {
		optsList := strings.SplitN(o, "=", 2)
		if len(optsList) == 2 {
			opts[optsList[0]] = optsList[1]
		} else {
			opts[o] = ""
		}
	}
	s, err := vc.vClient.Logical().Read(id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read secret %q", id)
	}
	if s == nil {
		return nil, errors.Errorf("secret %q not found", id)
	}
	secret := &vaultSecret{Secret: s, options: opts}
	return secret, nil
}

func (vc *vaultClient) startRenewing() {
	go func() {
		renewer, err := vc.vClient.NewRenewer(&api.RenewerInput{
			Secret: vc.token,
		})
		if err != nil {
			log.Print("Failed to create renewer for the Vault token")
		}
		go renewer.Renew()
		defer renewer.Stop()

		for {
			select {
			case err := <-renewer.DoneCh():
				if err != nil {
					log.Fatal(err)
				}

				// Renewal is now over
			case renewal := <-renewer.RenewCh():
				log.Debugf("Successfully renewed vault auth token at: %v", renewal.RenewedAt)
			case <-vc.shutdownCh:
				log.Debug("stopping vault client token renewal")
				return
			}
		}
	}()
}

func (vc *vaultClient) Shutdown() error {
	return nil
}

type vaultSecret struct {
	*api.Secret
	options map[string]string
}

func (vs *vaultSecret) String() string {
	if d, ok := vs.options["data"]; ok {
		data := vs.Data
		_, okMeta := vs.Data["metadata"]
		_, okData := vs.Data["data"]
		if okMeta && okData {
			// v2 kv secret
			data = vs.Data["data"].(map[string]interface{})
		}

		return fmt.Sprint(data[d])
	}
	return fmt.Sprint(vs.Data)
}

func (vs *vaultSecret) Raw() interface{} {
	return vs.Secret
}
