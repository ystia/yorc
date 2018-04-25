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
	"crypto/x509"
	_ "net/http/httptest"
	"net"
	"crypto/tls"
	_ "crypto/x509"
	_"crypto/tls"
	
	"net/http"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/log"
	"github.com/hashicorp/consul/testutil"
	"github.com/hashicorp/consul/api"
	"io/ioutil"

	"testing"
)

	func testSSLREST(t *testing.T, client *api.Client, srv *testutil.TestServer) {
		t.Run("testSSLEnabledNoCerts", func(t *testing.T) {
			testSSLEnabledNoCerts(t, client, srv)
		})
		t.Run("testSSLVerifyNoCA", func(t *testing.T) {
			testSSLVerifyNoCA(t, client, srv)
		})
		t.Run("testSSLEnabledNoVerify", func(t *testing.T) {
			testSSLEnabledNoVerify(t, client, srv)
		})
		t.Run("testSSLEnabledVerifyUnsignedClientCerts", func(t *testing.T) {
			testSSLEnabledVerifyUnsignedClientCerts(t, client, srv)
		})
		t.Run("testSSLEnabledVerifyNoClientCerts", func(t *testing.T) {
			testSSLEnabledVerifyNoClientCerts(t, client, srv)
		})
		t.Run("testSSLEnabledVerifySignedClientCerts", func(t *testing.T) {
			testSSLEnabledVerifySignedClientCerts(t, client, srv)
		})
	}

	func testSSLEnabledNoCerts(t *testing.T, client *api.Client, srv *testutil.TestServer) {
		log.SetDebug(true)

		cfg := config.Configuration{
			SSLEnabled: true,
			SSLVerify: false,
		}

		ln, err := wrapListenerTLS(nil, cfg)
		require.NotNil(t, err, "unexpected nil error : SSL without certs should raise error")
		require.Nil(t, ln, "unexpected Not nil response")
	}

	func testSSLVerifyNoCA(t *testing.T, client *api.Client, srv *testutil.TestServer) {
		t.Parallel()

		cfg := config.Configuration{
			SSLEnabled: true,
			SSLVerify: true,
			CertFile: "testdata/server-cert.pem",
			KeyFile: "testdata/server-key.pem",
		}

		ln, err := wrapListenerTLS(nil, cfg)
		require.NotNil(t, err, "unexpected nil error : SSL verify without CA should raise error")
		require.Nil(t, ln, "unexpected Not nil response")
	}

	func testSSLEnabledNoVerify(t *testing.T, client *api.Client, srv *testutil.TestServer) {
		t.Parallel()
				
		cfg := &config.Configuration{
			SSLVerify: false,
			CertFile: "testdata/server-cert.pem",
			KeyFile:  "testdata/server-key.pem",
		}

		httpsSrv, url := makeSSLtestServer(cfg, client)
		defer httpsSrv.Shutdown()

		req, _ := http.NewRequest("GET", url+"/deployments", nil )
		req.Header.Add("Accept", "application/json")

		hclient := makeSSLtestClient("", "", "", false)
		resp, err := hclient.Do(req)
		require.Nil(t, err, "unexpected error performing request")
		require.NotNil(t, resp, "response shouldn't be nil")
		_, err = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		require.Nil(t, err, "unexpected error reading body response")
		require.Equal(t, http.StatusNoContent, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)
	}
	
	func testSSLEnabledVerifyUnsignedClientCerts(t *testing.T, client *api.Client, srv *testutil.TestServer) {
		t.Parallel()
				
		cfg := &config.Configuration{
			SSLVerify: true,
			CertFile: "testdata/server-cert.pem",
			KeyFile:  "testdata/server-key.pem",
			CAFile:   "testdata/ca-cert.pem",
		}

		httpsSrv, url := makeSSLtestServer(cfg, client)
		defer httpsSrv.Shutdown()

		req, _ := http.NewRequest("GET", url+"/deployments", nil )
		req.Header.Add("Accept", "application/json")

		hclient := makeSSLtestClient("testdata/unsigned-cert.pem", "testdata/unsigned-key.pem", "", false)
		resp, err := hclient.Do(req)
		require.NotNil(t, err, "unsigned client certs should raise an error")
		require.Nil(t, resp, "response should be nil")
	}

	func testSSLEnabledVerifyNoClientCerts(t *testing.T, client *api.Client, srv *testutil.TestServer) {
		t.Parallel()
				
		cfg := &config.Configuration{
			SSLVerify: true,
			CertFile: "testdata/server-cert.pem",
			KeyFile:  "testdata/server-key.pem",
			CAFile:   "testdata/ca-cert.pem",
		}

		httpsSrv, url := makeSSLtestServer(cfg, client)
		defer httpsSrv.Shutdown()

		req, _ := http.NewRequest("GET", url+"/deployments", nil )
		req.Header.Add("Accept", "application/json")

		hclient := makeSSLtestClient("", "", "", false)
		resp, err := hclient.Do(req)
		require.NotNil(t, err, "no client certs should raise an error")
		require.Nil(t, resp, "response should be nil")
	}


	func testSSLEnabledVerifySignedClientCerts(t *testing.T, client *api.Client, srv *testutil.TestServer) {
		t.Parallel()
				
		cfg := &config.Configuration{
			SSLVerify: true,
			CertFile: "testdata/server-cert.pem",
			KeyFile:  "testdata/server-key.pem",
			CAFile:   "testdata/ca-cert.pem",
		}

		httpsSrv, url := makeSSLtestServer(cfg, client)
		defer httpsSrv.Shutdown()

		req, _ := http.NewRequest("GET", url+"/deployments", nil )
		req.Header.Add("Accept", "application/json")

		hclient := makeSSLtestClient("testdata/client-cert.pem", "testdata/client-key.pem", "", false)
		resp, err := hclient.Do(req)
		require.Nil(t, err, "unexpected error performing request")
		require.NotNil(t, resp, "response shouldn't be nil")
		_, err = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		require.Nil(t, err, "unexpected error reading body response")
		require.Equal(t, http.StatusNoContent, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)
		
		
	} 

	func makeSSLtestServer(cfg *config.Configuration, client *api.Client) (*Server, string){
		ln, err := net.Listen("tcp", "")
		ln, err = wrapListenerTLS(ln, *cfg)
		if err != nil {
			log.Fatal( "Failed listenner ")
		}
		httpsrv := &Server{
			router: newRouter(),
			listener: ln,
			consulClient: client,

		} 
		httpsrv.registerHandlers()
		sURL := "https://" + ln.Addr().String()

		go http.Serve(httpsrv.listener, httpsrv.router)
		return httpsrv , sURL
	}
	
	func makeSSLtestClient(cliCert, cliKey , caCert string, verify bool) http.Client{
		tlsConf := &tls.Config{}
		client := http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConf,
			}, 
		}
		if !verify{
			tlsConf.InsecureSkipVerify = true
		}else{
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
		tlsConf.InsecureSkipVerify = true
		return client
	} 