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
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/config"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov/hostspool"
)

func testHostsPoolHandlers(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	t.Run("testListHostsInPool", func(t *testing.T) {
		testListHostsInPool(t, client, cfg, srv)
	})
	t.Run("testListNoHostsInPool", func(t *testing.T) {
		testListNoHostsInPool(t, client, cfg, srv)
	})
	t.Run("testListHostsInPoolWithBadFilter", func(t *testing.T) {
		testListHostsInPoolWithBadFilter(t, client, cfg, srv)
	})
	t.Run("testDeleteHostInPool", func(t *testing.T) {
		testDeleteHostInPool(t, client, cfg, srv)
	})
	t.Run("testDeleteHostInPoolNotFound", func(t *testing.T) {
		testDeleteHostInPoolNotFound(t, client, cfg, srv)
	})
	t.Run("testNewHostInPool", func(t *testing.T) {
		testNewHostInPool(t, client, cfg, srv)
	})
	t.Run("testNewHostInPoolWithConnectionError", func(t *testing.T) {
		testNewHostInPoolWithConnectionError(t, client, cfg, srv)
	})
	t.Run("testNewHostInPoolWithoutConnectionInfo", func(t *testing.T) {
		testNewHostInPoolWithoutConnectionInfo(t, client, cfg, srv)
	})
	t.Run("testNewHostInPoolWithoutPrivateKeyOrPassword", func(t *testing.T) {
		testNewHostInPoolWithoutPrivateKeyOrPassword(t, client, cfg, srv)
	})
	t.Run("testUpdateHostInPool", func(t *testing.T) {
		testUpdateHostInPool(t, client, cfg, srv)
	})
	t.Run("testUpdateHostInPoolWithConnectionError", func(t *testing.T) {
		testUpdateHostInPoolWithConnectionError(t, client, cfg, srv)
	})
	t.Run("testGetHostInPool", func(t *testing.T) {
		testGetHostInPool(t, client, cfg, srv)
	})
	t.Run("testListHostsPoolLocations", func(t *testing.T) {
		testListHostsPoolLocations(t, client, cfg, srv)
	})
	t.Run("testApplyHostsPoolConfiguration", func(t *testing.T) {
		testApplyHostsPoolConfiguration(t, client, cfg, srv)
	})
}

func testListHostsInPool(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	log.SetDebug(true)

	srv.PopulateKV(t, map[string][]byte{
		consulutil.HostsPoolPrefix + "/myHostsPoolLocationTest/host21/status": []byte("free"),
		consulutil.HostsPoolPrefix + "/myHostsPoolLocationTest/host22/status": []byte("free"),
		consulutil.HostsPoolPrefix + "/myHostsPoolLocationTest/host23/status": []byte("free"),
	})

	req := httptest.NewRequest("GET", "/hosts_pool/myHostsPoolLocationTest", nil)
	req.Header.Add("Accept", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)
	body, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)
	require.Equal(t, mimeTypeApplicationJSON, resp.Header.Get("Content-Type"))

	var collection HostsCollection
	err = json.Unmarshal(body, &collection)
	require.Nil(t, err, "unexpected error unmarshalling json body")
	require.NotNil(t, collection, "unexpected nil hosts collection")
	require.Equal(t, 3, len(collection.Hosts))
	require.Equal(t, "/hosts_pool/myHostsPoolLocationTest/host21", collection.Hosts[0].Href)
	require.Equal(t, "host", collection.Hosts[0].Rel)
	require.Equal(t, "/hosts_pool/myHostsPoolLocationTest/host22", collection.Hosts[1].Href)
	require.Equal(t, "host", collection.Hosts[1].Rel)
	require.Equal(t, "/hosts_pool/myHostsPoolLocationTest/host23", collection.Hosts[2].Href)
	require.Equal(t, "host", collection.Hosts[2].Rel)

	client.KV().DeleteTree(consulutil.HostsPoolPrefix+"/myHostsPoolLocationTest/host21", nil)
	client.KV().DeleteTree(consulutil.HostsPoolPrefix+"/myHostsPoolLocationTest/host22", nil)
	client.KV().DeleteTree(consulutil.HostsPoolPrefix+"/myHostsPoolLocationTest/host23", nil)
}

func testListNoHostsInPool(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/hosts_pool/myHostsPoolLocationTest", nil)
	req.Header.Add("Accept", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)
	_, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusNoContent, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusNoContent)
}

func testListHostsInPoolWithBadFilter(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	t.Parallel()
	log.SetDebug(true)

	req := httptest.NewRequest("GET", "/hosts_pool/myHostsPoolLocationTest", nil)
	filters := []string{"bad++"}
	q := req.URL.Query()
	for i := range filters {
		q.Add("filter", filters[i])
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Accept", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)
	_, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)
}

func testDeleteHostInPool(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	t.Parallel()
	srv.PopulateKV(t, map[string][]byte{
		consulutil.HostsPoolPrefix + "/myHostsPoolLocationTest/host13/status":                 []byte("free"),
		consulutil.HostsPoolPrefix + "/myHostsPoolLocationTest/host13/connection/host":        []byte("1.2.3.4"),
		consulutil.HostsPoolPrefix + "/myHostsPoolLocationTest/host13/connection/port":        []byte("22"),
		consulutil.HostsPoolPrefix + "/myHostsPoolLocationTest/host13/connection/private_key": []byte("test/cert1.pem"),
		consulutil.HostsPoolPrefix + "/myHostsPoolLocationTest/host13/connection/user":        []byte("user1"),
	})

	req := httptest.NewRequest("DELETE", "/hosts_pool/myHostsPoolLocationTest/host13", nil)
	resp := newTestHTTPRouter(client, cfg, req)
	_, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)
}

func testDeleteHostInPoolNotFound(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	t.Parallel()

	req := httptest.NewRequest("DELETE", "/hosts_pool/myHostsPoolLocationTest/hostNOTFOUND", nil)
	resp := newTestHTTPRouter(client, cfg, req)
	_, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusNotFound)
}

func testNewHostInPool(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
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
	req := httptest.NewRequest("PUT", "/hosts_pool/myHostsPoolLocationTest/host01", bytes.NewBuffer([]byte(string(tmp))))
	req.Header.Add("Content-Type", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)
	_, err = ioutil.ReadAll(resp.Body)
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusCreated, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusCreated)
	require.Equal(t, []string{"/hosts_pool/myHostsPoolLocationTest/host01"}, resp.Header["Location"])

	client.KV().DeleteTree(consulutil.HostsPoolPrefix+"/myHostsPoolLocationTest/host01", nil)
}

func testNewHostInPoolWithConnectionError(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	t.Parallel()

	var hostRequest HostRequest
	hostRequest.Connection = &hostspool.Connection{
		User:       "fail",
		Host:       "1.2.3.4",
		Port:       22,
		PrivateKey: "../prov/hostspool/testdata/new_key.pem",
	}
	hostRequest.Labels = append(hostRequest.Labels, MapEntry{MapEntryOperationAdd, "key1", "val1"})

	tmp, err := json.Marshal(hostRequest)
	require.Nil(t, err, "unexpected error marshalling data to provide body request")
	req := httptest.NewRequest("PUT", "/hosts_pool/myHostsPoolLocationTest/host111", bytes.NewBuffer([]byte(string(tmp))))
	req.Header.Add("Content-Type", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)
	_, err = ioutil.ReadAll(resp.Body)
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusCreated, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusCreated)
	require.Equal(t, []string{"/hosts_pool/myHostsPoolLocationTest/host111"}, resp.Header["Location"])
	require.Equal(t, []string{"failed to connect to host \"host111\": Failed to connect"}, resp.Header["Warning"])

	client.KV().DeleteTree(consulutil.HostsPoolPrefix+"/myHostsPoolLocationTest/host111", nil)
}

func testNewHostInPoolWithoutConnectionInfo(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	t.Parallel()

	var hostRequest HostRequest
	hostRequest.Labels = append(hostRequest.Labels, MapEntry{MapEntryOperationAdd, "key1", "val1"})

	tmp, err := json.Marshal(hostRequest)
	require.Nil(t, err, "unexpected error marshalling data to provide body request")
	req := httptest.NewRequest("PUT", "/hosts_pool/myHostsPoolLocationTest/host12", bytes.NewBuffer([]byte(string(tmp))))
	req.Header.Add("Content-Type", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)
	_, err = ioutil.ReadAll(resp.Body)
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)
}

func testNewHostInPoolWithoutPrivateKeyOrPassword(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
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
	req := httptest.NewRequest("PUT", "/hosts_pool/myHostsPoolLocationTest/host11", bytes.NewBuffer([]byte(string(tmp))))
	req.Header.Add("Content-Type", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)
	_, err = ioutil.ReadAll(resp.Body)
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)
}

func testUpdateHostInPool(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	t.Parallel()

	srv.PopulateKV(t, map[string][]byte{
		consulutil.HostsPoolPrefix + "/myHostsPoolLocationTest/host112/status": []byte("free"),
	})

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
	req := httptest.NewRequest("PATCH", "/hosts_pool/myHostsPoolLocationTest/host112", bytes.NewBuffer([]byte(string(tmp))))
	req.Header.Add("Content-Type", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)
	_, err = ioutil.ReadAll(resp.Body)
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)

	client.KV().DeleteTree(consulutil.HostsPoolPrefix+"/myHostsPoolLocationTest/host112", nil)
}

func testUpdateHostInPoolWithConnectionError(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	t.Parallel()

	srv.PopulateKV(t, map[string][]byte{
		consulutil.HostsPoolPrefix + "/myHostsPoolLocationTest/host113/status": []byte("free"),
	})
	var hostRequest HostRequest
	hostRequest.Connection = &hostspool.Connection{
		User:       "fail",
		Host:       "1.2.3.4",
		Port:       22,
		PrivateKey: "../prov/hostspool/testdata/new_key.pem",
	}
	hostRequest.Labels = append(hostRequest.Labels, MapEntry{MapEntryOperationAdd, "key1", "val1"})

	tmp, err := json.Marshal(hostRequest)
	require.Nil(t, err, "unexpected error marshalling data to provide body request")
	req := httptest.NewRequest("PATCH", "/hosts_pool/myHostsPoolLocationTest/host113", bytes.NewBuffer([]byte(string(tmp))))
	req.Header.Add("Content-Type", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)
	_, err = ioutil.ReadAll(resp.Body)
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)
	require.Equal(t, []string{"failed to connect to host \"host113\": Failed to connect"}, resp.Header["Warning"])

	client.KV().DeleteTree(consulutil.HostsPoolPrefix+"/myHostsPoolLocationTest/host113", nil)
}

func testGetHostInPool(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	t.Parallel()

	srv.PopulateKV(t, map[string][]byte{
		consulutil.HostsPoolPrefix + "/myHostsPoolLocationTest/host17/status":                 []byte("free"),
		consulutil.HostsPoolPrefix + "/myHostsPoolLocationTest/host17/connection/host":        []byte("1.2.3.4"),
		consulutil.HostsPoolPrefix + "/myHostsPoolLocationTest/host17/connection/port":        []byte("22"),
		consulutil.HostsPoolPrefix + "/myHostsPoolLocationTest/host17/connection/private_key": []byte("test/cert1.pem"),
		consulutil.HostsPoolPrefix + "/myHostsPoolLocationTest/host17/connection/user":        []byte("user1"),
	})

	req := httptest.NewRequest("GET", "/hosts_pool/myHostsPoolLocationTest/host17", nil)
	req.Header.Add("Accept", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)
	body, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusNoContent)
	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)
	require.Equal(t, mimeTypeApplicationJSON, resp.Header.Get("Content-Type"))

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

	client.KV().DeleteTree(consulutil.HostsPoolPrefix+"/myHostsPoolLocationTest/host17", nil)
}

func testListHostsPoolLocations(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	log.SetDebug(true)

	srv.PopulateKV(t, map[string][]byte{
		consulutil.HostsPoolPrefix + "/myHostsPoolLocationTest/host21/status": []byte("free"),
		consulutil.HostsPoolPrefix + "/myHostsPoolLocationTest/host22/status": []byte("free"),
		consulutil.HostsPoolPrefix + "/myHostsPoolLocationTest/host23/status": []byte("free"),
	})

	req := httptest.NewRequest("GET", "/hosts_pool", nil)
	req.Header.Add("Accept", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)
	body, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)
	require.Equal(t, mimeTypeApplicationJSON, resp.Header.Get("Content-Type"))

	var hostsPoolLocations HostsPoolLocations
	err = json.Unmarshal(body, &hostsPoolLocations)
	require.Nil(t, err, "unexpected error unmarshalling json body")
	require.NotNil(t, hostsPoolLocations, "unexpected nil hosts collection")
	require.Equal(t, 1, len(hostsPoolLocations.Locations))
	require.Equal(t, "myHostsPoolLocationTest", hostsPoolLocations.Locations[0])

	client.KV().DeleteTree(consulutil.HostsPoolPrefix+"/myHostsPoolLocationTest/host21", nil)
	client.KV().DeleteTree(consulutil.HostsPoolPrefix+"/myHostsPoolLocationTest/host22", nil)
	client.KV().DeleteTree(consulutil.HostsPoolPrefix+"/myHostsPoolLocationTest/host23", nil)
}

func testApplyHostsPoolConfiguration(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	poolRequest := &HostsPoolRequest{Hosts: []HostConfig{{Name: "host1", Connection: hostspool.Connection{User: "user1", Password: "password", Host: "1.2.3.4"}}}}
	tmp, err := json.Marshal(poolRequest)
	require.Nil(t, err, "unexpected error marshalling data to provide body request")

	req := httptest.NewRequest("PUT", "/hosts_pool/myHostsPoolLocationTest", bytes.NewBuffer([]byte(string(tmp))))
	req.Header.Add("Content-Type", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)
	_, err = ioutil.ReadAll(resp.Body)
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusCreated)

	client.KV().DeleteTree(consulutil.HostsPoolPrefix+"/myHostsPoolLocationTest/host1", nil)
}
