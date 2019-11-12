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
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

// storeNodes stores topology nodes
func storeNodes(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix, importPath, rootDefPath string) {
	nodesPrefix := path.Join(topologyPrefix, "nodes")
	for nodeName, node := range topology.TopologyTemplate.NodeTemplates {
		nodePrefix := nodesPrefix + "/" + nodeName
		//consulStore.StoreConsulKeyAsString(nodePrefix+"/type", node.Type)
		storage.GetStore(types.StoreTypeDeployment).Set(nodePrefix+"/type", node.Type)
		if node.Directives != nil {
			storage.GetStore(types.StoreTypeDeployment).Set(path.Join(nodePrefix, "directives"), node.Directives)
			//consulStore.StoreConsulKeyAsString(
			//	path.Join(nodePrefix, "directives"),
			//	strings.Join(node.Directives, ","))
		}
		storeMapValueAssignment(consulStore, path.Join(nodePrefix, "properties"), node.Properties)
		storeMapValueAssignment(consulStore, path.Join(nodePrefix, "attributes"), node.Attributes)
		capabilitiesPrefix := nodePrefix + "/capabilities"
		for capName, capability := range node.Capabilities {
			capabilityPrefix := capabilitiesPrefix + "/" + capName
			storeMapValueAssignment(consulStore, path.Join(capabilityPrefix, "properties"), capability.Properties)
			storeMapValueAssignment(consulStore, path.Join(capabilityPrefix, "attributes"), capability.Attributes)
		}
		requirementsPrefix := nodePrefix + "/requirements"
		for reqIndex, reqValueMap := range node.Requirements {
			for reqName, reqValue := range reqValueMap {
				reqPrefix := requirementsPrefix + "/" + strconv.Itoa(reqIndex)
				StoreRequirementAssignment(consulStore, reqValue, reqPrefix, reqName)
			}
		}
		artifactsPrefix := nodePrefix + "/artifacts"
		for artName, artDef := range node.Artifacts {
			artFile := filepath.Join(rootDefPath, filepath.FromSlash(path.Join(importPath, artDef.File)))
			log.Debugf("Looking if artifact %q exists on filesystem", artFile)
			if _, err := os.Stat(artFile); os.IsNotExist(err) {
				log.Printf("Warning: Artifact %q for node %q with computed path %q doesn't exists on filesystem, ignoring it.", artName, nodeName, artFile)
				continue
			}
			storeArtifact(consulStore, artDef, path.Join(artifactsPrefix, artName))
		}

		metadataPrefix := nodePrefix + "/metadata/"
		//for metaName, metaValue := range node.Metadata {
		//	consulStore.StoreConsulKeyAsString(metadataPrefix+metaName, metaValue)
		//}

		if len(node.Metadata) > 0 {
			storage.GetStore(types.StoreTypeDeployment).Set(metadataPrefix, node.Metadata)
		}

		storeInterfaces(consulStore, node.Interfaces, nodePrefix, false)
	}

}
