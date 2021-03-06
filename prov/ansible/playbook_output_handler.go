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

package ansible

import (
	"context"
	"os/exec"
)

// Handler implementing the interface outputHandler.
// This handler logs ansible playbook outputs.
// It is started before the playbook is run,
// and stopped once the playbook was run.
type playbookOutputHandler struct {
	execution *executionAnsible
	context   context.Context
}

// start - starts handling the output for the command in argument
func (h *playbookOutputHandler) start(cmd *exec.Cmd) error {

	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	go logAnsibleOutputInConsul(h.context, h.execution.deploymentID,
		h.execution.NodeName, h.execution.hosts, cmdReader)
	return nil
}

// stop - stops handling the output of a command
func (h *playbookOutputHandler) stop() error {
	return nil
}
