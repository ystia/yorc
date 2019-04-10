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

package monitoring

import (
	"crypto/tls"
	"fmt"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-rootcerts"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v3/log"
	"net"
	"net/http"
	"path"
	"time"
)

type checkExecution interface {
	execute(timeout time.Duration) (CheckStatus, string)
}

type tcpCheckExecution struct {
	address string
	port    int
}

type tlsClientConfig struct {
	caFile        string
	caPath        string
	certFile      string
	keyFile       string
	skipTLSVerify bool
}

type httpCheckExecution struct {
	httpClient *http.Client
	scheme     string
	address    string
	port       int
	path       string
	headersMap map[string]string
	header     http.Header
	tlsConf    *tlsClientConfig
}

func (ce *tcpCheckExecution) execute(timeout time.Duration) (CheckStatus, string) {
	tcpAddr := fmt.Sprintf("%s:%d", ce.address, ce.port)
	conn, err := net.DialTimeout("tcp", tcpAddr, timeout)
	if err != nil {
		log.Debugf("[WARN] TCP check execution failed for address:%s", tcpAddr)
		return CheckStatusCRITICAL, ""
	}
	conn.Close()
	return CheckStatusPASSING, ""
}

func (ce *httpCheckExecution) execute(timeout time.Duration) (CheckStatus, string) {
	// instantiate httpClient if not already done
	if ce.httpClient == nil {
		trans := cleanhttp.DefaultTransport()
		trans.DisableKeepAlives = true
		tlsConf, err := ce.buildTlsClientConfig()
		if err != nil {
			log.Debugf("[WARN] check HTTP execution failed due to error:%+v", err)
			return CheckStatusWARNING, ""
		}
		trans.TLSClientConfig = tlsConf
		ce.httpClient = &http.Client{
			Timeout:   timeout,
			Transport: trans,
		}
	}

	// Create HTTP Request
	url := fmt.Sprintf("%s://%s:%d", ce.scheme, ce.address, ce.port)
	if ce.path != "" {
		url = path.Join(url, ce.path)
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Debugf("[WARN] check HTTP execution failed for url:%q due to error:%+v", url, err)
		return CheckStatusCRITICAL, ""
	}

	// instantiate headers
	if ce.header == nil {
		ce.header = make(http.Header)
		for k, v := range ce.headersMap {
			ce.header.Add(k, v)
		}

		if ce.header.Get("Accept") == "" {
			ce.header.Set("Accept", "text/plain, text/*, */*")
		}
	}
	req.Header = ce.header

	// Send request
	resp, err := ce.httpClient.Do(req)
	if err != nil {
		log.Debugf("[WARN] check HTTP execution failed for url:%q due to error:%+v", url, err)
		return CheckStatusCRITICAL, ""
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return CheckStatusPASSING, ""
	} else if resp.StatusCode == 429 {
		// 429 Too Many Requests (RFC 6585)
		log.Debugf("[WARN] check HTTP execution failed for url:%q with status code:%d", url, resp.StatusCode)
		return CheckStatusWARNING, ""
	} else {
		log.Debugf("[WARN] check HTTP execution failed for url:%q with status code:%d", url, resp.StatusCode)
		return CheckStatusCRITICAL, ""
	}
}

func (ce *httpCheckExecution) buildTlsClientConfig() (*tls.Config, error) {
	if ce.tlsConf != nil {
		return &tls.Config{
			InsecureSkipVerify: true,
		}, nil
	}

	tlsConfig := &tls.Config{ServerName: ce.address}
	if ce.tlsConf.certFile != "" && ce.tlsConf.keyFile != "" {
		cert, err := tls.LoadX509KeyPair(ce.tlsConf.certFile, ce.tlsConf.keyFile)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to load TLS certificates for http check with tls")
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	if ce.tlsConf.caFile != "" || ce.tlsConf.caPath != "" {
		cfg := &rootcerts.Config{
			CAFile: ce.tlsConf.caFile,
			CAPath: ce.tlsConf.caPath,
		}
		rootcerts.ConfigureTLS(tlsConfig, cfg)
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		tlsConfig.BuildNameToCertificate()
	}
	if ce.tlsConf.skipTLSVerify {
		tlsConfig.InsecureSkipVerify = true
	}
	return tlsConfig, nil
}
