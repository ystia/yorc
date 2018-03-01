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
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov/terraform"
	"github.com/ystia/yorc/registry"
)

func init() {
	reg := registry.GetRegistry()
	reg.RegisterDelegates([]string{`yorc\.nodes\.openstack\..*`}, terraform.NewExecutor(&osGenerator{}, preDestroyInfraCallback), registry.BuiltinOrigin)
}

func preDestroyInfraCallback(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, logOptFields events.LogOptionalFields) (bool, error) {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return false, err
	}
	// TODO  consider making this generic: references to OpenStack should not be found here.
	if nodeType == "yorc.nodes.openstack.BlockStorage" {
		var deletable string
		var found bool
		found, deletable, err = deployments.GetNodeProperty(kv, deploymentID, nodeName, "deletable")
		if err != nil {
			return false, err
		}
		if !found || strings.ToLower(deletable) != "true" {
			// False by default
			msg := fmt.Sprintf("Node %q is a BlockStorage without the property 'deletable' do not destroy it...", nodeName)
			log.Debug(msg)
			events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString(msg)
			return false, nil
		}
	}
	return true, nil
}
