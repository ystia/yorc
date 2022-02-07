// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

//go:build testing
// +build testing

package bootstrap

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	res "gopkg.in/cookieo9/resources-go.v2"

	"github.com/ystia/yorc/v4/resources"
)

func getTOSCADefinition(name string) ([]byte, error) {
	dataDirPath := filepath.Clean(filepath.Join("..", "..", "data"))
	if _, err := os.Stat(dataDirPath); err != nil && os.IsNotExist(err) {
		return nil, errors.Errorf("can't find local data resources, %s does not exit", dataDirPath)
	}
	bundle := res.OpenFS(dataDirPath)
	defer bundle.Close()

	searcher := bundle.(res.Searcher)
	toscaRes, err := searcher.Find("tosca/" + name)
	if err != nil {
		return nil, errors.Wrapf(err, "can't find tosca/%s in %s", name, dataDirPath)
	}
	return resources.ReadResource(toscaRes)
}

func getResourcesZipPath() (string, error) {
	return filepath.Clean(filepath.Join("resources", "topology", "tosca_types.zip")), nil
}
