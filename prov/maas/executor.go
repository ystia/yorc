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
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/locations"
)

const infrastructureType = "maas"

type defaultExecutor struct {
}

func (e *defaultExecutor) ExecDelegate(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {

	var locationProps config.DynamicMap
	locationMgr, err := locations.GetManager(cfg)
	if err == nil {
		locationProps, err = locationMgr.GetLocationPropertiesForNode(ctx, deploymentID, nodeName, infrastructureType)
	}
	if err != nil {
		return err
	}

	tmp := locationProps.GetString("api_token")
	fmt.Println("HERE :: " + tmp)

	operation := strings.ToLower(delegateOperation)
	switch {
	case operation == "install":
		err = e.installNode(ctx, cfg, locationProps, deploymentID, nodeName, operation)
	// case operation == "uninstall":
	// 	err = e.uninstallNode(ctx, cfg, locationProps, deploymentID, nodeName, instances, operation)
	default:
		return errors.Errorf("Unsupported operation %q", delegateOperation)
	}
	return err
}

func (e *defaultExecutor) installNode(ctx context.Context, cfg config.Configuration, locationProps config.DynamicMap, deploymentID, nodeName string, operation string) error {
	// infra, err := generateInfrastructure(ctx, locationProps, deploymentID, nodeName, operation)
	// if err != nil {
	// 	return err
	// }

	// return e.createInfrastructure(ctx, cfg, locationProps, deploymentID, nodeName, infra)
	return nil
}
