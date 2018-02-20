package server

import (
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/registry"
	"github.com/ystia/yorc/vault"
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
