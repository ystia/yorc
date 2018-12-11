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

package commons

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/log"
)

// FileOutputPrefix is the prefix to identify file output
const FileOutputPrefix = "file:"

// PostApplyCallback is a callback function called after Terraform apply execution
type PostApplyCallback func()

// A Generator is used to generate the Terraform infrastructure for a given TOSCA node
type Generator interface {
	// GenerateTerraformInfraForNode generates the Terraform infrastructure file for the given node.
	// It returns 'true' if a file was generated and 'false' otherwise (in case of a infrastructure component
	// already exists for this node and should just be reused).
	// GenerateTerraformInfraForNode can also return a map of outputs names indexed by consul keys into which the outputs results should be stored.
	// And a list of environment variables in form "key=value" to be added to the current process environment when running terraform commands.
	// This is particularly useful for adding secrets that should not be in tf files.
	// It returns too a callback function allowing execution some cleanup stuff once the infrastructure has been applied
	GenerateTerraformInfraForNode(ctx context.Context, cfg config.Configuration, deploymentID, nodeName, infrastructurePath string) (bool, map[string]string, []string, PostApplyCallback, error)
}

// PreDestroyInfraCallback is a function that is call before destroying an infrastructure. If it returns false the node will not be destroyed.
type PreDestroyInfraCallback func(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName, infrastructurePath string) (bool, error)

// PreDestroyStorageInfraCallback is a callback of type PreDestroyInfraCallback
// checking if a block storage node is deletable on undeployment.
func PreDestroyStorageInfraCallback(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName, infrastructurePath string) (bool, error) {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return false, err
	}
	isBlockStorage, err := deployments.IsTypeDerivedFrom(kv, deploymentID, nodeType, "tosca.nodes.BlockStorage")
	if err != nil {
		return false, err
	}

	if isBlockStorage {

		deletable, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "deletable")
		if err != nil {
			return false, err
		}
		if deletable == nil || strings.ToLower(deletable.RawString()) != "true" {
			// False by default
			msg := fmt.Sprintf("Node %q is a BlockStorage without the property 'deletable', so not destroyed on undeployment...", nodeName)
			log.Debug(msg)
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(msg)
			return false, nil
		}
	}
	return true, nil
}
