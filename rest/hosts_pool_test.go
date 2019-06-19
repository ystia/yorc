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
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov/hostspool"
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
	t.Run("testNewHostInPool", func(t *testing.T) {
		testNewHostInPool(t, client, srv)
	})
	t.Run("testNewHostInPoolWithoutConnectionInfo", func(t *testing.T) {
		testNewHostInPoolWithoutConnectionInfo(t, client, srv)
	})
	t.Run("testNewHostInPoolWithoutPrivateKeyOrPassword", func(t *testing.T) {
		testNewHostInPoolWithoutPrivateKeyOrPassword(t, client, srv)
	})
	t.Run("testGetHostInPool", func(t *testing.T) {
		testGetHostInPool(t, client, srv)
	})
}

func testListHostsInPool(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	log.SetDebug(true)

	srv.PopulateKV(t, map[string][]byte{
		consulutil.HostsPoolPrefix + "/host21/status": []byte("free"),
		consulutil.HostsPoolPrefix + "/host22/status": []byte("free"),
		consulutil.HostsPoolPrefix + "/host23/status": []byte("free"),
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
	require.Equal(t, "/hosts_pool/host21", collection.Hosts[0].Href)
	require.Equal(t, "host", collection.Hosts[0].Rel)
	require.Equal(t, "/hosts_pool/host22", collection.Hosts[1].Href)
	require.Equal(t, "host", collection.Hosts[1].Rel)
	require.Equal(t, "/hosts_pool/host23", collection.Hosts[2].Href)
	require.Equal(t, "host", collection.Hosts[2].Rel)

	client.KV().DeleteTree(consulutil.HostsPoolPrefix+"/host21", nil)
	client.KV().DeleteTree(consulutil.HostsPoolPrefix+"/host22", nil)
	client.KV().DeleteTree(consulutil.HostsPoolPrefix+"/host23", nil)
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
		consulutil.HostsPoolPrefix + "/host13/status":                 []byte("free"),
		consulutil.HostsPoolPrefix + "/host13/connection/host":        []byte("1.2.3.4"),
		consulutil.HostsPoolPrefix + "/host13/connection/port":        []byte("22"),
		consulutil.HostsPoolPrefix + "/host13/connection/private_key": []byte("test/cert1.pem"),
		consulutil.HostsPoolPrefix + "/host13/connection/user":        []byte("user1"),
	})

	req := httptest.NewRequest("DELETE", "/hosts_pool/host13", nil)
	resp := newTestHTTPRouter(client, req)
	_, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)
}

func testDeleteHostInPoolNotFound(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()

	req := httptest.NewRequest("DELETE", "/hosts_pool/hostNOTFOUND", nil)
	resp := newTestHTTPRouter(client, req)
	_, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusNotFound)
}

func testNewHostInPool(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()

	var hostRequest HostRequest
	hostRequest.Connection = &hostspool.Connection{
		User:       "user1",
		Host:       "1.2.3.4",
		Port:       22,
		PrivateKey: "../prov/hostspool/testdata/new_key.pem",
	}
	hostRequest.Labels = append(hostRequest.Labels, MapEntry{MapEntryOperationAdd, "key1", "val1"})

	tmp, err := json.Marshal(hostRequest)
	require.Nil(t, err, "unexpected error marshalling data to provide body request")
	req := httptest.NewRequest("PUT", "/hosts_pool/host11", bytes.NewBuffer([]byte(string(tmp))))
	req.Header.Add("Content-Type", "application/json")
	resp := newTestHTTPRouter(client, req)
	_, err = ioutil.ReadAll(resp.Body)
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusCreated, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusCreated)
	require.Equal(t, []string{"/hosts_pool/host11"}, resp.Header["Location"])

	client.KV().DeleteTree(consulutil.HostsPoolPrefix+"/host11", nil)
}

func testNewHostInPoolWithoutConnectionInfo(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()

	var hostRequest HostRequest
	hostRequest.Labels = append(hostRequest.Labels, MapEntry{MapEntryOperationAdd, "key1", "val1"})

	tmp, err := json.Marshal(hostRequest)
	require.Nil(t, err, "unexpected error marshalling data to provide body request")
	req := httptest.NewRequest("PUT", "/hosts_pool/host12", bytes.NewBuffer([]byte(string(tmp))))
	req.Header.Add("Content-Type", "application/json")
	resp := newTestHTTPRouter(client, req)
	_, err = ioutil.ReadAll(resp.Body)
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)
}

func testNewHostInPoolWithoutPrivateKeyOrPassword(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()

	var hostRequest HostRequest
	hostRequest.Connection = &hostspool.Connection{
		User: "user1",
		Host: "1.2.3.4",
		Port: 22,
	}
	hostRequest.Labels = append(hostRequest.Labels, MapEntry{MapEntryOperationAdd, "key1", "val1"})

	tmp, err := json.Marshal(hostRequest)
	require.Nil(t, err, "unexpected error marshalling data to provide body request")
	req := httptest.NewRequest("PUT", "/hosts_pool/host11", bytes.NewBuffer([]byte(string(tmp))))
	req.Header.Add("Content-Type", "application/json")
	resp := newTestHTTPRouter(client, req)
	_, err = ioutil.ReadAll(resp.Body)
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)
}

func testGetHostInPool(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()

	srv.PopulateKV(t, map[string][]byte{
		consulutil.HostsPoolPrefix + "/host17/status":                 []byte("free"),
		consulutil.HostsPoolPrefix + "/host17/connection/host":        []byte("1.2.3.4"),
		consulutil.HostsPoolPrefix + "/host17/connection/port":        []byte("22"),
		consulutil.HostsPoolPrefix + "/host17/connection/private_key": []byte("test/cert1.pem"),
		consulutil.HostsPoolPrefix + "/host17/connection/user":        []byte("user1"),
	})

	req := httptest.NewRequest("GET", "/hosts_pool/host17", nil)
	req.Header.Add("Accept", "application/json")
	resp := newTestHTTPRouter(client, req)
	body, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusNoContent)
	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var host Host
	err = json.Unmarshal(body, &host)
	require.Nil(t, err, "unexpected error unmarshalling json body")
	require.NotNil(t, host, "unexpected nil hosts collection")
	require.Equal(t, "host17", host.Name, "unexpected host name %q", host.Name)
	require.NotNil(t, host.Connection, "unexpected nil connection")
	require.Equal(t, 22, int(host.Connection.Port), "unexpected host connection port %q", host.Connection.Port)
	require.Equal(t, "1.2.3.4", host.Connection.Host, "unexpected host connection port %q", host.Connection.Host)
	require.Equal(t, "user1", host.Connection.User, "unexpected host connection port %q", host.Connection.User)
	require.Equal(t, "test/cert1.pem", host.Connection.PrivateKey, "unexpected host connection private key %q", host.Connection.PrivateKey)

	require.Equal(t, hostspool.HostStatusFree, host.Status, "unexpected not free host status")

	client.KV().DeleteTree(consulutil.HostsPoolPrefix+"/host17", nil)
}
