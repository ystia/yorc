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

package vault

import (
	"fmt"

	"github.com/ystia/yorc/config"
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

// A ClientBuilder builds a Vault client based on Yorc configuration
type ClientBuilder interface {
	// BuildClient builds a Vault client based on Yorc configuration
	BuildClient(cfg config.Configuration) (Client, error)
}
