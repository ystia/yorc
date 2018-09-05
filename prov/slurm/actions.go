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
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/helper/sshutil"
	"github.com/ystia/yorc/prov"
)

type actionOperator struct {
	client *sshutil.SSHClient
}

func (o actionOperator) ExecAction(ctx context.Context, cfg config.Configuration, action prov.Action, deploymentID string) error {
	var err error
	o.client, err = GetSSHClient(cfg)
	if err != nil {
		return err
	}
	switch action.ActionType {
	case "job-monitoring":
		return o.monitorJob()
	default:
		return errors.Errorf("Unsupported actionType %q", action.ActionType)
	}
	return nil
}

func (o actionOperator) monitorJob() error {
	return nil
}
