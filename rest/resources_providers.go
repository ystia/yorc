package rest

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tasks"
)

func (s *Server) getResourcesProvidersUsageHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	provName := params.ByName("providerName")

	data := make(map[string]string)
	data["query_name"] = "GetResourcesUsage"
	data["target_type"] = "resourcesProvider"
	taskID, err := s.tasksCollector.RegisterTaskWithData(provName, tasks.Query, data)
	if err != nil {
		if tasks.IsAnotherLivingTaskAlreadyExistsError(err) {
			writeError(w, r, newBadRequestError(err))
			return
		}
		log.Panic(err)
	}

	w.Header().Set("Location", fmt.Sprintf("/providers/%s/tasks/%s", provName, taskID))
	w.WriteHeader(http.StatusAccepted)
}
