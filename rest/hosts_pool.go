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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/helper/labelsutil"
	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/prov/hostspool"
)

func (s *Server) deleteHostInPool(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	hostname := params.ByName("host")
	err := s.hostsPoolMgr.Remove(hostname)
	if err != nil {
		if hostspool.IsHostNotFoundError(err) {
			writeError(w, r, errNotFound)
			return
		}
		if hostspool.IsBadRequestError(err) {
			writeError(w, r, newBadRequestError(err))
			return
		}
		log.Panic(err)
	}

	w.WriteHeader(http.StatusOK)
}
func (s *Server) newHostInPool(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	hostname := params.ByName("host")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Panic(err)
	}

	var host HostRequest
	err = json.Unmarshal(body, &host)
	if err != nil {
		writeError(w, r, newBadRequestError(err))
		return
	}

	if host.Connection == nil {
		writeError(w, r, newBadRequestMessage(`"connection" parameter is missing for host addition`))
		return
	}
	labels := make(map[string]string, len(host.Labels))
	for _, entry := range host.Labels {
		if entry.Op != MapEntryOperationAdd {
			writeError(w, r, newBadRequestMessage(fmt.Sprintf("unsupported operation %q for tag %q for host addition", entry.Op.String(), entry.Name)))
			return
		}
		labels[entry.Name] = entry.Value
	}

	err = s.hostsPoolMgr.Add(hostname, *host.Connection, labels)
	if err != nil {
		if hostspool.IsHostAlreadyExistError(err) || hostspool.IsBadRequestError(err) {
			writeError(w, r, newBadRequestError(err))
			return
		}
		log.Panic(err)
	}
	w.Header().Set("Location", fmt.Sprintf("/hosts_pool/%s", hostname))
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) updateHostInPool(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	hostname := params.ByName("host")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Panic(err)
	}

	var host HostRequest
	err = json.Unmarshal(body, &host)
	if err != nil {
		writeError(w, r, newBadRequestError(err))
		return
	}

	if host.Connection != nil {
		err = s.hostsPoolMgr.UpdateConnection(hostname, *host.Connection)
		if err != nil {
			if hostspool.IsBadRequestError(err) {
				writeError(w, r, newBadRequestError(err))
				return
			}
			if hostspool.IsHostNotFoundError(err) {
				writeError(w, r, errNotFound)
				return
			}
			log.Panic(err)
		}
	}
	labelsAdd := make(map[string]string)
	labelsDelete := make([]string, 0)
	for _, entry := range host.Labels {
		if entry.Op == MapEntryOperationAdd {
			labelsAdd[entry.Name] = entry.Value
		}
		if entry.Op == MapEntryOperationRemove {
			labelsDelete = append(labelsDelete, entry.Name)
		}
	}
	if len(labelsDelete) > 0 {
		err = s.hostsPoolMgr.RemoveLabels(hostname, labelsDelete)
		if err != nil {
			if hostspool.IsBadRequestError(err) {
				writeError(w, r, newBadRequestError(err))
				return
			}
			if hostspool.IsHostNotFoundError(err) {
				writeError(w, r, errNotFound)
				return
			}
			log.Panic(err)
		}
	}
	if len(labelsAdd) > 0 {
		err = s.hostsPoolMgr.AddLabels(hostname, labelsAdd)
		if err != nil {
			if hostspool.IsBadRequestError(err) {
				writeError(w, r, newBadRequestError(err))
				return
			}
			if hostspool.IsHostNotFoundError(err) {
				writeError(w, r, errNotFound)
				return
			}
			log.Panic(err)
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) getHostInPool(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	hostname := params.ByName("host")

	host, err := s.hostsPoolMgr.GetHost(hostname)
	if err != nil {
		if hostspool.IsHostNotFoundError(err) {
			writeError(w, r, errNotFound)
			return
		}
		if hostspool.IsBadRequestError(err) {
			writeError(w, r, newBadRequestError(err))
			return
		}
		log.Panic(err)
	}

	restHost := Host{Host: host, Links: make([]AtomLink, 1)}
	restHost.Links[0] = newAtomLink(LinkRelSelf, fmt.Sprintf("/hosts_pool/%s", hostname))
	encodeJSONResponse(w, r, restHost)
}

func (s *Server) listHostsInPool(w http.ResponseWriter, r *http.Request) {
	filtersString := r.URL.Query()["filter"]
	filters := make([]labelsutil.Filter, len(filtersString))
	for i := range filtersString {
		var err error
		str, err := url.QueryUnescape(filtersString[i])
		if err != nil {
			writeError(w, r, newBadRequestError(fmt.Errorf("%v with filter %s", err, filtersString)))
			return
		}
		filters[i], err = labelsutil.CreateFilter(str)
		if err != nil {
			writeError(w, r, newBadRequestError(fmt.Errorf("%v with filter %s", err, filtersString)))
			return
		}
	}

	hostsNames, warnings, checkpoint, err := s.hostsPoolMgr.List(filters...)
	if err != nil {
		log.Panic(err)
	}

	if len(hostsNames) == 0 && len(warnings) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	hostsCol := HostsCollection{}
	hostsCol.Checkpoint = checkpoint
	if len(hostsNames) > 0 {
		hostsCol.Hosts = make([]AtomLink, len(hostsNames))
	}
	for i, h := range hostsNames {
		hostsCol.Hosts[i] = newAtomLink(LinkRelHost, fmt.Sprintf("/hosts_pool/%s", h))
	}
	if len(warnings) > 0 {
		hostsCol.Warnings = make([]string, len(warnings))
	}
	for i, w := range warnings {
		hostsCol.Warnings[i] = w.Error()
	}

	encodeJSONResponse(w, r, hostsCol)
}

func (s *Server) applyHostsPool(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Panic(err)
	}
	var hostsPoolCheckpoint *uint64
	var value uint64
	if r.Method == http.MethodPost {
		if strValue, ok := r.URL.Query()["checkpoint"]; ok {
			if value, err = strconv.ParseUint(strValue[0], 10, 64); err == nil {
				hostsPoolCheckpoint = &value
			} else {
				writeError(w, r, newBadRequestError(err))
				return
			}
		} else {
			writeError(w, r, newBadRequestError(errors.Errorf("Missing query parameter checkpoint")))
			return
		}
	}

	var poolRequest HostsPoolRequest
	err = json.Unmarshal(body, &poolRequest)
	if err != nil {
		writeError(w, r, newBadRequestError(err))
		return
	}

	pool := make([]hostspool.Host, len(poolRequest.Hosts))
	for i, host := range poolRequest.Hosts {
		pool[i] = hostspool.Host{
			Name:       host.Name,
			Connection: host.Connection,
			Labels:     host.Labels,
		}
	}

	err = s.hostsPoolMgr.Apply(pool, hostsPoolCheckpoint)
	if err != nil {
		if hostspool.IsHostAlreadyExistError(err) || hostspool.IsBadRequestError(err) {
			writeError(w, r, newBadRequestError(err))
			return
		}
		log.Panic(err)
	}

	if r.Method == http.MethodPost {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}
