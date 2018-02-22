package rest

import (
	"fmt"
	"net/http"

	"path"

	"strconv"

	"github.com/julienschmidt/httprouter"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/helper/collections"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tasks"
)

func (s *Server) newWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	deploymentID := params.ByName("id")
	workflowName := params.ByName("workflowName")

	dExits, err := deployments.DoesDeploymentExists(s.consulClient.KV(), deploymentID)
	if err != nil {
		log.Panicf("%v", err)
	}
	if !dExits {
		writeError(w, r, errNotFound)
		return
	}

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
	if _, ok := r.URL.Query()["continueOnError"]; ok {
		data["continueOnError"] = strconv.FormatBool(true)
	} else {
		data["continueOnError"] = strconv.FormatBool(false)
	}

	taskID, err := s.tasksCollector.RegisterTaskWithData(deploymentID, tasks.CustomWorkflow, data)
	if err != nil {
		if ok, _ := tasks.IsAnotherLivingTaskAlreadyExistsError(err); ok {
			writeError(w, r, newBadRequestError(err))
			return
		}
		log.Panic(err)
	}

	w.Header().Set("Location", fmt.Sprintf("/deployments/%s/tasks/%s", deploymentID, taskID))
	w.WriteHeader(http.StatusCreated)

}

func (s *Server) listWorkflowsHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	deploymentID := params.ByName("id")

	dExits, err := deployments.DoesDeploymentExists(s.consulClient.KV(), deploymentID)
	if err != nil {
		log.Panicf("%v", err)
	}
	if !dExits {
		writeError(w, r, errNotFound)
		return
	}

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
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	deploymentID := params.ByName("id")
	workflowName := params.ByName("workflowName")
	kv := s.consulClient.KV()

	dExits, err := deployments.DoesDeploymentExists(s.consulClient.KV(), deploymentID)
	if err != nil {
		log.Panicf("%v", err)
	}
	if !dExits {
		writeError(w, r, errNotFound)
		return
	}

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
