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
	"time"

	"github.com/ystia/yorc/v4/tosca/types"
)

type infrastructure struct {
	nodes []*nodeAllocation
}

type nodeAllocation struct {
	cpu          string
	memory       string
	gres         string
	constraint   string
	partition    string
	jobName      string
	credentials  *types.Credential
	instanceName string
	account      string
	reservation  string
}

type jobInfo struct {
	ID                     string                      `json:"id,omitempty"`
	Name                   string                      `json:"name,omitempty"`
	Tasks                  int                         `json:"tasks,omitempty"`
	Cpus                   int                         `json:"cpus,omitempty"`
	Nodes                  int                         `json:"nodes,omitempty"`
	Mem                    string                      `json:"mem,omitempty"`
	MaxTime                string                      `json:"max_time,omitempty"`
	Opts                   []string                    `json:"opts,omitempty"`
	ExecutionOptions       types.SlurmExecutionOptions `json:"execution_options,omitempty"`
	Inputs                 map[string]string           `json:"inputs,omitempty"`
	MonitoringTimeInterval time.Duration               `json:"monitoring_time_interval,omitempty"`
	Account                string                      `json:"account,omitempty"`
	Reservation            string                      `json:"reservation,omitempty"`
	WorkingDir             string                      `json:"working_directory,omitempty"`
	Artifacts              []string                    `json:"artifacts,omitempty"`
	EnvFile                string                      `json:"env_file,omitempty"`
}
