package config

import (
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
)

// GetConsulClient returns a Consul client from a given Configuration
func (cfg Configuration) GetConsulClient() (*api.Client, error) {

	consulCustomConfig := api.DefaultConfig()
	if cfg.ConsulAddress != "" {
		consulCustomConfig.Address = cfg.ConsulAddress
	}
	if cfg.ConsulDatacenter != "" {
		consulCustomConfig.Datacenter = cfg.ConsulDatacenter
	}
	if cfg.ConsulToken != "" {
		consulCustomConfig.Token = cfg.ConsulToken
	}
	if cfg.ConsulSSL {
		consulCustomConfig.Scheme = "https"
	}
	if cfg.ConsulKey != "" {
		consulCustomConfig.TLSConfig.KeyFile = cfg.ConsulKey
	}
	if cfg.ConsulCert != "" {
		consulCustomConfig.TLSConfig.CertFile = cfg.ConsulCert
	}
	if cfg.ConsulCA != "" {
		consulCustomConfig.TLSConfig.CAFile = cfg.ConsulCA
	}
	if cfg.ConsulCAPath != "" {
		consulCustomConfig.TLSConfig.CAPath = cfg.ConsulCAPath
	}
	if !cfg.ConsulSSLVerify {
		consulCustomConfig.TLSConfig.InsecureSkipVerify = true
	}
	client, err := api.NewClient(consulCustomConfig)
	return client, errors.Wrapf(err, "Failed to connect to consul %q", cfg.ConsulAddress)
}
