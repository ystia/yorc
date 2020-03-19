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
	"crypto/x509"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/log"
)

func testSSLREST(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Run("groupSSLserverPOView", func(t *testing.T) {
		t.Run("testSSLEnabledNoServerVerify", func(t *testing.T) {
			testSSLEnabledNoServerVerify(t, client, srv)
		})
		t.Run("testSSLEnabledServerVerifyUnsignedClientCerts", func(t *testing.T) {
			testSSLEnabledServerVerifyUnsignedClientCerts(t, client, srv)
		})
		t.Run("testSSLEnabledServerVerifyNoClientCerts", func(t *testing.T) {
			testSSLEnabledServerVerifyNoClientCerts(t, client, srv)
		})
		t.Run("testSSLEnabledServerVerifySignedClientCerts", func(t *testing.T) {
			testSSLEnabledServerVerifySignedClientCerts(t, client, srv)
		})
	})
	t.Run("groupSSLclientPOView", func(t *testing.T) {
		t.Run("testSSLEnabledClientVerifyUnsignedServerCerts", func(t *testing.T) {
			testSSLEnabledClientVerifyUnsignedServerCerts(t, client, srv)
		})
		t.Run("testSSLEnabledMutualVerification", func(t *testing.T) {
			testSSLEnabledMutualVerification(t, client, srv)
		})
	})

}

func TestSSLEnabledNoCerts(t *testing.T) {
	t.Parallel()

	cfg := config.Configuration{}

	ln, err := wrapListenerTLS(nil, cfg)
	require.NotNil(t, err, "unexpected nil error : SSL without certs should raise error")
	require.Nil(t, ln, "unexpected Not nil response")
}

func TestSSLVerifyNoCA(t *testing.T) {
	t.Parallel()

	cfg := config.Configuration{
		SSLVerify: true,
		CertFile:  "testdata/server-cert.pem",
		KeyFile:   "testdata/server-key.pem",
	}

	ln, err := wrapListenerTLS(nil, cfg)
	require.NotNil(t, err, "unexpected nil error : SSL verify without CA should raise error")
	require.Nil(t, ln, "unexpected Not nil response")
}

func testSSLEnabledNoServerVerify(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()

	cfg := &config.Configuration{
		SSLVerify: false,
		CertFile:  "testdata/server-cert.pem",
		KeyFile:   "testdata/server-key.pem",
	}

	httpsSrv, url := makeSSLtestServer(cfg, client)
	defer httpsSrv.Shutdown()

	req, _ := http.NewRequest("GET", url+"/deployments", nil)
	req.Header.Add("Accept", mimeTypeApplicationJSON)

	hclient := makeSSLtestClient("", "", "", false)
	resp, err := hclient.Do(req)
	require.Nil(t, err, "unexpected error performing request")
	require.NotNil(t, resp, "response shouldn't be nil")
	_, err = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	require.Nil(t, err, "unexpected error reading body response")
	require.Equal(t, http.StatusNoContent, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)
}

func testSSLEnabledServerVerifyUnsignedClientCerts(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()

	cfg := &config.Configuration{
		SSLVerify: true,
		CertFile:  "testdata/server-cert.pem",
		KeyFile:   "testdata/server-key.pem",
		CAFile:    "testdata/ca-cert.pem",
	}

	httpsSrv, url := makeSSLtestServer(cfg, client)
	defer httpsSrv.Shutdown()

	req, _ := http.NewRequest("GET", url+"/deployments", nil)
	req.Header.Add("Accept", mimeTypeApplicationJSON)

	hclient := makeSSLtestClient("testdata/unsigned-cert.pem", "testdata/unsigned-key.pem", "", false)
	resp, err := hclient.Do(req)
	require.NotNil(t, err, "unsigned client certs should raise an error")
	require.Nil(t, resp, "response should be nil")
}

func testSSLEnabledServerVerifyNoClientCerts(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()

	cfg := &config.Configuration{
		SSLVerify: true,
		CertFile:  "testdata/server-cert.pem",
		KeyFile:   "testdata/server-key.pem",
		CAFile:    "testdata/ca-cert.pem",
	}

	httpsSrv, url := makeSSLtestServer(cfg, client)
	defer httpsSrv.Shutdown()

	req, _ := http.NewRequest("GET", url+"/deployments", nil)
	req.Header.Add("Accept", mimeTypeApplicationJSON)

	hclient := makeSSLtestClient("", "", "", false)
	resp, err := hclient.Do(req)
	require.NotNil(t, err, "no client certs should raise an error")
	require.Nil(t, resp, "response should be nil")
}

func testSSLEnabledServerVerifySignedClientCerts(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()

	cfg := &config.Configuration{
		SSLVerify: true,
		CertFile:  "testdata/server-cert.pem",
		KeyFile:   "testdata/server-key.pem",
		CAFile:    "testdata/ca-cert.pem",
	}

	httpsSrv, url := makeSSLtestServer(cfg, client)
	defer httpsSrv.Shutdown()

	req, _ := http.NewRequest("GET", url+"/deployments", nil)
	req.Header.Add("Accept", mimeTypeApplicationJSON)

	hclient := makeSSLtestClient("testdata/client-cert.pem", "testdata/client-key.pem", "", false)
	resp, err := hclient.Do(req)
	require.Nil(t, err, "unexpected error performing request")
	require.NotNil(t, resp, "response shouldn't be nil")
	_, err = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	require.Nil(t, err, "unexpected error reading body response")
	require.Equal(t, http.StatusNoContent, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)

}

func testSSLEnabledClientVerifyUnsignedServerCerts(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()

	cfg := &config.Configuration{
		SSLVerify: false,
		CertFile:  "testdata/unsigned-cert.pem",
		KeyFile:   "testdata/unsigned-key.pem",
		CAFile:    "testdata/ca-cert.pem",
	}

	httpsSrv, url := makeSSLtestServer(cfg, client)
	defer httpsSrv.Shutdown()

	req, _ := http.NewRequest("GET", url+"/deployments", nil)
	req.Header.Add("Accept", mimeTypeApplicationJSON)

	hclient := makeSSLtestClient("testdata/client-cert.pem", "testdata/client-key.pem", "testdata/ca-cert.pem", true)
	resp, err := hclient.Do(req)
	require.NotNil(t, err, "performing request should raise error")
	require.Nil(t, resp, "response should be nil")
}

func testSSLEnabledMutualVerification(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()

	cfg := &config.Configuration{
		SSLVerify: true,
		CertFile:  "testdata/server-cert.pem",
		KeyFile:   "testdata/server-key.pem",
		CAFile:    "testdata/ca-cert.pem",
	}

	httpsSrv, url := makeSSLtestServer(cfg, client)
	defer httpsSrv.Shutdown()

	req, _ := http.NewRequest("GET", url+"/deployments", nil)
	req.Header.Add("Accept", mimeTypeApplicationJSON)

	hclient := makeSSLtestClient("testdata/client-cert.pem", "testdata/client-key.pem", "testdata/ca-cert.pem", true)
	resp, err := hclient.Do(req)
	require.Nil(t, err, "unexpected error performing request")
	require.NotNil(t, resp, "response shouldn't be nil")
	_, err = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	require.Nil(t, err, "unexpected error reading body response")
	require.Equal(t, http.StatusNoContent, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)
}

func makeSSLtestServer(cfg *config.Configuration, client *api.Client) (*Server, string) {
	ln, err := net.Listen("tcp", "127.0.0.1:")
	ln, err = wrapListenerTLS(ln, *cfg)
	if err != nil {
		log.Fatal(err)
	}
	httpsrv := &Server{
		router:       newRouter(),
		listener:     ln,
		consulClient: client,
	}
	httpsrv.registerHandlers()
	sURL := "https://" + ln.Addr().String()

	go http.Serve(httpsrv.listener, httpsrv.router)
	return httpsrv, sURL
}

func makeSSLtestClient(cliCert, cliKey, caCert string, verify bool) http.Client {
	tlsConf := &tls.Config{}
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConf,
		},
	}
	if !verify {
		tlsConf.InsecureSkipVerify = true
	} else {
		certpool := x509.NewCertPool()
		pem, _ := ioutil.ReadFile(caCert)
		certpool.AppendCertsFromPEM(pem)
		tlsConf.RootCAs = certpool
	}

	if cliCert == "" || cliKey == "" {
		return client
	}
	cert, _ := tls.LoadX509KeyPair(cliCert, cliKey)
	tlsConf.Certificates = []tls.Certificate{cert}
	return client
}
