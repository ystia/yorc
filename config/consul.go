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
	client, err := api.NewClient(consulCustomConfig)
	return client, errors.Wrapf(err, "Failed to connect to consul %q", cfg.ConsulAddress)
}
