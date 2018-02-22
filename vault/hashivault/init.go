package hashivault

import "github.com/ystia/yorc/registry"

func init() {
	registry.GetRegistry().RegisterVaultClientBuilder("hashicorp", &clientBuilder{}, registry.BuiltinOrigin)
}
