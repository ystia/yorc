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

package google

import (
	"context"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/prov/terraform/commons"
)

func (g *googleGenerator) generateComputeAddress(ctx context.Context, kv *api.KV,
	cfg config.Configuration, deploymentID, nodeName, instanceName string, instanceID int,
	infrastructure *commons.Infrastructure,
	outputs map[string]string) error {

	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if nodeType != "yorc.nodes.google.StaticIP" {
		return errors.Errorf("Unsupported node type for %q: %s", nodeName, nodeType)
	}

	return nil
}
