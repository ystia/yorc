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
)

func testLocationsHandlers(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	populateKV(t, srv)
	t.Run("testListLocations", func(t *testing.T) {
		testListLocations(t, client, cfg, srv)
	})
	t.Run("testDeleteLocation", func(t *testing.T) {
		testDeleteLocation(t, client, cfg, srv)
	})
	t.Run("testDeleteLocation", func(t *testing.T) {
		testDeleteInexistentLocation(t, client, cfg, srv)
	})
	t.Run("testGetLocation", func(t *testing.T) {
		testGetLocationInfo(t, client, cfg, srv)
	})
	t.Run("testCreateNoDataLocation", func(t *testing.T) {
		testCreateNoDataLocation(t, client, cfg, srv)
	})
	t.Run("testCreateLocation", func(t *testing.T) {
		testCreateLocation(t, client, cfg, srv)
	})
	t.Run("testCreateAlreadyExistentLocation", func(t *testing.T) {
		testCreateAlreadyExistentLocation(t, client, cfg, srv)
	})
	t.Run("testUpdateLocation", func(t *testing.T) {
		testUpdateLocation(t, client, cfg, srv)
	})
	t.Run("testUpdateNoDataLocation", func(t *testing.T) {
		testUpdateNoDataLocation(t, client, cfg, srv)
	})
	cleanUpKV(client)
}

func populateKV(t *testing.T, srv *testutil.TestServer) {

	// Populate KV base with some location definitions
	srv.PopulateKV(t, map[string][]byte{
		consulutil.LocationsPrefix + "/loc1": []byte("{\"Name\":\"loc1\",\"Type\":\"t1\",\"Properties\":{\"p11\":\"v11\",\"p21\":\"v21\"}}"),
		consulutil.LocationsPrefix + "/loc2": []byte("{\"Name\":\"loc2\",\"Type\":\"t2\",\"Properties\":{\"p12\":\"v12\",\"p22\":\"v22\"}}"),
		consulutil.LocationsPrefix + "/loc3": []byte("{\"Name\":\"loc3\",\"Type\":\"t3\",\"Properties\":{\"p13\":\"v13\",\"p23\":\"v23\"}}"),
	})
}

func cleanUpKV(client *api.Client) {
	client.KV().DeleteTree(consulutil.LocationsPrefix+"/loc1", nil)
	client.KV().DeleteTree(consulutil.LocationsPrefix+"/loc2", nil)
	client.KV().DeleteTree(consulutil.LocationsPrefix+"/loc3", nil)
}

func testListLocations(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	log.SetDebug(true)

	// Make a request
	req := httptest.NewRequest("GET", "/locations", nil)
	req.Header.Add("Accept", "application/json")
	resp := newTestHTTPRouter(client, cfg, req)
	body, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var collection LocationCollection
	err = json.Unmarshal(body, &collection)
	require.Nil(t, err, "unexpected error unmarshalling json body")
	require.NotNil(t, collection, "unexpected nil locations collection")
	require.Equal(t, 3, len(collection.Locations))
	require.Equal(t, "/locations/loc1", collection.Locations[0].Href)
	require.Equal(t, "location", collection.Locations[0].Rel)
	require.Equal(t, "/locations/loc2", collection.Locations[1].Href)
	require.Equal(t, "location", collection.Locations[1].Rel)
	require.Equal(t, "/locations/loc3", collection.Locations[2].Href)
	require.Equal(t, "location", collection.Locations[2].Rel)

}

func testDeleteLocation(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	req := httptest.NewRequest("DELETE", "/locations/loc1", nil)
	resp := newTestHTTPRouter(client, cfg, req)
	_, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)
}

func testDeleteInexistentLocation(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	req := httptest.NewRequest("DELETE", "/locations/loc1", nil)
	resp := newTestHTTPRouter(client, cfg, req)
	_, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")

	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)
}

func testGetLocationInfo(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	req := httptest.NewRequest("GET", "/locations/loc2", nil)
	req.Header.Add("Accept", "application/json")
	resp := newTestHTTPRouter(client, cfg, req)
	body, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.NotNil(t, body, "unexpected nil body")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)
}

func testCreateNoDataLocation(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	req := httptest.NewRequest("PUT", "/locations/loc1", nil)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	resp := newTestHTTPRouter(client, cfg, req)
	_, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)
}

func testCreateLocation(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	locationProps := config.DynamicMap{
		"p11": "v11",
		"p12": "v12",
	}
	locationReq := LocationRequest{
		Type:       "t1",
		Properties: locationProps,
	}
	tmp, err := json.Marshal(locationReq)
	require.Nil(t, err, "unexpected error marshalling data to provide body request")
	req := httptest.NewRequest("PUT", "/locations/loc1", bytes.NewBuffer([]byte(string(tmp))))
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	resp := newTestHTTPRouter(client, cfg, req)
	_, err = ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusCreated, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusCreated)
}

func testCreateAlreadyExistentLocation(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	locationProps := config.DynamicMap{
		"p11": "v11",
		"p12": "v12",
	}
	locationReq := LocationRequest{
		Type:       "t1",
		Properties: locationProps,
	}
	tmp, err := json.Marshal(locationReq)
	require.Nil(t, err, "unexpected error marshalling data to provide body request")
	req := httptest.NewRequest("PUT", "/locations/loc1", bytes.NewBuffer([]byte(string(tmp))))
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	resp := newTestHTTPRouter(client, cfg, req)
	_, err = ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)
}

func testUpdateLocation(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	locationProps := config.DynamicMap{
		"p11": "v11",
		"p12": "v15",
	}
	locationReq := LocationRequest{
		Type:       "t1",
		Properties: locationProps,
	}
	tmp, err := json.Marshal(locationReq)
	require.Nil(t, err, "unexpected error marshalling data to provide body request")
	req := httptest.NewRequest("PATCH", "/locations/loc1", bytes.NewBuffer([]byte(string(tmp))))
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	resp := newTestHTTPRouter(client, cfg, req)
	_, err = ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)
}

func testUpdateNoDataLocation(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	req := httptest.NewRequest("PATCH", "/locations/loc1", nil)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	resp := newTestHTTPRouter(client, cfg, req)
	_, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)
}
