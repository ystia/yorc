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
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/log"
)

// RequirementDefinitionMap is a map of RequirementDefinition indexed by name
type RequirementDefinitionMap map[string]RequirementDefinition

// An RequirementDefinition is the representation of a TOSCA Requirement Definition
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ELEMENT_REQUIREMENT_DEF for more details
type RequirementDefinition struct {
	Capability   string `yaml:"capability"`
	Node         string `yaml:"node,omitempty"`
	Relationship string `yaml:"relationship,omitempty"`
	Occurrences  Range  `yaml:"occurrences,omitempty"`
	// Non Tosca-Standard A4C capability_name property
	CapabilityName string `yaml:"capability_name,omitempty"`
	// Extra types used in list (A4C) mode
	name string
}

// UnmarshalYAML unmarshals a yaml into an RequirementDefinitionMap
func (rdm *RequirementDefinitionMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Either requirement def (Alien mode) or a map
	*rdm = make(RequirementDefinitionMap)

	log.Debugf("Resolving in requirements in standard format")
	var m map[string]RequirementDefinition
	if err := unmarshal(&m); err == nil {
		log.Debugf("Length of requirement definition map: %d", len(m))
		if len(m) == 1 {
			for k, v := range m {
				(*rdm)[k] = v
			}
			return nil
		}
	}

	log.Debugf("Trying to Resolving in requirements in Alien format")
	var req RequirementDefinition
	if err := unmarshal(&req); err != nil {
		return err
	}
	(*rdm)[req.name] = req
	return nil
}

// UnmarshalYAML unmarshals a yaml into an RequirementDefinition
func (a *RequirementDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err == nil {
		a.Capability = s
		return nil
	}
	var str struct {
		Capability        string `yaml:"capability"`
		Node              string `yaml:"node,omitempty"`
		Relationship      string `yaml:"relationship,omitempty"`
		RelationshipAlien string `yaml:"relationship_type,omitempty"`
		Occurrences       Range  `yaml:"occurrences,omitempty"`
		CapabilityName    string `yaml:"capability_name,omitempty"`

		// Extra types for Alien Parsing
		LowerBound string                 `yaml:"lower_bound,omitempty"`
		UpperBound string                 `yaml:"upper_bound,omitempty"`
		XXX        map[string]interface{} `yaml:",inline"`
	}
	if err := unmarshal(&str); err != nil {
		return err
	}
	log.Debugf("Unmarshalled complex RequirementDefinition %#v", str)
	a.Capability = str.Capability
	a.Node = str.Node
	a.Relationship = str.Relationship
	a.Occurrences = str.Occurrences
	a.CapabilityName = str.CapabilityName
	if str.Capability == "" && len(str.XXX) == 1 {
		for k, v := range str.XXX {
			a.name = k
			a.Capability = fmt.Sprint(v)
		}
	}
	if a.Relationship == "" && str.RelationshipAlien != "" {
		a.Relationship = str.RelationshipAlien
	}
	if str.LowerBound != "" {
		bound, err := strconv.ParseUint(str.LowerBound, 10, 0)
		if err != nil {
			return errors.Errorf("Expecting a unsigned integer as lower bound got: %q", str.LowerBound)
		}
		a.Occurrences.LowerBound = bound
	}
	if str.UpperBound != "" {
		if bound, err := strconv.ParseUint(str.UpperBound, 10, 0); err != nil {
			if strings.ToUpper(str.UpperBound) != "UNBOUNDED" {
				return errors.Errorf("Expecting a unsigned integer or the 'UNBOUNDED' keyword as upper bound of the range got: %q", str.UpperBound)
			}
			a.Occurrences.UpperBound = UNBOUNDED
		} else {
			a.Occurrences.UpperBound = bound
		}
	}
	return nil
}

// RequirementAssignmentMap is a map of RequirementAssignment
type RequirementAssignmentMap map[string]RequirementAssignment

// An RequirementAssignment is the representation of a TOSCA Requirement Assignment
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ELEMENT_REQUIREMENT_ASSIGNMENT for more details
type RequirementAssignment struct {
	Capability        string `yaml:"capability"`
	Node              string `yaml:"node,omitempty"`
	Relationship      string `yaml:"relationship,omitempty"`
	RelationshipProps map[string]*ValueAssignment
	// Non Tosca-Standard A4C type_requirement property
	TypeRequirement string `yaml:"type_requirement,omitempty"`
}

// An RequirementRelationship is the representation of the relationship part of a TOSCA Requirement Assignment
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ELEMENT_REQUIREMENT_ASSIGNMENT for more details
type RequirementRelationship struct {
	Type       string                      `yaml:"type"`
	Properties map[string]*ValueAssignment `yaml:"properties,omitempty"`
}

// UnmarshalYAML unmarshals a yaml into an RequirementAssignment
func (r *RequirementAssignment) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var ras string
	if err := unmarshal(&ras); err == nil {
		r.Node = ras
		return nil
	}

	var ra struct {
		Capability      string                      `yaml:"capability"`
		Node            string                      `yaml:"node,omitempty"`
		Relationship    string                      `yaml:"relationship,omitempty"`
		TypeRequirement string                      `yaml:"type_requirement,omitempty"`
		Properties      map[string]*ValueAssignment `yaml:"properties,omitempty"`
	}

	if err := unmarshal(&ra); err == nil {
		r.Capability = ra.Capability
		r.Node = ra.Node
		r.Relationship = ra.Relationship
		r.TypeRequirement = ra.TypeRequirement
		r.RelationshipProps = ra.Properties
		return nil
	}

	var rac struct {
		Capability      string                  `yaml:"capability"`
		Node            string                  `yaml:"node,omitempty"`
		Relationship    RequirementRelationship `yaml:"relationship,omitempty"`
		TypeRequirement string                  `yaml:"type_requirement,omitempty"`
	}
	if err := unmarshal(&rac); err != nil {
		return err
	}
	r.Capability = rac.Capability
	r.Node = rac.Node
	r.Relationship = rac.Relationship.Type
	r.RelationshipProps = rac.Relationship.Properties
	r.TypeRequirement = rac.TypeRequirement
	return nil
}
