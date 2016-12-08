package rest

import (
	"archive/zip"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/satori/go.uuid"
	"io"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"os"
	"path"
	"path/filepath"
	"strings"
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

	uid := fmt.Sprint(uuid.NewV4())
	log.Printf("Analyzing deployment %s\n", uid)

	var err error
	var file *os.File
	uploadPath := filepath.Join("work", "deployments", uid)
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
		log.Fatal(err)
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
			for _, yamlFile := range yamls {
				yamlList = append(yamlList, yamlFile)
			}
		}
	}
	if len(yamlList) != 1 {
		log.Panic("One and only one YAML (.yml or .yaml) file should be present at the root of deployment archive")
	}

	if err := deployments.StoreDeploymentDefinition(r.Context(), s.consulClient.KV(), uid, yamlList[0]); err != nil {
		log.Debugf("ERROR: %+v", err)
		log.Panic(err)
	}

	if _, err := s.tasksCollector.RegisterTask(uid, tasks.Deploy); err != nil {
		if tasks.IsAnotherLivingTaskAlreadyExistsError(err) {
			WriteError(w, r, NewBadRequestError(err))
			return
		}
		log.Panic(err)
	}

	w.Header().Set("Location", fmt.Sprintf("/deployments/%s", uid))
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) deleteDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")

	var taskType tasks.TaskType
	if _, ok := r.URL.Query()["purge"]; ok {
		taskType = tasks.Purge
	} else {
		taskType = tasks.UnDeploy
	}

	if taskId, err := s.tasksCollector.RegisterTask(id, taskType); err != nil {
		if tasks.IsAnotherLivingTaskAlreadyExistsError(err) {
			WriteError(w, r, NewBadRequestError(err))
			return
		}
		log.Panic(err)
	} else {
		w.Header().Set("Location", fmt.Sprintf("/deployments/%s/tasks/%s", id, taskId))
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) getDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")

	kv := s.consulClient.KV()
	status, err := deployments.GetDeploymentStatus(kv, id)
	if err != nil {
		if deployments.IsDeploymentNotFoundError(err) {
			WriteError(w, r, ErrNotFound)
			return
		}
		log.Panic(err)
	}

	deployment := Deployment{Id: id, Status: status.String()}
	links := []AtomLink{newAtomLink(LINK_REL_SELF, r.URL.Path)}
	nodes, err := deployments.GetNodes(kv, id)
	if err != nil {
		log.Panic(err)
	}
	for _, node := range nodes {
		links = append(links, newAtomLink(LINK_REL_NODE, path.Join(r.URL.Path, "nodes", node)))
	}

	tasksList, err := tasks.GetTasksIdsForTarget(kv, id)
	if err != nil {
		log.Panic(err)
	}
	for _, task := range tasksList {
		links = append(links, newAtomLink(LINK_REL_TASK, path.Join(r.URL.Path, "tasks", task)))
	}

	links = append(links, s.listOutputsLinks(id)...)

	deployment.Links = links
	encodeJsonResponse(w, r, deployment)
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

	depCol := DeploymentsCollection{Deployments: make([]AtomLink, len(depPaths))}
	for depIndex, depPath := range depPaths {
		depId := strings.TrimRight(strings.TrimPrefix(depPath, consulutil.DeploymentKVPrefix), "/ ")
		link := newAtomLink(LINK_REL_DEPLOYMENT, "/deployments"+depId)
		depCol.Deployments[depIndex] = link
	}
	encodeJsonResponse(w, r, depCol)
}
