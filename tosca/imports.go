package tosca

// ImportMap is a map of ImportDefinition indexed by name
type ImportMap map[string]ImportDefinition

// An ImportDefinition is the representation of a TOSCA Import Definition
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.0/TOSCA-Simple-Profile-YAML-v1.0.html#DEFN_ELEMENT_IMPORT_DEF for more details
type ImportDefinition struct {
	File            string `yaml:"file,omitempty"`
	Repository      string `yaml:"repository,omitempty"`
	NamespaceURI    string `yaml:"namespace_uri,omitempty"`
	NamespacePrefix string `yaml:"namespace_prefix,omitempty"`
}

// UnmarshalYAML unmarshals a yaml into an ImportDefinition
func (i *ImportDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string

	if err := unmarshal(&s); err == nil {
		i.File = s
		return nil
	}
	var str struct {
		File            string `yaml:"file,omitempty"`
		Repository      string `yaml:"repository,omitempty"`
		NamespaceURI    string `yaml:"namespace_uri,omitempty"`
		NamespacePrefix string `yaml:"namespace_prefix,omitempty"`
	}
	if err := unmarshal(&str); err != nil {
		return err
	}
	i.File = str.File
	i.Repository = str.Repository
	i.NamespaceURI = str.NamespaceURI
	i.NamespacePrefix = str.NamespacePrefix
	return nil
}
