package tosca

// An Implementation is the representation of the implementation part of a TOSCA Operation Definition
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.0/TOSCA-Simple-Profile-YAML-v1.0.html#DEFN_ELEMENT_OPERATION_DEF for more details
type Implementation struct {
	Primary      string   `yaml:"primary"`
	Dependencies []string `yaml:"dependencies,omitempty"`
}

// UnmarshalYAML unmarshals a yaml into an Implementation
func (i *Implementation) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err == nil {
		i.Primary = s
		return nil
	}
	var str struct {
		Primary      string   `yaml:"primary"`
		Dependencies []string `yaml:"dependencies,omitempty"`
	}
	if err := unmarshal(&str); err != nil {
		return err
	}
	i.Primary = str.Primary
	i.Dependencies = str.Dependencies
	return nil
}
