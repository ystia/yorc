package tosca

type RequirementDefinitionMap map[string]RequirementDefinition
type RequirementDefinition struct {
	Capability   string `yaml:"capability"`
	Node         string `yaml:"node,omitempty"`
	Relationship string `yaml:"relationship,omitempty"`
	Occurrences  string `yaml:"occurrences,omitempty"`
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
	Type       string `yaml:"type"`
	Properties map[string]ValueAssignment      `yaml:"properties,omitempty"`
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
		Capability   string `yaml:"capability"`
		Node         string `yaml:"node,omitempty"`
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
