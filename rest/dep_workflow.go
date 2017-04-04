package rest

import (
	"fmt"
	"net/http"

	"path"

	"github.com/julienschmidt/httprouter"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/collections"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tasks"
)

func (s *Server) newWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	deploymentID := params.ByName("id")
	workflowName := params.ByName("workflowName")

	workflows, err := deployments.GetWorkflows(s.consulClient.KV(), deploymentID)
	if err != nil {
		log.Panic(err)
	}

	if !collections.ContainsString(workflows, workflowName) {
		writeError(w, r, errNotFound)
		return
	}

	data := make(map[string]string)
	data["workflowName"] = workflowName

	taskID, err := s.tasksCollector.RegisterTaskWithData(deploymentID, tasks.CustomWorkflow, data)
	if err != nil {
		if tasks.IsAnotherLivingTaskAlreadyExistsError(err) {
			writeError(w, r, newBadRequestError(err))
			return
		}
		log.Panic(err)
	}

	w.Header().Set("Location", fmt.Sprintf("/deployments/%s/workflows/%s", deploymentID, taskID))
	w.WriteHeader(http.StatusCreated)

}

func (s *Server) listWorkflowsHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	deploymentID := params.ByName("id")
	workflows, err := deployments.GetWorkflows(s.consulClient.KV(), deploymentID)
	if err != nil {
		log.Panic(err)
	}

	wfCol := WorkflowsCollection{Workflows: make([]AtomLink, len(workflows))}
	for i, wf := range workflows {
		wfCol.Workflows[i] = newAtomLink(LinkRelWorkflow, path.Join("/deployments", deploymentID, "workflows", wf))
	}
	encodeJSONResponse(w, r, wfCol)
}

func (s *Server) getWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	deploymentID := params.ByName("id")
	workflowName := params.ByName("workflowName")
	kv := s.consulClient.KV()
	workflows, err := deployments.GetWorkflows(kv, deploymentID)
	if err != nil {
		log.Panic(err)
	}

	if !collections.ContainsString(workflows, workflowName) {
		writeError(w, r, errNotFound)
		return
	}
	wfSteps, err := deployments.ReadWorkflow(kv, deploymentID, workflowName)
	if err != nil {
		log.Panic(err)
	}
	wf := Workflow{Name: workflowName, Workflow: wfSteps}
	encodeJSONResponse(w, r, wf)
}
