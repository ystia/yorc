package hashivault

import (
	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/vault"
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

	token, err := client.Auth().Token().LookupSelf()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HashiCorp Vault client, retrieving client token failed")
	}

	//log.Debugf("Janus Token %+v", token)

	return &vaultClient{vClient: client, token: token}, nil
}

type vaultClient struct {
	vClient *api.Client
	token   *api.Secret
}
