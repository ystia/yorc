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
	"github.com/ystia/yorc/prov/hostspool"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testListHostsInPool(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()
	log.SetDebug(true)

	srv.PopulateKV(t, map[string][]byte{
		consulutil.HostsPoolPrefix + "/host1/status": []byte("free"),
		consulutil.HostsPoolPrefix + "/host2/status": []byte("free"),
		consulutil.HostsPoolPrefix + "/host3/status": []byte("free"),
	})

	httpSrv := &Server{
		hostsPoolMgr: hostspool.NewManager(client),
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		httpSrv.listHostsInPool(w, r)
	}

	req := httptest.NewRequest("GET", "/hosts_pool", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)

	require.Nil(t, err, "unexpected error reading body response")
	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, 200, resp.StatusCode)
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
