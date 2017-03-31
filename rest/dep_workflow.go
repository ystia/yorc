package rest

import (
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tasks"
)

func (s *Server) newWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	workflowName := params.ByName("workflowName")

	data := make(map[string]string)
	data["workflowName"] = workflowName

	taskID, err := s.tasksCollector.RegisterTaskWithData(id, tasks.CustomWorkflow, data)
	if err != nil {
		if tasks.IsAnotherLivingTaskAlreadyExistsError(err) {
			writeError(w, r, newBadRequestError(err))
			return
		}
		log.Panic(err)
	}

	w.Header().Set("Location", fmt.Sprintf("/deployments/%s/workflows/%s", id, taskID))
	w.WriteHeader(http.StatusCreated)

}
