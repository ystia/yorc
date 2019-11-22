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
	"fmt"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"

	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/registry"
	"github.com/ystia/yorc/v4/tasks"
)

func (s *Server) postInfraUsageHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	infraName := params.ByName("infraName")
	locationName := params.ByName("locationName")
	log.Debugf("Posting query for getting infra usage information with infra:%q, location:%q", infraName, locationName)

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

	data := make(map[string]string)
	data["locationName"] = locationName

	// Add as well query parameters to task data
	values := r.URL.Query()
	for k, v := range values {
		data[k] = strings.Join(v, ",")
	}
	taskID, err := s.tasksCollector.RegisterTaskWithData(targetID, tasks.TaskTypeQuery, data)
	if err != nil {
		log.Panic(err)
	}

	w.Header().Set("Location", fmt.Sprintf("/infra_usage/%s/%s/tasks/%s", infraName, locationName, taskID))
	w.WriteHeader(http.StatusAccepted)
}
