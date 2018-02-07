package rest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"novaforge.bull.com/starlings-janus/janus/prov/hostspool"

	"github.com/julienschmidt/httprouter"
	"novaforge.bull.com/starlings-janus/janus/log"
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
	w.WriteHeader(http.StatusOK)
}

func (s *Server) listHostsInPool(w http.ResponseWriter, r *http.Request) {
	filters := r.URL.Query()["filter"]
	if filters == nil {
		filters = make([]string, 0, 0)
	}
	hostsNames, err := s.hostsPoolMgr.List(filters...)
	if err != nil {
		log.Panic(err)
	}

	if len(hostsNames) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	hostsCol := HostsCollection{Hosts: make([]AtomLink, len(hostsNames))}
	for i, h := range hostsNames {
		hostsCol.Hosts[i] = newAtomLink(LinkRelHost, fmt.Sprintf("/hosts_pool/%s", h))
	}
	encodeJSONResponse(w, r, hostsCol)
}
