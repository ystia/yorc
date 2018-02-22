package rest

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/registry"
	"github.com/ystia/yorc/tasks"
	"net/http"
)

func (s *Server) postInfraUsageHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	infraName := params.ByName("infraName")
	log.Debugf("Posting query for getting infra usage information with infra:%q", infraName)

	// Check an infraUsageCollector with the defined infra name exists
	var reg = registry.GetRegistry()
	_, err := reg.GetInfraUsageCollector(infraName)
	if err != nil {
		log.Printf("[ERROR] %v", err)
		writeError(w, r, newBadRequestError(err))
		return
	}

	// Build a task targetID to describe query
	targetID := fmt.Sprintf("infra_usage:%s", infraName)
	taskID, err := s.tasksCollector.RegisterTask(targetID, tasks.Query)
	if err != nil {
		// If any identical query is running : we provide the related task ID
		if ok, currTaskID := tasks.IsAnotherLivingTaskAlreadyExistsError(err); ok {
			w.Header().Set("Location", fmt.Sprintf("/infra_usage/%s/tasks/%s", infraName, currTaskID))
			w.WriteHeader(http.StatusAccepted)
			return
		}
		log.Panic(err)
	}

	w.Header().Set("Location", fmt.Sprintf("/infra_usage/%s/tasks/%s", infraName, taskID))
	w.WriteHeader(http.StatusAccepted)
}
