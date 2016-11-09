package rest

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/log"
	"strconv"
	"strings"
	"time"
)

func (s *Server) pollEvents(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	kv := s.consulClient.KV()
	if depExist, err := deployments.DoesDeploymentExists(kv, id); err != nil {
		log.Panic(err)
	} else if !depExist {
		WriteError(w, r, ErrNotFound)
		return
	}

	sub := events.NewSubscriber(kv, id)
	values := r.URL.Query()
	var err error
	var waitIndex uint64 = 1
	var timeout time.Duration = 5 * time.Minute
	if idx := values.Get("index"); idx != "" {
		if waitIndex, err = strconv.ParseUint(idx, 10, 64); err != nil {
			WriteError(w, r, NewBadRequestParameter("index", err))
			return
		}
	}

	if dur := values.Get("wait"); dur != "" {
		if timeout, err = time.ParseDuration(dur); err != nil {
			WriteError(w, r, NewBadRequestParameter("index", err))
			return
		}
		if timeout > 10*time.Minute {
			timeout = 10 * time.Minute
		}
	}

	evts, lastIdx, err := sub.NewEvents(waitIndex, timeout)
	if err != nil {
		log.Panicf("Can't retrieve events: %v", err)
	}

	eventsCollection := EventsCollection{Events: evts, LastIndex: lastIdx}
	w.Header().Add(JanusIndexHeader, strconv.FormatUint(lastIdx, 10))
	encodeJsonResponse(w, r, eventsCollection)
}

func (s *Server) pollLogs(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	kv := s.consulClient.KV()
	if depExist, err := deployments.DoesDeploymentExists(kv, id); err != nil {
		log.Panic(err)
	} else if !depExist {
		WriteError(w, r, ErrNotFound)
		return
	}
	sub := events.NewSubscriber(kv, id)
	values := r.URL.Query()
	var err error
	var waitIndex uint64 = 1
	var timeout time.Duration = 5 * time.Minute
	if idx := values.Get("index"); idx != "" {
		if waitIndex, err = strconv.ParseUint(idx, 10, 64); err != nil {
			WriteError(w, r, NewBadRequestParameter("index", err))
			return
		}
	}

	if dur := values.Get("wait"); dur != "" {
		if timeout, err = time.ParseDuration(dur); err != nil {
			WriteError(w, r, NewBadRequestParameter("index", err))
			return
		}
		if timeout > 10*time.Minute {
			timeout = 10 * time.Minute
		}
	}
	result := []string{"all"}
	if filtr := values.Get("filter"); filtr != "" {
		res := strings.Split(filtr, ",")
		if len(res) != 1 {
			result = res
		} else if strings.Contains(filtr, deployments.ENGINE_LOG_PREFIX) ||
			strings.Contains(filtr, deployments.INFRA_LOG_PREFIX) ||
			strings.Contains(filtr, deployments.SOFTWARE_LOG_PREFIX) {
			result[0] = filtr
		}
	}

	var logs []deployments.Logs
	var lastIdx uint64

	for _, data := range result {
		tmp, idx, err := sub.LogsEvents(data, waitIndex, timeout)
		if err != nil {
			log.Panicf("Can't retrieve events: %v", err)
		}
		logs = append(logs, tmp...)
		lastIdx = idx
	}

	logCollection := LogsCollection{Logs: logs, LastIndex: lastIdx}
	w.Header().Add(JanusIndexHeader, strconv.FormatUint(lastIdx, 10))
	encodeJsonResponse(w, r, logCollection)

}

func (s *Server) headEventsIndex(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	kv := s.consulClient.KV()
	if depExist, err := deployments.DoesDeploymentExists(kv, id); err != nil {
		log.Panic(err)
	} else if !depExist {
		WriteError(w, r, ErrNotFound)
		return
	}
	lastIdx, err := events.GetEventsIndex(kv, id)
	if err != nil {
		log.Panic(err)
	}
	w.Header().Add(JanusIndexHeader, strconv.FormatUint(lastIdx, 10))
	w.WriteHeader(http.StatusOK)
}

func (s *Server) headLogsEventsIndex(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	kv := s.consulClient.KV()
	if depExist, err := deployments.DoesDeploymentExists(kv, id); err != nil {
		log.Panic(err)
	} else if !depExist {
		WriteError(w, r, ErrNotFound)
		return
	}
	lastIdx, err := events.GetLogsEventsIndex(kv, id)
	if err != nil {
		log.Panic(err)
	}
	w.Header().Add(JanusIndexHeader, strconv.FormatUint(lastIdx, 10))
	w.WriteHeader(http.StatusOK)
}
