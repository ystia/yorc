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

import "github.com/ystia/yorc/registry"

const (
	implementationArtifactBash         = "tosca.artifacts.Implementation.Bash"
	implementationArtifactPython       = "tosca.artifacts.Implementation.Python"
	implementationArtifactAnsible      = "tosca.artifacts.Implementation.Ansible"
	implementationArtifactAnsibleAlien = "org.alien4cloud.artifacts.AnsiblePlaybook"
)

func init() {
	reg := registry.GetRegistry()
	executor := newExecutor()
	reg.RegisterOperationExecutor(
		[]string{
			implementationArtifactBash,
			implementationArtifactPython,
			implementationArtifactAnsible,
			implementationArtifactAnsibleAlien,
		}, executor, registry.BuiltinOrigin)
	reg.RegisterActionOperator([]string{"ansible-job-monitoring"}, &actionOperator{executor: executor}, registry.BuiltinOrigin)
}
