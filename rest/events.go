package rest

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/log"
	"strconv"
	"time"
)

func (s *Server) pollEvents(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	sub := events.NewSubscriber(s.consulClient.KV(), id)
	values := r.URL.Query()
	var err error
	var waitIndex uint64 = 1
	var timeout time.Duration = 5 * time.Minute
	if idx := values.Get("index"); idx != "" {
		if waitIndex, err = strconv.ParseUint(idx, 10, 64); err != nil {
			WriteError(w, NewBadRequestParameter("index", err))
			return
		}
	}

	if dur := values.Get("wait"); dur != "" {
		if timeout, err = time.ParseDuration(dur); err != nil {
			WriteError(w, NewBadRequestParameter("index", err))
			return
		}
		if timeout > 10*time.Minute {
			timeout = 10 * time.Minute
		}
	}

	events, lastIdx, err := sub.NewEvents(waitIndex, timeout)
	if err != nil {
		log.Panicf("Can't retrieve events: %v", err)
	}

	eventsCollection := EventsCollection{Events: events, LastIndex: lastIdx}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(eventsCollection)
}
