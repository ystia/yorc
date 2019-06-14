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
	"time"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/prov/terraform"
)

type defaultExecutor struct {
	generator *googleGenerator
}

func (e *defaultExecutor) ExecOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) error {
	log.Debugf("google defaultExecutor: Execute the operation:%+v", operation)
	var delegateOp string
	switch operation.Name {
	case "standard.create":
		delegateOp = "install"
	case "standard.delete":
		delegateOp = "uninstall"
	default:
		return errors.Errorf("Unsupported operation %q", operation.Name)

	}

	delegate := terraform.NewExecutor(e.generator, nil)
	return delegate.ExecDelegate(ctx, conf, taskID, deploymentID, nodeName, delegateOp)
}

func (e *defaultExecutor) ExecAsyncOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation, stepName string) (*prov.Action, time.Duration, error) {
	return nil, 0, errors.New("asynchronous operation is not yet handled by this executor")
}
