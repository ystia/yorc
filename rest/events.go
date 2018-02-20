package rest

import (
	"net/http"
	"strconv"
	"time"

	"encoding/json"

	"github.com/julienschmidt/httprouter"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/log"
)

func (s *Server) pollEvents(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	kv := s.consulClient.KV()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")
	if id != "" {
		if depExist, err := deployments.DoesDeploymentExists(kv, id); err != nil {
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
	evts, lastIdx, err := events.StatusEvents(kv, id, waitIndex, timeout)
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
	kv := s.consulClient.KV()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")
	if id != "" {
		if depExist, err := deployments.DoesDeploymentExists(kv, id); err != nil {
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
	logs, idx, err := events.LogsEvents(kv, id, waitIndex, timeout)
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
	kv := s.consulClient.KV()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")
	if id != "" {
		if depExist, err := deployments.DoesDeploymentExists(kv, id); err != nil {
			log.Panic(err)
		} else if !depExist {
			writeError(w, r, errNotFound)
			return
		}
	}
	lastIdx, err := events.GetStatusEventsIndex(kv, id)
	if err != nil {
		log.Panic(err)
	}
	w.Header().Add(YorcIndexHeader, strconv.FormatUint(lastIdx, 10))
	w.WriteHeader(http.StatusOK)
}

func (s *Server) headLogsEventsIndex(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	kv := s.consulClient.KV()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")
	if id != "" {
		if depExist, err := deployments.DoesDeploymentExists(kv, id); err != nil {
			log.Panic(err)
		} else if !depExist {
			writeError(w, r, errNotFound)
			return
		}
	}
	lastIdx, err := events.GetLogsEventsIndex(kv, id)
	if err != nil {
		log.Panic(err)
	}
	w.Header().Add(YorcIndexHeader, strconv.FormatUint(lastIdx, 10))
	w.WriteHeader(http.StatusOK)
}
