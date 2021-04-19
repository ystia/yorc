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

package maas

import (
	"context"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
)

func generateNodeAllocation(ctx context.Context, locationProps config.DynamicMap, deploymentID string, nodeName, instanceName string, infra *infrastructure) error {
	nodeType, err := deployments.GetNodeType(ctx, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if nodeType != "yorc.nodes.maas.Compute" {
		return errors.Errorf("Unsupported node type for %q: %s", nodeName, nodeType)
	}

	node := &nodeAllocation{instanceName: instanceName}

	infra.nodes = append(infra.nodes, node)
	return nil
}
