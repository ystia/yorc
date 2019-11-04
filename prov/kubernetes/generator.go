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

package kubernetes

import (
	"github.com/ystia/yorc/v4/config"
)

// A k8sGenerator is used to generate the Kubernetes objects for a given TOSCA node
type k8sGenerator struct {
	cfg config.Configuration
}

// newGenerator create a K8sGenerator
func newGenerator(cfg config.Configuration) *k8sGenerator {
	return &k8sGenerator{cfg: cfg}
}
