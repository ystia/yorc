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
	Capability   string `yaml:"capability"`
	Node         string `yaml:"node,omitempty"`
	Relationship string `yaml:"relationship,omitempty"`
	// NodeFilter
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

	if err := unmarshal(&ra); err != nil {
		return err
	}
	r.Capability = ra.Capability
	r.Node = ra.Node
	r.Relationship = ra.Relationship
	return nil
}
