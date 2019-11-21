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
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"
	"os"
	"path"
	"path/filepath"
)

// storeNodes stores topology nodes
func storeNodes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath, rootDefPath string) error {
	nodesPrefix := path.Join(topologyPrefix, "nodes")
	for nodeName, node := range topology.TopologyTemplate.NodeTemplates {
		nodePrefix := nodesPrefix + "/" + nodeName

		for artName, artDef := range node.Artifacts {
			artFile := filepath.Join(rootDefPath, filepath.FromSlash(path.Join(importPath, artDef.File)))
			log.Debugf("Looking if artifact %q exists on filesystem", artFile)
			if _, err := os.Stat(artFile); os.IsNotExist(err) {
				log.Printf("Warning: Artifact %q for node %q with computed path %q doesn't exists on filesystem, ignoring it.", artName, nodeName, artFile)
				continue
			}
			delete(node.Artifacts, artName)
		}

		err := storage.GetStore(types.StoreTypeDeployment).Set(nodePrefix, node)
		if err != nil {
			return errors.Wrapf(err, "failed to store node with name:%q and value:%+v", nodeName, node)
		}
	}
	return nil
}
