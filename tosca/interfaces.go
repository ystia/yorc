package tosca

// InterfaceDefinitionMap is a map of InterfaceDefinition indexed by name
type InterfaceDefinitionMap map[string]InterfaceDefinition

// An InterfaceDefinition is the representation of a TOSCA Interface Definition
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.0/TOSCA-Simple-Profile-YAML-v1.0.html#DEFN_ELEMENT_INTERFACE_DEF for more details
type InterfaceDefinition struct {
	Inputs         map[string]Input `yaml:"inputs,omitempty"`
	Description    string           `yaml:"description,omitempty"`
	Implementation Implementation   `yaml:"implementation,omitempty"`
}

// UnmarshalYAML unmarshals a yaml into an InterfaceDefinition
func (i *InterfaceDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err == nil {
		i.Implementation = Implementation{Primary: s}
		return nil
	}
	var str struct {
		Inputs         map[string]Input `yaml:"inputs,omitempty"`
		Description    string           `yaml:"description,omitempty"`
		Implementation Implementation   `yaml:"implementation,omitempty"`
	}
	if err := unmarshal(&str); err != nil {
		return err
	}
	i.Inputs = str.Inputs
	i.Implementation = str.Implementation
	i.Description = str.Description
	return nil
}
