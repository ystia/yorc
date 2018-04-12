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
	"encoding/json"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testHostsPoolHandlers(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Run("testListHostsInPool", func(t *testing.T) {
		testListHostsInPool(t, client, srv)
	})
	t.Run("testListNoHostsInPool", func(t *testing.T) {
		testListNoHostsInPool(t, client, srv)
	})
	t.Run("testListHostsInPoolWithBadFilter", func(t *testing.T) {
		testListHostsInPoolWithBadFilter(t, client, srv)
	})
	t.Run("testDeleteHostInPool", func(t *testing.T) {
		testDeleteHostInPool(t, client, srv)
	})
	t.Run("testDeleteHostInPoolNotFound", func(t *testing.T) {
		testDeleteHostInPoolNotFound(t, client, srv)
	})
}


func testListHostsInPool(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()
	log.SetDebug(true)

	srv.PopulateKV(t, map[string][]byte{
		consulutil.HostsPoolPrefix + "/host1/status": []byte("free"),
		consulutil.HostsPoolPrefix + "/host2/status": []byte("free"),
		consulutil.HostsPoolPrefix + "/host3/status": []byte("free"),
	})

	req := httptest.NewRequest("GET", "/hosts_pool", nil)
	req.Header.Add("Accept", "application/json")
	resp := newTestHTTPRouter(client, req)
	body, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var collection HostsCollection
	err = json.Unmarshal(body, &collection)
	require.Nil(t, err, "unexpected error unmarshalling json body")
	require.NotNil(t, collection, "unexpected nil hosts collection")
	require.Equal(t, 3, len(collection.Hosts))
	require.Equal(t, "/hosts_pool/host1", collection.Hosts[0].Href)
	require.Equal(t, "host", collection.Hosts[0].Rel)
	require.Equal(t, "/hosts_pool/host2", collection.Hosts[1].Href)
	require.Equal(t, "host", collection.Hosts[1].Rel)
	require.Equal(t, "/hosts_pool/host3", collection.Hosts[2].Href)
	require.Equal(t, "host", collection.Hosts[2].Rel)
}

func testListNoHostsInPool(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/hosts_pool", nil)
	req.Header.Add("Accept", "application/json")
	resp := newTestHTTPRouter(client, req)
	_, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusNoContent, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusNoContent)
}

func testListHostsInPoolWithBadFilter(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()
	log.SetDebug(true)

	req := httptest.NewRequest("GET", "/hosts_pool", nil)
	filters := []string{"bad++"}
	q := req.URL.Query()
	for i := range filters {
		q.Add("filter", filters[i])
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Accept", "application/json")
	resp := newTestHTTPRouter(client, req)
	_, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)
}

func testDeleteHostInPool(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()
	srv.PopulateKV(t, map[string][]byte{
		consulutil.HostsPoolPrefix + "/host1/status": []byte("free"),
		consulutil.HostsPoolPrefix + "/host1/connection/host":    []byte("1.2.3.4"),
		consulutil.HostsPoolPrefix + "/host1/connection/port":    []byte("22"),
		consulutil.HostsPoolPrefix + "/host1/connection/private_key":    []byte("test/cert1.pem"),
		consulutil.HostsPoolPrefix + "/host1/connection/user":    []byte("user1"),
	})

	req := httptest.NewRequest("DELETE", "/hosts_pool/host1", nil)
	resp := newTestHTTPRouter(client, req)
	_, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)
}

func testDeleteHostInPoolNotFound(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()

	req := httptest.NewRequest("DELETE", "/hosts_pool/host1", nil)
	resp := newTestHTTPRouter(client, req)
	_, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusNotFound)
}



