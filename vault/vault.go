package vault

import (
	"fmt"

	"novaforge.bull.com/starlings-janus/janus/config"
)

// Client is the common interface for Vault clients.
//
// Basically it allows to interact with a Vault to resolve a secret.
type Client interface {
	// GetSecret allows to retrieve a secret within a Vault.
	//
	// Some extra options may be given. It is up to the Vault client implementation to choose
	// to honor them.
	GetSecret(id string, options ...string) (Secret, error)

	// Shutdown tells a Client to shutdown, close all connections and release any created resources
	Shutdown() error
}

// A Secret is a resolved secret instance.
//
// Based on the Vault implementation it could be the resolved secret like a string for instance
// or the implementation secret wrapped into a vault.Secret interface-compatible structure.
type Secret interface {
	// Any secret should be serializable into a string to get the resulting data
	fmt.Stringer
	// Raw returns the Vault implementation secret.
	//
	// This is useful when used into Go templates like in config files.
	Raw() interface{}
}

// A ClientBuilder builds a Vault client based on Janus configuration
type ClientBuilder interface {
	// BuildClient builds a Vault client based on Janus configuration
	BuildClient(cfg config.Configuration) (Client, error)
}
