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
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"regexp"

	"net/url"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tasks"
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
		dExits, err := deployments.DoesDeploymentExists(s.consulClient.KV(), id)
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

	var err error
	var file *os.File
	uploadPath := filepath.Join(s.config.WorkingDirectory, "deployments", uid)
	if err = os.MkdirAll(uploadPath, 0775); err != nil {
		log.Panicf("%+v", err)
	}

	file, err = os.Create(fmt.Sprintf("%s/deployment.zip", uploadPath))
	// check err
	if err != nil {
		log.Panicf("%+v", err)
	}

	_, err = io.Copy(file, r.Body)
	if err != nil {
		log.Panicf("%+v", err)
	}
	destDir := filepath.Join(uploadPath, "overlay")
	if err = os.MkdirAll(destDir, 0775); err != nil {
		log.Panicf("%+v", err)
	}
	zipReader, err := zip.OpenReader(file.Name())
	if err != nil {
		log.Panicf("%+v", err)
	}
	defer zipReader.Close()

	// Iterate through the files in the archive,
	// and extract them.
	// TODO: USe go routines to process files concurrently
	for _, f := range zipReader.File {
		fPath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			// Ensure that we have full rights on directory to be able to extract files into them
			if err = os.MkdirAll(fPath, f.Mode()|0700); err != nil {
				log.Panicf("%+v", err)
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
		if yamls, err := filepath.Glob(filepath.Join(destDir, pattern.pattern)); err != nil {
			log.Panicf("%+v", err)
		} else {
			yamlList = append(yamlList, yamls...)
		}
	}
	if len(yamlList) != 1 {
		log.Panic("One and only one YAML (.yml or .yaml) file should be present at the root of deployment archive")
	}

	if err := deployments.StoreDeploymentDefinition(r.Context(), s.consulClient.KV(), uid, yamlList[0]); err != nil {
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

func (s *Server) deleteDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	id := params.ByName("id")

	dExits, err := deployments.DoesDeploymentExists(s.consulClient.KV(), id)
	if err != nil {
		log.Panicf("%v", err)
	}
	if !dExits {
		writeError(w, r, errNotFound)
		return
	}

	var taskType tasks.TaskType
	if _, ok := r.URL.Query()["purge"]; ok {
		log.Debugf("A purge task on deployment:%s has been requested", id)
		taskType = tasks.TaskTypePurge
	} else {
		taskType = tasks.TaskTypeUnDeploy
	}

	if taskType == tasks.TaskTypeUnDeploy {
		status, err := deployments.GetDeploymentStatus(s.consulClient.KV(), id)
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
	if taskID, err := s.tasksCollector.RegisterTaskWithData(id, taskType, data); err != nil {
		log.Debugln("register task err" + err.Error())
		if ok, _ := tasks.IsAnotherLivingTaskAlreadyExistsError(err); ok {
			log.Debugln("another task is living")
			writeError(w, r, newBadRequestError(err))
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

	kv := s.consulClient.KV()
	status, err := deployments.GetDeploymentStatus(kv, id)
	if err != nil {
		if deployments.IsDeploymentNotFoundError(err) {
			writeError(w, r, errNotFound)
			return
		}
		log.Panic(err)
	}

	deployment := Deployment{ID: id, Status: status.String()}
	links := []AtomLink{newAtomLink(LinkRelSelf, r.URL.Path)}
	nodes, err := deployments.GetNodes(kv, id)
	if err != nil {
		log.Panic(err)
	}
	for _, node := range nodes {
		links = append(links, newAtomLink(LinkRelNode, path.Join(r.URL.Path, "nodes", node)))
	}

	tasksList, err := tasks.GetTasksIdsForTarget(kv, id)
	if err != nil {
		log.Panic(err)
	}
	for _, task := range tasksList {
		links = append(links, newAtomLink(LinkRelTask, path.Join(r.URL.Path, "tasks", task)))
	}

	links = append(links, s.listOutputsLinks(id)...)

	deployment.Links = links
	encodeJSONResponse(w, r, deployment)
}

func (s *Server) listDeploymentsHandler(w http.ResponseWriter, r *http.Request) {
	kv := s.consulClient.KV()
	depPaths, _, err := kv.Keys(consulutil.DeploymentKVPrefix+"/", "/", nil)
	if err != nil {
		log.Panic(err)
	}
	if len(depPaths) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	deps := make([]Deployment, 0)
	depPrefix := consulutil.DeploymentKVPrefix + "/"
	for _, depPath := range depPaths {
		deploymentID := strings.TrimRight(strings.TrimPrefix(depPath, depPrefix), "/ ")
		status, err := deployments.GetDeploymentStatus(kv, deploymentID)
		if err != nil {
			if deployments.IsDeploymentNotFoundError(err) {
				// Deployment is not found : we force deletion and ignore it
				go func() {
					log.Debugf("Force purge inconsistent deployment with ID:%q", deploymentID)
					if _, err := s.tasksCollector.RegisterTask(deploymentID, tasks.TaskTypeForcePurge); err != nil {
						log.Printf("Failed to force purge deployment with ID:%q due to error:%+v", deploymentID, err)
					}
				}()
				continue
			} else {
				log.Panic(err)
			}
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
