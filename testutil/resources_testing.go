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

package testutil

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"
	res "gopkg.in/cookieo9/resources-go.v2"

	"github.com/ystia/yorc/v4/deployments/store"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/resources"
)

func storeCommonDefinitions() {
	resources, err := getToscaResources()
	if err != nil {
		log.Panicf("Failed to load builtin Tosca definition. %v", err)
	}
	ctx := context.Background()
	for defName, defContent := range resources {
		store.CommonDefinition(ctx, defName, store.BuiltinOrigin, defContent)
	}
}

func getToscaResources() (map[string][]byte, error) {
	_, pFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil, errors.New("can't find local TOSCA resources, can't determine package path")
	}
	toscaDirPath := filepath.Clean(filepath.Join(filepath.Dir(pFile), "..", "data", "tosca"))
	if _, err := os.Stat(toscaDirPath); err != nil && os.IsNotExist(err) {
		return nil, errors.Errorf("can't find local TOSCA resources, %s does not exit", toscaDirPath)
	}
	bundle := res.OpenFS(toscaDirPath)
	defer bundle.Close()

	searcher := bundle.(res.Searcher)
	toscaResources, err := searcher.Glob("*.yml")
	if err != nil {
		return nil, errors.Wrapf(err, "can't find local TOSCA resources in %s", toscaDirPath)
	}
	return resources.ReadResources(toscaResources)
}
