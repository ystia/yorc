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

package internal

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ystia/yorc/v4/helper/consulutil"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/tosca"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"
)

// storeImports parses and store imports.
func storeImports(ctx context.Context, consulStore consulutil.ConsulStore, errGroup *errgroup.Group, topology tosca.Topology, deploymentID, topologyPrefix, importPath, rootDefPath string) error {
	for _, element := range topology.Imports {

		importURI := strings.Trim(element.File, " \t")
		importedTopology := tosca.Topology{}

		if strings.HasPrefix(importURI, "<") && strings.HasSuffix(importURI, ">") {
			// Internal import
		} else {
			uploadFile := filepath.Join(rootDefPath, filepath.FromSlash(importPath), filepath.FromSlash(importURI))

			definition, err := os.Open(uploadFile)
			if err != nil {
				return errors.Errorf("Failed to parse internal definition %s: %v", importURI, err)
			}

			defBytes, err := ioutil.ReadAll(definition)
			if err != nil {
				return errors.Errorf("Failed to parse internal definition %s: %v", importURI, err)
			}

			if err = yaml.Unmarshal(defBytes, &importedTopology); err != nil {
				return errors.Errorf("Failed to parse internal definition %s: %v", importURI, err)
			}

			errGroup.Go(func() error {
				// Using flat keys under imports, for convenience when searching
				// for an import containing a given metadata template name
				// (function getSubstitutableNodeType in this package)
				importPrefix := path.Join("imports", strings.Replace(path.Join(importPath, importURI), "/", "_", -1))
				return StoreTopology(ctx, consulStore, errGroup, importedTopology, deploymentID, topologyPrefix, importPrefix, path.Dir(path.Join(importPath, importURI)), rootDefPath)
			})
		}
	}
	return nil
}
