package tosca

type InterfaceDefinitionMap map[string]InterfaceDefinition

type InterfaceDefinition struct {
	Inputs         map[string]ValueAssignment `yaml:"inputs,omitempty"`
	Description    string                     `yaml:"description,omitempty"`
	Implementation Implementation             `yaml:"implementation,omitempty"`
}

func (i *InterfaceDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err == nil {
		i.Implementation = Implementation{Primary: s}
		return nil
	}
	var str struct {
		Inputs         map[string]ValueAssignment `yaml:"inputs,omitempty"`
		Description    string                     `yaml:"description,omitempty"`
		Implementation Implementation             `yaml:"implementation,omitempty"`
	}
	if err := unmarshal(&str); err != nil {
		return err
	}
	i.Inputs = str.Inputs
	i.Implementation = str.Implementation
	i.Description = str.Description
	return nil
}
