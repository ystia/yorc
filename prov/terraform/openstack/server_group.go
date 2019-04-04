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

package openstack

import (
	"context"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/deployments"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/prov/terraform/commons"
	"path"
)

func (g *osGenerator) generateServerGroup(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, infrastructure *commons.Infrastructure, outputs map[string]string, env *[]string) error {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if nodeType != "yorc.nodes.openstack.ServerGroup" {
		return errors.Errorf("Unsupported node type for %q: %s", nodeName, nodeType)
	}

	serverGroup := &ServerGroup{}
	serverGroup.Name, err = deployments.GetStringNodeProperty(kv, deploymentID, nodeName, "name", true)
	if err != nil {
		return err
	}
	policy, err := deployments.GetStringNodeProperty(kv, deploymentID, nodeName, "policy", true)
	if err != nil {
		return err
	}
	serverGroup.Policies = []string{policy}
	commons.AddResource(infrastructure, "openstack_compute_servergroup_v2", serverGroup.Name, serverGroup)

	// Provide output for server group ID
	idKey := nodeName + "-id"
	commons.AddOutput(infrastructure, idKey, &commons.Output{Value: fmt.Sprintf("${openstack_compute_servergroup_v2.%s.id}", serverGroup.Name)})
	outputs[path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName, "/0/attributes/id")] = idKey
	return nil
}
