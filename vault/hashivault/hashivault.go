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
	"encoding/base64"
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
	log.Debugf("Getting secret: %q", id)
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
	// TODO: in the future combine this with mountPathDetection to see secret engine use
	path := strings.SplitN(strings.TrimPrefix(strings.TrimSpace(id), "/"), "/", 2)
	var secret vault.Secret
	switch path[0] {
	case "gcp":
		secret = &gcpSecret{defaultSecret{Secret: s, Options: opts}}
		//return &gcpSecret{Secret: s, options: opts}, nil
	case "secret":
		// /secret/data/foo is a kv v2 path
		if strings.HasPrefix(path[1], "data") {
			secret = &kvV2Secret{kvSecret{defaultSecret{Secret: s, Options: opts}}}
		} else {
			secret = &kvV1Secret{kvSecret{defaultSecret{Secret: s, Options: opts}}}
		}
	default:
		secret = &defaultSecret{Secret: s, Options: opts}
	}
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

/* Vault users could register secret engine at different path. For example a gcp secret engine could be at /infra-prod/
   So it will be better to know which path corresponds to which type to correctly manage secrets. TODO: In the future
func (vc *vaultClient) registerMounts()error{
	mounts, err := vc.vClient.Sys().ListMounts()
	if err != nil {
		return errors.Errorf("Unable to list mounts. Err : %v", err)
	}
	for path, mount := range mounts {
		type := mount.Type
	}
}
*/

func (vc *vaultClient) Revoke(secret *api.Secret) error {
	if secret.LeaseID == "" {
		log.Debugf("Secret %v is non-revocable since it as no lease_id.", secret.WrapInfo.CreationPath)
		return nil
	}
	err := vc.vClient.Sys().Revoke(secret.LeaseID)
	if err != nil {
		return errors.Errorf("Secret revocation failed. Err : %v", err)
	}
	return nil
}

func (vc *vaultClient) Shutdown() error {
	return nil
}

// Default secret is the basic secret type, used type is not yet supported
type defaultSecret struct {
	*api.Secret
	Options map[string]string
}

func (ds *defaultSecret) Raw() interface{} {
	return ds.Secret
}

func (ds *defaultSecret) String() string {
	if key, ok := ds.Options["data"]; ok {
		return fmt.Sprint(ds.Data[key])
	}
	return fmt.Sprint(ds.Data)
}

type kvSecret struct {
	defaultSecret
}

// Generic method for kv secret engines
func (kvs *kvSecret) String() string {
	if key, ok := kvs.Options["data"]; ok {
		if secretValue, ok := getData(kvs.Secret)[key]; ok {
			return fmt.Sprint(secretValue)
		}
	}
	return fmt.Sprint(getData(kvs.Secret))
}

// Management of KV v1 secret engine
type kvV1Secret struct {
	kvSecret
}

// Management of KV v2 secret engine
type kvV2Secret struct {
	kvSecret
}

// Management of GCP secret engine
type gcpSecret struct {
	defaultSecret
}

func (gcps *gcpSecret) String() string {
	data := gcps.Data
	if key, ok := gcps.Options["data"]; ok {
		switch key {
		// Private key is base64 encoded
		case "private_key_data":
			decodedKey, err := base64.StdEncoding.DecodeString(fmt.Sprint(data[key]))
			if err == nil {
				return string(decodedKey)
			}
			break
		default:
			return fmt.Sprint(data[key])
		}
	}
	return fmt.Sprint(data)
}

//DEPRECATED use defaultSecret type instead
type vaultSecret struct {
	*api.Secret
	options map[string]string
}

//DEPRECATED use defaultSecret type instead
func (vs *vaultSecret) String() string {
	//Option exists
	if key, ok := vs.options["data"]; ok {
		if secretValue, ok := getData(vs.Secret)[key]; ok {
			return fmt.Sprint(secretValue)
		}
	}
	return fmt.Sprint(getData(vs.Secret))
}

//DEPRECATED use defaultSecret type instead
func (vs *vaultSecret) Raw() interface{} {
	return vs.Secret
}

// In case of data nested into another data map return the actual data.
// KV secret engine has 2 version; version 2 manage secret versioning. For a KV v2, secret are sored in a struct like : map[data:map[k1: val1 k2: val2] metadata:map[created_time:2020 version:3]]
// For a KV v1, only the data map is stored
func getData(secret *api.Secret) map[string]interface{} {
	data := secret.Data
	// Case of a KV v2 :
	if nestedData, ok := data["data"].(map[string]interface{}); ok {
		return nestedData
	}
	return data
}
