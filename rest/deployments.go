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
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/hashicorp/go-multierror"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/internal/operations"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tasks"
)

func extractFile(f *zip.File, path string) {
	fileReader, err := f.Open()
	if err != nil {
		log.Panic(err)
	}
	defer fileReader.Close()

	targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		log.Panic(err)
	}
	defer targetFile.Close()

	if _, err := io.Copy(targetFile, fileReader); err != nil {
		log.Panic(err)
	}
}

// unzipArchiveGetTopology unzips an archive and return the path to its topology
// yaml file
func unzipArchiveGetTopology(workingDir, deploymentID string, r *http.Request) (string, *Error) {
	var err error
	var file *os.File

	uploadPath := filepath.Join(workingDir, "deployments", deploymentID)

	if err = os.MkdirAll(uploadPath, 0775); err != nil {
		return "", newInternalServerError(err)
	}

	file, err = os.Create(fmt.Sprintf("%s/deployment.zip", uploadPath))
	// check err
	if err != nil {
		return "", newInternalServerError(err)
	}

	_, err = io.Copy(file, r.Body)
	if err != nil {
		return "", newInternalServerError(err)
	}
	destDir := filepath.Join(uploadPath, "overlay")
	if err = os.MkdirAll(destDir, 0775); err != nil {
		return "", newInternalServerError(err)
	}
	zipReader, err := zip.OpenReader(file.Name())
	if err != nil {
		return "", newInternalServerError(err)
	}
	defer zipReader.Close()

	// Iterate through the files in the archive,
	// and extract them.
	for _, f := range zipReader.File {
		fPath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			// Ensure that we have full rights on directory to be able to extract files into them
			if err = os.MkdirAll(fPath, f.Mode()|0700); err != nil {
				return "", newInternalServerError(err)
			}
			continue
		}
		extractFile(f, fPath)
	}

	patterns := []struct {
		pattern string
	}{
		{"*.yml"},
		{"*.yaml"},
	}
	var yamlList []string
	for _, pattern := range patterns {
		var yamls []string
		if yamls, err = filepath.Glob(filepath.Join(destDir, pattern.pattern)); err != nil {
			return "", newInternalServerError(err)
		}
		yamlList = append(yamlList, yamls...)
	}
	if len(yamlList) != 1 {
		err = fmt.Errorf(
			"one and only one YAML (.yml or .yaml) file should be present at the root of archive for deployment %s",
			deploymentID)
		return "", newBadRequestError(err)
	}

	return yamlList[0], nil

}

func (s *Server) newDeploymentHandler(w http.ResponseWriter, r *http.Request) {

	var uid string
	if r.Method == http.MethodPut {
		var params httprouter.Params
		ctx := r.Context()
		params = ctx.Value(paramsLookupKey).(httprouter.Params)
		id := params.ByName("id")
		id, err := url.QueryUnescape(id)
		if err != nil {
			log.Panicf("%v", errors.Wrapf(err, "Failed to unescape given deployment id %q", id))
		}
		matched, err := regexp.MatchString(YorcDeploymentIDPattern, id)
		if err != nil {
			log.Panicf("%v", errors.Wrapf(err, "Failed to parse given deployment id %q", id))
		}
		if !matched {
			writeError(w, r, newBadRequestError(errors.Errorf("Deployment id should respect the following format: %q", YorcDeploymentIDPattern)))
			return
		}
		// Do not impose a max id length as it doesn't have a concrete impact for now
		// if len(id) > YorcDeploymentIDMaxLength {
		// 	writeError(w, r, newBadRequestError(errors.Errorf("Deployment id should be less than %d characters (actual size %d)", YorcDeploymentIDMaxLength, len(id))))
		// 	return
		// }
		dExits, err := deployments.DoesDeploymentExists(ctx, id)
		if err != nil {
			log.Panicf("%v", err)
		}
		if dExits {
			mess := fmt.Sprintf("Deployment with id %q already exists", id)
			log.Debugf("[ERROR]: %s", mess)
			writeError(w, r, newConflictRequest(mess))
			return
		}
		uid = id
	} else {
		uid = fmt.Sprint(uuid.NewV4())
	}
	log.Printf("Analyzing deployment %s\n", uid)

	yamlFile, archiveErr := unzipArchiveGetTopology(s.config.WorkingDirectory, uid, r)
	if archiveErr != nil {
		log.Printf("Error analyzing archive for deployment %s\n", uid)
		writeError(w, r, archiveErr)
		return
	}

	// TODO(loicalbertin) why do we create a new context here?
	// I was expecting to use the one from http.Request
	// To be checked if there is a good reason for this.
	ctx := context.Background()
	err := deployments.CleanupPurgedDeployments(ctx, s.consulClient, s.config.PurgedDeploymentsEvictionTimeout, uid)
	if err != nil {
		log.Panicf("%v", err)
	}

	if !checkBlockingOperationOnDeployment(ctx, uid, w, r) {
		return
	}

	if err := deployments.AddBlockingOperationOnDeploymentFlag(r.Context(), uid); err != nil {
		log.Debugf("ERROR: %+v", err)
		log.Panic(err)
	}
	defer deployments.RemoveBlockingOperationOnDeploymentFlag(ctx, uid)

	if err := deployments.StoreDeploymentDefinition(r.Context(), uid, yamlFile); err != nil {
		log.Debugf("ERROR: %+v", err)
		log.Panic(err)
	}
	data := map[string]string{
		"workflowName": "install",
	}
	taskID, err := s.tasksCollector.RegisterTaskWithData(uid, tasks.TaskTypeDeploy, data)
	if err != nil {
		if ok, _ := tasks.IsAnotherLivingTaskAlreadyExistsError(err); ok {
			writeError(w, r, newBadRequestError(err))
			return
		}
		log.Panic(err)
	}

	w.Header().Set("Location", fmt.Sprintf("/deployments/%s/tasks/%s", uid, taskID))
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) updateDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")
	id, err := url.QueryUnescape(id)
	if err != nil {
		log.Panicf("%v", errors.Wrapf(err, "Failed to unescape given deployment id %q", id))
	}
	dExits, err := deployments.DoesDeploymentExists(ctx, id)
	if err != nil {
		log.Panicf("%v", err)
	}
	if !dExits {
		writeError(w, r, errNotFound)
		return
	}

	if !checkBlockingOperationOnDeployment(ctx, id, w, r) {
		return
	}

	// Update is a blocking operation
	if err := deployments.AddBlockingOperationOnDeploymentFlag(r.Context(), id); err != nil {
		log.Debugf("ERROR: %+v", err)
		log.Panic(err)
	}
	defer deployments.RemoveBlockingOperationOnDeploymentFlag(ctx, id)

	s.updateDeployment(w, r, id)
}

func (s *Server) deleteDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")

	dExits, err := deployments.DoesDeploymentExists(ctx, id)
	if err != nil {
		log.Panicf("%v", err)
	}
	if !dExits {
		writeError(w, r, errNotFound)
		return
	}

	if !checkBlockingOperationOnDeployment(ctx, id, w, r) {
		return
	}

	purge, err := getBoolQueryParam(r, "purge")
	if err != nil {
		writeError(w, r, newBadRequestMessage("purge query parameter must be a boolean value"))
		return
	}
	var taskType tasks.TaskType
	if purge {
		log.Debugf("A purge task on deployment:%s has been requested", id)
		taskType = tasks.TaskTypePurge
	} else {
		taskType = tasks.TaskTypeUnDeploy
	}

	if taskType == tasks.TaskTypeUnDeploy {
		status, err := deployments.GetDeploymentStatus(ctx, id)
		if err != nil {
			log.Panicf("%v", err)
		}
		if status == deployments.UNDEPLOYED {
			writeError(w, r, newBadRequestMessage("Deployment already undeployed"))
			return
		}
	}
	data := map[string]string{
		"workflowName": "uninstall",
	}
	// Default is not to stop on error for undeployment
	stopOnError, err := getBoolQueryParam(r, "stopOnError")
	if err != nil {
		writeError(w, r, newBadRequestMessage("stopOnError query parameter must be a boolean value"))
		return
	}
	data["continueOnError"] = strconv.FormatBool(!stopOnError)
	if taskID, err := s.tasksCollector.RegisterTaskWithData(id, taskType, data); err != nil {
		log.Debugf("register task has returned an err:%q", err.Error())
		if ok, _ := tasks.IsAnotherLivingTaskAlreadyExistsError(err); ok {
			log.Debugln("another task is living")
			writeError(w, r, newBadRequestError(err))
			return
		}

		// Inconsistent deployment: force purge enters in action
		if ok := deployments.IsInconsistentDeploymentError(err); ok {
			log.Debugf("inconsistent deployment with ID:%q. We force purge it.", id)
			newTaskID, err := s.tasksCollector.RegisterTask(id, tasks.TaskTypeForcePurge)
			if err != nil {
				log.Printf("Failed to force purge deployment with ID:%q due to error:%+v", id, err)
				writeError(w, r, newInternalServerError(err))
				return
			}
			w.Header().Set("Location", fmt.Sprintf("/deployments/%s/tasks/%s", id, newTaskID))
			w.WriteHeader(http.StatusAccepted)
			return
		}
		log.Panic(err)
	} else {
		w.Header().Set("Location", fmt.Sprintf("/deployments/%s/tasks/%s", id, taskID))
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) getDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")

	status, err := deployments.GetDeploymentStatus(ctx, id)
	if err != nil {
		if deployments.IsDeploymentNotFoundError(err) {
			writeError(w, r, errNotFound)
			return
		}
		log.Panic(err)
	}

	deployment := Deployment{ID: id, Status: status.String()}
	links := []AtomLink{newAtomLink(LinkRelSelf, r.URL.Path)}
	nodes, err := deployments.GetNodes(ctx, id)
	if err != nil {
		log.Panic(err)
	}
	for _, node := range nodes {
		links = append(links, newAtomLink(LinkRelNode, path.Join(r.URL.Path, "nodes", node)))
	}

	tasksList, err := deployments.GetDeploymentTaskList(ctx, id)
	if err != nil {
		log.Panic(err)
	}
	for _, task := range tasksList {
		links = append(links, newAtomLink(LinkRelTask, path.Join(r.URL.Path, "tasks", task)))
	}

	links = append(links, s.listOutputsLinks(ctx, id)...)

	deployment.Links = links
	encodeJSONResponse(w, r, deployment)
}

func (s *Server) listDeploymentsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	deploymentsIDs, err := deployments.GetDeploymentsIDs(ctx)
	if err != nil {
		log.Panic(err)
	}
	if len(deploymentsIDs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	deps := make([]Deployment, 0)
	for _, deploymentID := range deploymentsIDs {
		status, err := deployments.GetDeploymentStatus(ctx, deploymentID)
		if err != nil {
			if deployments.IsDeploymentNotFoundError(err) {
				// Deployment is not found : we force rapport error and ignore it
				log.Printf("[WARNING] deployment %q is inconsistent, ignoring it from deployments list. Please investigate and report this issue.", deploymentID)
				continue
			}
			log.Panic(err)
		}
		deps = append(deps, Deployment{
			ID:     deploymentID,
			Status: status.String(),
			Links:  []AtomLink{newAtomLink(LinkRelDeployment, "/deployments/"+deploymentID)},
		})
	}
	if len(deps) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	encodeJSONResponse(w, r, DeploymentsCollection{Deployments: deps})
}

func (s *Server) purgeDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	deploymentID := params.ByName("id")

	errs := new(Errors)
	forcePurge, err := getBoolQueryParam(r, "force")
	if err != nil {
		writeError(w, r, newBadRequestMessage("force query parameter must be a boolean value"))
		return
	}
	err = operations.PurgeDeployment(ctx, deploymentID, s.config.WorkingDirectory, forcePurge, true)
	if err != nil {
		log.Printf("purge error for deployment: %q\n%v", deploymentID, err)
		if merr, ok := err.(*multierror.Error); ok {
			for _, err := range merr.Errors {
				errs.Errors = append(errs.Errors, &Error{"internal_server_error", http.StatusInternalServerError, "Internal Server Error", err.Error()})
			}
		} else {
			errs.Errors = append(errs.Errors, newInternalServerError(err))
		}

	}
	w.Header().Set("Content-Type", mimeTypeApplicationJSON)
	if len(errs.Errors) > 0 {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	encodeJSONResponse(w, r, errs)
}
