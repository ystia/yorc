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

		resp, err := newTestHTTPSRouter(cfg, client, nil)
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

	 	resp, err := newTestHTTPSRouter(cfg, client, nil)
		require.NotNil(t, err, "unexpected nil error : SSL verify without CA should raise error")
		require.Nil(t, resp, "unexpected Not nil response")
	}

	func testSSLEnabledNoVerify(t *testing.T, client *api.Client, srv *testutil.TestServer) {
		t.Parallel()

		req := httptest.NewRequest("GET", "/deployments", nil)
		req.Header.Add("Accept", "application/json")
		cfg := config.Configuration{
			SSLEnabled: true,
			SSLVerify: false,
			CertFile: "testdata/server-cert.pem",
			KeyFile: "testdata/server-key.pem",
		}

		resp, err := newTestHTTPSRouter(cfg, client, req)
		require.Nil(t, err, "unexpected error creating router")
		_, err = ioutil.ReadAll(resp.Body)
		require.Nil(t, err, "unexpected error reading body response")
		require.NotNil(t, resp, "unexpected nil response")
		require.Equal(t, http.StatusNoContent, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)
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

		resp, err := newTestHTTPSRouter(cfg, client, req)
		require.Nil(t, err, "unexpected error creating router")
		_, err = ioutil.ReadAll(resp.Body)
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

		resp, err := newTestHTTPSRouter(cfg, client, req)
		require.Nil(t, err, "unexpected error creating router")
		_, err = ioutil.ReadAll(resp.Body)
		require.Nil(t, err, "unexpected error reading body response")
		require.NotNil(t, resp, "unexpected nil response")
		require.Equal(t, http.StatusNoContent, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)
	}
