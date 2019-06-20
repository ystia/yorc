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

package internal

import (
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/tosca"
)

func storeArtifacts(consulStore consulutil.ConsulStore, artifacts tosca.ArtifactDefMap, prefix string) {
	for artName, artDef := range artifacts {
		artPrefix := prefix + "/" + artName
		storeArtifact(consulStore, artDef, artPrefix)
	}
}
func storeArtifact(consulStore consulutil.ConsulStore, artifact tosca.ArtifactDefinition, artifactPrefix string) {
	consulStore.StoreConsulKeyAsString(artifactPrefix+"/file", artifact.File)
	consulStore.StoreConsulKeyAsString(artifactPrefix+"/type", artifact.Type)
	consulStore.StoreConsulKeyAsString(artifactPrefix+"/repository", artifact.Repository)
	consulStore.StoreConsulKeyAsString(artifactPrefix+"/deploy_path", artifact.DeployPath)
}
