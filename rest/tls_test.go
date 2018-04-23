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
		t.Run("testSSLEnabledVerifyUnsignedCerts", func(t *testing.T) {
			testSSLEnabledVerifyUnsignedCerts(t, client, srv)
		})
		t.Run("testSSLEnabledVerifySignedCerts", func(t *testing.T) {
			testSSLEnabledVerifySignedCerts(t, client, srv)
		})
	}

	func testSSLEnabledNoCerts(t *testing.T, client *api.Client, srv *testutil.TestServer) {
		log.SetDebug(true)

		cfg := config.Configuration{
			SSLEnabled: true,
			SSLVerify: false,
		}

		_, err := newTestHTTPSRouter(cfg, client, nil)
		require.NotNil(t, err, "unexpected nil error : SSL without certs should raise error")
	}

	func testSSLVerifyNoCA(t *testing.T, client *api.Client, srv *testutil.TestServer) {
		
		cfg := config.Configuration{
			SSLEnabled: true,
			SSLVerify: true,
			CertFile: "testdata/client-cert.pem",
			KeyFile: "testdata/client-key.pem",
		}

		_, err := newTestHTTPSRouter(cfg, client, nil)
		require.NotNil(t, err, "unexpected nil error : SSL verify without CA should raise error")
	}

	func testSSLEnabledNoVerify(t *testing.T, client *api.Client, srv *testutil.TestServer) {

		req := httptest.NewRequest("GET", "/deployments", nil)
		req.Header.Add("Accept", "application/json")
		cfg := config.Configuration{
			SSLEnabled: true,
			SSLVerify: false,
			CertFile: "testdata/client-cert.pem",
			KeyFile: "testdata/client-key.pem",
		}

		resp, err := newTestHTTPSRouter(cfg, client, req)
		require.Nil(t, err, "unexpected error creating router")
		log.Println(resp.StatusCode)
		_, err = ioutil.ReadAll(resp.Body)
		require.Nil(t, err, "unexpected error reading body response")
		require.NotNil(t, resp, "unexpected nil response")
	}

	func testSSLEnabledVerifyUnsignedCerts(t *testing.T, client *api.Client, srv *testutil.TestServer) {

		req := httptest.NewRequest("GET", "/deployments", nil)
		req.Header.Add("Accept", "application/json")
		cfg := config.Configuration{
			SSLEnabled: true,
			SSLVerify: true,
			CertFile: "testdata/unsigned-cert.pem",
			KeyFile: "testdata/unsigned-key.pem",
			CAFile: "testdata/ca-cert.pem",
		}

		resp, err := newTestHTTPSRouter(cfg, client, req)
		require.Nil(t, err, "unexpected error creating router")
		body, err := ioutil.ReadAll(resp.Body)
		log.Println(body)
		log.Println(resp.StatusCode)
		require.Nil(t, err, "unexpected error reading body response")
		require.NotNil(t, resp, "unexpected nil response")
	}

	func testSSLEnabledVerifySignedCerts(t *testing.T, client *api.Client, srv *testutil.TestServer) {

		req := httptest.NewRequest("GET", "/deployments", nil)
		req.Header.Add("Accept", "application/json")
		cfg := config.Configuration{
			SSLEnabled: true,
			SSLVerify: true,
			CertFile: "testdata/signed-cert.pem",
			KeyFile: "testdata/signed-key.pem",
			CAFile: "testdata/ca-cert.pem",
		}

		resp, err := newTestHTTPSRouter(cfg, client, req)
		log.Println(resp.StatusCode)
		require.Nil(t, err, "unexpected error creating router")
		_, err = ioutil.ReadAll(resp.Body)
		require.Nil(t, err, "unexpected error reading body response")
		require.NotNil(t, resp, "unexpected nil response")
	}
