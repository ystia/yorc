package server

import (
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/registry"
	"novaforge.bull.com/starlings-janus/janus/vault"
)

func buildVaultClient(cfg config.Configuration) (vault.Client, error) {
	vaultType := cfg.Vault.GetString("type")
	if vaultType == "" {
		return nil, nil
	}
	cb, err := registry.GetRegistry().GetVaultClientBuilder(vaultType)
	if err != nil {
		return nil, err
	}
	return cb.BuildClient(cfg)
}
