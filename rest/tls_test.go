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
	_ "crypto/x509"
	_"crypto/tls"
	"fmt"
	
	"net/http"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/log"
	"github.com/hashicorp/consul/testutil"
	"github.com/hashicorp/consul/api"
	"io/ioutil"
	"net/http/httptest"

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
		/*
		t.Run("testSSLEnabledVerifyUnsignedCerts", func(t *testing.T) {
			testSSLEnabledVerifyUnsignedCerts(t, client, srv)
		})
		t.Run("testSSLEnabledVerifySignedCerts", func(t *testing.T) {
			testSSLEnabledVerifySignedCerts(t, client, srv)
		}) */
	}

	func testSSLEnabledNoCerts(t *testing.T, client *api.Client, srv *testutil.TestServer) {
		log.SetDebug(true)

		cfg := config.Configuration{
			SSLEnabled: true,
			SSLVerify: false,
		}

		resp, err := newTestHTTPSRouter(cfg, client, nil, nil)
		require.NotNil(t, err, "unexpected nil error : SSL without certs should raise error")
		require.Nil(t, resp, "unexpected Not nil response")
	}

	func testSSLVerifyNoCA(t *testing.T, client *api.Client, srv *testutil.TestServer) {
		t.Parallel()

		cfg := config.Configuration{
			SSLEnabled: true,
			SSLVerify: true,
			CertFile: "testdata/server-cert.pem",
			KeyFile: "testdata/server-key.pem",
		}

	 	resp, err := newTestHTTPSRouter(cfg, client, nil, nil)
		require.NotNil(t, err, "unexpected nil error : SSL verify without CA should raise error")
		require.Nil(t, resp, "unexpected Not nil response")
	}

	func testSSLEnabledNoVerify(t *testing.T, client *api.Client, srv *testutil.TestServer) {
		log.SetDebug(true)
		t.Parallel()
		/*		
		cfg := config.Configuration{
			SSLVerify: false,
			CertFile: "testdata/server-cert.pem",
			KeyFile: "testdata/server-key.pem",
		}
		*/
		log.Debugln("-------ServersTLScreation------")
		ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Hello, client")
		}))
		//ts.Listener, _ = wrapListenerTLS(ts.Listener, cfg)
		ts.StartTLS()
		defer ts.Close()
/*
		cert, err := tls.LoadX509KeyPair("testdata/client-cert.pem", "testdata/client-key.pem")
		if err != nil {
			log.Fatal( "Failed to load TLS certificates")
		}
		
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
				InsecureSkipVerify: true,
			},
		}hclient := ts.Client()
		hclient.Transport = tr
		
		cert, err := x509.ParseCertificate(ts.TLS.Certificates[0].Certificate[0])
		if err != nil {
			log.Fatal(err)
		}

		certpool := x509.NewCertPool()
		certpool.AddCert(cert)
*/
/*
		hclient := http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					//RootCAs: certpool,
					InsecureSkipVerify: true,
				},
			},
		}
		*/
		hclient := http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
		resp, err := hclient.Get(ts.URL)
		if err != nil {
			log.Fatal(err)
		}
		
		//resp, err := newTestHTTPSRouter(cfg, client, hclient, req)
		//require.Nil(t, err, "unexpected error creating router")
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		log.Debugln("----------Body-------")
		log.Debugln(string(body))
		/*
		require.Nil(t, err, "unexpected error reading body response")
		require.NotNil(t, resp, "unexpected nil response")
		require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest) */
	}

	func testSSLEnabledVerifyUnsignedCerts(t *testing.T, client *api.Client, srv *testutil.TestServer) {
		t.Parallel()

		req := httptest.NewRequest("GET", "/deployments", nil)
		req.Header.Add("Accept", "application/json")
		cfg := config.Configuration{
			SSLEnabled: true,
			SSLVerify: true,
			CertFile: "testdata/server-cert.pem",
			KeyFile: "testdata/server-key.pem",
			CAFile: "testdata/ca-cert.pem",
		}

		resp, err := newTestHTTPSRouter(cfg, client, nil, req)
		require.Nil(t, err, "unexpected error creating router")
		body, err := ioutil.ReadAll(resp.Body)
		log.Debugln("----------Body-------")
		log.Debugln(string(body))
		require.Nil(t, err, "unexpected error reading body response")
		require.NotNil(t, resp, "unexpected nil response")
		require.Equal(t, http.StatusNoContent, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)
	}

	func testSSLEnabledVerifySignedCerts(t *testing.T, client *api.Client, srv *testutil.TestServer) {
		t.Parallel()

		req := httptest.NewRequest("GET", "/deployments", nil)
		req.Header.Add("Accept", "application/json")
		cfg := config.Configuration{
			SSLEnabled: true,
			SSLVerify: true,
			CertFile: "testdata/server-cert.pem",
			KeyFile: "testdata/server-key.pem",
			CAFile: "testdata/ca-cert.pem",
		}

		resp, err := newTestHTTPSRouter(cfg, client, nil, req)
		require.Nil(t, err, "unexpected error creating router")
		body, err := ioutil.ReadAll(resp.Body)
		log.Debugln("----------Body-------")
		log.Debugln(string(body))
		require.Nil(t, err, "unexpected error reading body response")
		require.NotNil(t, resp, "unexpected nil response")
		require.Equal(t, http.StatusNoContent, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)
	}
