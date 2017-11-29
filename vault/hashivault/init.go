package hashivault

import "novaforge.bull.com/starlings-janus/janus/registry"

func init() {
	registry.GetRegistry().RegisterVaultClientBuilder("hashicorp", &clientBuilder{}, registry.BuiltinOrigin)
}
