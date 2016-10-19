package tosca

import (
	"fmt"
	"novaforge.bull.com/starlings-janus/janus/log"
	"strconv"
)

type RequirementDefinitionMap map[string]RequirementDefinition
type RequirementDefinition struct {
	Capability   string     `yaml:"capability"`
	Node         string     `yaml:"node,omitempty"`
	Relationship string     `yaml:"relationship,omitempty"`
	Occurrences  ToscaRange `yaml:"occurrences,omitempty"`
	// Extra types used in list (A4C) mode
	name string
}

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

func (a *RequirementDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err == nil {
		a.Capability = s
		return nil
	}
	var str struct {
		Capability        string     `yaml:"capability"`
		Node              string     `yaml:"node,omitempty"`
		Relationship      string     `yaml:"relationship,omitempty"`
		RelationshipAlien string     `yaml:"relationship_type,omitempty"`
		Occurrences       ToscaRange `yaml:"occurrences,omitempty"`

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
		if bound, err := strconv.ParseUint(str.LowerBound, 10, 0); err != nil {
			return fmt.Errorf("Expecting a unsigned integer as lower bound got: %q", str.LowerBound)
		} else {
			a.Occurrences.LowerBound = bound
		}
	}
	if str.UpperBound != "" {
		if bound, err := strconv.ParseUint(str.UpperBound, 10, 0); err != nil {
			if str.UpperBound != "UNBOUNDED" {
				return fmt.Errorf("Expecting a unsigned integer or the 'UNBOUNDED' keyword as upper bound of the range got: %q", str.UpperBound)
			}
			a.Occurrences.UpperBound = UNBOUNDED
		} else {
			a.Occurrences.UpperBound = bound
		}
	}
	return nil
}

type RequirementAssignmentMap map[string]RequirementAssignment
type RequirementAssignment struct {
	Capability        string `yaml:"capability"`
	Node              string `yaml:"node,omitempty"`
	Relationship      string `yaml:"relationship,omitempty"`
	RelationshipProps map[string]ValueAssignment
	// NodeFilter
}

type RequirementRelationship struct {
	Type       string                     `yaml:"type"`
	Properties map[string]ValueAssignment `yaml:"properties,omitempty"`
}

func (r *RequirementAssignment) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var ras string
	if err := unmarshal(&ras); err == nil {
		r.Node = ras
		return nil
	}

	var ra struct {
		Capability   string `yaml:"capability"`
		Node         string `yaml:"node,omitempty"`
		Relationship string `yaml:"relationship,omitempty"`
	}

	if err := unmarshal(&ra); err == nil {
		r.Capability = ra.Capability
		r.Node = ra.Node
		r.Relationship = ra.Relationship
		return nil
	}

	var rac struct {
		Capability   string                  `yaml:"capability"`
		Node         string                  `yaml:"node,omitempty"`
		Relationship RequirementRelationship `yaml:"relationship,omitempty"`
	}
	if err := unmarshal(&rac); err != nil {
		return err
	}
	r.Capability = rac.Capability
	r.Node = rac.Node
	r.Relationship = rac.Relationship.Type
	r.RelationshipProps = rac.Relationship.Properties
	return nil
}
