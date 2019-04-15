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
	"crypto/x509"
	"fmt"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v3/log"
	"net"
	"net/http"
	"strings"
	"time"
)

type checkExecution interface {
	execute(timeout time.Duration) (CheckStatus, string)
}

type tcpCheckExecution struct {
	address string
}

type httpCheckExecution struct {
	httpClient *http.Client
	url        string
	header     http.Header
}

func newTCPCheckExecution(address string, port int) *tcpCheckExecution {
	tcpAddr := fmt.Sprintf("%s:%d", address, port)
	return &tcpCheckExecution{
		address: tcpAddr,
	}
}

func (ce *tcpCheckExecution) execute(timeout time.Duration) (CheckStatus, string) {
	conn, err := net.DialTimeout("tcp", ce.address, timeout)
	if err != nil {
		log.Debugf("[WARN] TCP check execution failed for address:%s", ce.address)
		return CheckStatusCRITICAL, ""
	}
	conn.Close()
	return CheckStatusPASSING, ""
}

func newHTTPCheckExecution(address string, port int, scheme, urlPath string, headers, tlsConfig map[string]string) (*httpCheckExecution, error) {
	execution := &httpCheckExecution{}
	// set http url
	execution.url = fmt.Sprintf("%s://%s:%d", scheme, address, port)
	if urlPath != "" {
		if !strings.HasPrefix(urlPath, "/") {
			urlPath = "/" + urlPath
		}
		execution.url = fmt.Sprintf("%s%s", execution.url, urlPath)
	}
	log.Debugf("url=%q", execution.url)

	// Set default header
	execution.header = make(http.Header)
	for k, v := range headers {
		execution.header.Add(k, v)
	}
	if execution.header.Get("Accept") == "" {
		execution.header.Set("Accept", "text/plain, text/*, */*")
	}

	// Set HTTPClient
	trans := cleanhttp.DefaultTransport()
	trans.DisableKeepAlives = true
	tlsConf, err := buildTLSClientConfig(address, tlsConfig)
	if err != nil {
		return nil, err
	}
	trans.TLSClientConfig = tlsConf
	execution.httpClient = &http.Client{
		Transport: trans,
	}
	return execution, nil
}

func (ce *httpCheckExecution) execute(timeout time.Duration) (CheckStatus, string) {
	// Create HTTP Request
	req, err := http.NewRequest("GET", ce.url, nil)
	if err != nil {
		return CheckStatusCRITICAL, fmt.Sprintf("[WARN] check HTTP execution failed for url:%q due to error:%v", ce.url, err)
	}
	req.Header = ce.header

	// Send request
	ce.httpClient.Timeout = timeout
	resp, err := ce.httpClient.Do(req)
	if err != nil {
		return CheckStatusCRITICAL, fmt.Sprintf("[WARN] check HTTP execution failed for url:%q due to error:%v", ce.url, err)
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return CheckStatusPASSING, ""
	} else if resp.StatusCode == 429 {
		// 429 Too Many Requests (RFC 6585)
		return CheckStatusWARNING, fmt.Sprintf("[WARN] check HTTP execution failed for url:%q with status code:%d", ce.url, resp.StatusCode)
	} else {
		return CheckStatusCRITICAL, fmt.Sprintf("[WARN] check HTTP execution failed for url:%q with status code:%d", ce.url, resp.StatusCode)
	}
}

func buildTLSClientConfig(address string, tlsConfigMap map[string]string) (*tls.Config, error) {
	if tlsConfigMap == nil || len(tlsConfigMap) == 0 {
		return &tls.Config{
			InsecureSkipVerify: true,
		}, nil
	}

	tlsConfig := &tls.Config{ServerName: address}
	if tlsConfigMap["client_cert"] != "" && tlsConfigMap["client_key"] != "" {
		cert, err := tls.X509KeyPair([]byte(tlsConfigMap["client_cert"]), []byte(tlsConfigMap["client_key"]))
		if err != nil {
			return nil, errors.Wrap(err, "Failed to load TLS certificates for http check with tls")
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	if tlsConfigMap["ca_cert"] != "" {
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM([]byte(tlsConfigMap["ca_cert"])) {
			return nil, errors.New("bad CA Cert")
		}
		tlsConfig.RootCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		tlsConfig.BuildNameToCertificate()
	}
	if tlsConfigMap["skip_verify"] == "true" {
		tlsConfig.InsecureSkipVerify = true
	}
	return tlsConfig, nil
}
