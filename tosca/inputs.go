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

import "github.com/ystia/yorc/v4/log"

// An Input is the representation of the input part of a TOSCA Operation Definition
//
// It could be either a Value Assignment or a Property Definition.
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ELEMENT_OPERATION_DEF for more details
type Input struct {
	ValueAssign *ValueAssignment    `json:"value_assignment,omitempty"`
	PropDef     *PropertyDefinition `json:"property_definition,omitempty"`
}

// UnmarshalYAML unmarshals a yaml into an Input
func (i *Input) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// an Input is either a Property definition or a ValueAssignment
	i.ValueAssign = nil
	i.PropDef = nil
	log.Debug("Try to parse input as a PropertyDefinition")
	p := new(PropertyDefinition)
	if err := unmarshal(p); err == nil && p.Type != "" {
		i.PropDef = p
		return nil
	}
	log.Debug("Try to parse input as a ValueAssignment")
	v := new(ValueAssignment)
	if err := unmarshal(v); err != nil {
		return err
	}
	i.ValueAssign = v
	return nil
}
