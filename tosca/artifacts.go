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

package tosca

import (
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/log"
)

// ArtifactDefMap is a map of ArtifactDefinition
type ArtifactDefMap map[string]ArtifactDefinition

// UnmarshalYAML unmarshals a yaml into an ArtifactDefMap
func (adm *ArtifactDefMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	log.Debugf("Resolving in artifacts in standard TOSCA format")
	// Either a map or a seq
	*adm = make(ArtifactDefMap)
	var m map[string]ArtifactDefinition
	if err := unmarshal(&m); err == nil {
		for k, v := range m {
			(*adm)[k] = v
		}
		return nil
	}

	log.Debugf("Resolving in artifacts in Alien format 1.2")
	//var l []map[string]interface{}
	var l []ArtifactDefinition
	if err := unmarshal(&l); err == nil {

		log.Debugf("list: %v", l)
		for _, a := range l {
			(*adm)[a.name] = a
		}
		return nil
	}

	log.Debugf("Resolving in artifacts in Alien format 1.3")
	var lmap []ArtifactDefMap
	if err := unmarshal(&lmap); err != nil {
		return err
	}
	for _, m := range lmap {
		for k, v := range m {
			(*adm)[k] = v
		}
	}
	return nil
}

// An ArtifactDefinition is the representation of a TOSCA Artifact Definition
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ENTITY_ARTIFACT_DEF for more details
type ArtifactDefinition struct {
	Type        string `yaml:"type,omitempty"`
	File        string `yaml:"file,omitempty"`
	Description string `yaml:"description,omitempty"`
	Repository  string `yaml:"repository,omitempty"`
	DeployPath  string `yaml:"deploy_path,omitempty"`
	// Extra types used in list (A4C) mode
	name string
}

// UnmarshalYAML unmarshals a yaml into an ArtifactDefinition
func (a *ArtifactDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err == nil {
		a.File = s
		return nil
	}
	var str struct {
		Type        string `yaml:"type"`
		File        string `yaml:"file"`
		Description string `yaml:"description,omitempty"`
		Repository  string `yaml:"repository,omitempty"`
		DeployPath  string `yaml:"deploy_path,omitempty"`

		// Extra types
		MimeType string                 `yaml:"mime_type,omitempty"`
		XXX      map[string]interface{} `yaml:",inline"`
	}
	if err := unmarshal(&str); err != nil {
		return err
	}
	log.Debugf("Unmarshalled complex ArtifactDefinition %+v", str)
	a.Type = str.Type
	a.File = str.File
	a.Description = str.Description
	a.Repository = str.Repository
	a.DeployPath = str.DeployPath
	if str.File == "" && len(str.XXX) == 1 {
		for k, v := range str.XXX {
			a.name = k
			var ok bool
			a.File, ok = v.(string)
			if !ok {
				return errors.New("Missing mandatory attribute \"file\" for artifact")
			}
		}
	}

	if a.File == "" {
		return errors.New("Missing mandatory attribute \"file\" for artifact")
	}
	return nil
}
