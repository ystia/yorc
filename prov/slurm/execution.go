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

package slurm

import (
	"context"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
	"strings"
)

type execution interface {
	resolveExecution() error
	execute(ctx context.Context) error
}

type executionCommon struct {
	kv           *api.KV
	cfg          config.Configuration
	deploymentID string
	taskID       string
	NodeName     string
	operation    prov.Operation
	NodeType     string
}

func newExecution(kv *api.KV, cfg config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) (execution, error) {

	execCommon := &executionCommon{kv: kv,
		cfg:          cfg,
		deploymentID: deploymentID,
		NodeName:     nodeName,
		operation:    operation,
		taskID:       taskID,
	}

	return execCommon, execCommon.resolveOperation()
}

func (e *executionCommon) resolveOperation() error {
	return nil
}

func (e *executionCommon) resolveExecution() error {
	return nil
}

func (e *executionCommon) execute(ctx context.Context) (err error) {
	// Only runnable operation is currently supported
	log.Debugf("Execute the operation:%+v", e.operation)
	switch strings.ToLower(e.operation.Name) {
	case "tosca.interfaces.node.lifecycle.runnable.run":
		log.Printf("TODO Running the job: %s", e.operation.Name)
		// TODO exec the operation
		return nil
	default:
		return errors.Errorf("Unsupported operation %q", e.operation.Name)
	}
	return nil
}
