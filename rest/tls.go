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

package rest

import (
	"crypto/tls"
	"net"

	"github.com/hashicorp/go-rootcerts"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/config"
)

func wrapListenerTLS(listener net.Listener, cfg config.Configuration) (net.Listener, error) {
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to load TLS certificates")
	}
	tlsConf := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	if cfg.SSLVerify {
		if cfg.CAFile == "" && cfg.CAPath == "" {
			return nil, errors.New("SSL verify enabled but no CA provided")
		}
		cfg := &rootcerts.Config{
			CAFile: cfg.CAFile,
			CAPath: cfg.CAPath,
		}
		pool, err := rootcerts.LoadCACerts(cfg)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to load CA cert(s)")
		}
		tlsConf.ClientCAs = pool
		tlsConf.ClientAuth = tls.RequireAndVerifyClientCert
		tlsConf.BuildNameToCertificate()
	}
	return tls.NewListener(listener, tlsConf), nil
}
