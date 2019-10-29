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
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/log"
)

func (s *Server) pollEvents(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")
	if id != "" {
		if depExist, err := deployments.DoesDeploymentExists(id); err != nil {
			log.Panic(err)
		} else if !depExist {
			writeError(w, r, errNotFound)
			return
		}
	}

	values := r.URL.Query()
	var err error
	var waitIndex uint64 = 1
	timeout := 5 * time.Minute
	if idx := values.Get("index"); idx != "" {
		if waitIndex, err = strconv.ParseUint(idx, 10, 64); err != nil {
			writeError(w, r, newBadRequestParameter("index", err))
			return
		}
	}

	if dur := values.Get("wait"); dur != "" {
		if timeout, err = time.ParseDuration(dur); err != nil {
			writeError(w, r, newBadRequestParameter("index", err))
			return
		}
		if timeout > 10*time.Minute {
			timeout = 10 * time.Minute
		}
	}

	// If id parameter not set (id == ""), StatusEvents returns events for all the deployments
	evts, lastIdx, err := events.StatusEvents(id, waitIndex, timeout)
	if err != nil {
		log.Panicf("Can't retrieve events: %v", err)
	}

	eventsCollection := EventsCollection{Events: evts, LastIndex: lastIdx}
	w.Header().Add(YorcIndexHeader, strconv.FormatUint(lastIdx, 10))
	encodeJSONResponse(w, r, eventsCollection)
}

func (s *Server) pollLogs(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")
	if id != "" {
		if depExist, err := deployments.DoesDeploymentExists(id); err != nil {
			log.Panic(err)
		} else if !depExist {
			writeError(w, r, errNotFound)
			return
		}
	}
	values := r.URL.Query()
	var err error
	var waitIndex uint64 = 1
	timeout := 5 * time.Minute
	if idx := values.Get("index"); idx != "" {
		if waitIndex, err = strconv.ParseUint(idx, 10, 64); err != nil {
			writeError(w, r, newBadRequestParameter("index", err))
			return
		}
	}

	if dur := values.Get("wait"); dur != "" {
		if timeout, err = time.ParseDuration(dur); err != nil {
			writeError(w, r, newBadRequestParameter("index", err))
			return
		}
		if timeout > 10*time.Minute {
			timeout = 10 * time.Minute
		}
	}

	var logs []json.RawMessage
	var lastIdx uint64

	// If id parameter not set (id == ""), LogsEvents returns logs for all the deployments
	logs, idx, err := events.LogsEvents(id, waitIndex, timeout)
	if err != nil {
		log.Panicf("Can't retrieve events: %v", err)
	}
	lastIdx = idx

	logCollection := LogsCollection{Logs: logs, LastIndex: lastIdx}
	w.Header().Add(YorcIndexHeader, strconv.FormatUint(lastIdx, 10))
	encodeJSONResponse(w, r, logCollection)
}

func (s *Server) headEventsIndex(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")
	if id != "" {
		if depExist, err := deployments.DoesDeploymentExists(id); err != nil {
			log.Panic(err)
		} else if !depExist {
			writeError(w, r, errNotFound)
			return
		}
	}
	lastIdx, err := events.GetStatusEventsIndex(id)
	if err != nil {
		log.Panic(err)
	}
	w.Header().Add(YorcIndexHeader, strconv.FormatUint(lastIdx, 10))
	w.WriteHeader(http.StatusOK)
}

func (s *Server) headLogsEventsIndex(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")
	if id != "" {
		if depExist, err := deployments.DoesDeploymentExists(id); err != nil {
			log.Panic(err)
		} else if !depExist {
			writeError(w, r, errNotFound)
			return
		}
	}
	lastIdx, err := events.GetLogsEventsIndex(id)
	if err != nil {
		log.Panic(err)
	}
	w.Header().Add(YorcIndexHeader, strconv.FormatUint(lastIdx, 10))
	w.WriteHeader(http.StatusOK)
}
