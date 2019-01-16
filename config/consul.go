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

package config

import (
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/log"
)

var consulClient *api.Client
var once sync.Once

// GetConsulClient returns the Consul client singleton instance from a given Configuration
func (cfg Configuration) GetConsulClient() (*api.Client, error) {
	var err error
	once.Do(func() {
		consulClient, err = cfg.buildConsulClientInstance()
	})
	return consulClient, err
}

// GetNewConsulClient returns a Consul client instance from a given Configuration used for plugin, tests but no for server runtime needs
// For standard server runtime need, use the GetConsulClient which returns a singleton
func (cfg Configuration) GetNewConsulClient() (*api.Client, error) {
	return cfg.buildConsulClientInstance()
}

func (cfg Configuration) buildConsulClientInstance() (*api.Client, error) {
	consulCustomConfig := api.DefaultConfig()
	consulCustomConfig.Transport.MaxIdleConnsPerHost = cfg.Consul.PubMaxRoutines
	consulCustomConfig.Transport.MaxIdleConns = cfg.Consul.PubMaxRoutines
	consulCustomConfig.Transport.IdleConnTimeout = 10 * time.Second
	consulCustomConfig.Transport.TLSHandshakeTimeout = cfg.Consul.TLSHandshakeTimeout
	log.Debugf("consul http Transport config: %+v", consulCustomConfig.Transport)
	if cfg.Consul.Address != "" {
		consulCustomConfig.Address = cfg.Consul.Address
	}
	if cfg.Consul.Datacenter != "" {
		consulCustomConfig.Datacenter = cfg.Consul.Datacenter
	}
	if cfg.Consul.Token != "" {
		consulCustomConfig.Token = cfg.Consul.Token
	}
	if cfg.Consul.SSL {
		consulCustomConfig.Scheme = "https"
	}
	if cfg.Consul.Key != "" {
		consulCustomConfig.TLSConfig.KeyFile = cfg.Consul.Key
	}
	if cfg.Consul.Cert != "" {
		consulCustomConfig.TLSConfig.CertFile = cfg.Consul.Cert
	}
	if cfg.Consul.CA != "" {
		consulCustomConfig.TLSConfig.CAFile = cfg.Consul.CA
	}
	if cfg.Consul.CAPath != "" {
		consulCustomConfig.TLSConfig.CAPath = cfg.Consul.CAPath
	}
	if !cfg.Consul.SSLVerify {
		consulCustomConfig.TLSConfig.InsecureSkipVerify = true
	}
	client, err := api.NewClient(consulCustomConfig)
	return client, errors.Wrapf(err, "Failed to connect to consul %q", cfg.Consul.Address)
}
