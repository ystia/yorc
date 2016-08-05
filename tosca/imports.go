package tosca

type ImportMapInteface map[string][]map[string]ImportDefinitionBody

type ImportMap map[string]ImportDefinitionBody
type ImportDefinitionBody struct {
	File            string `yaml:"file,omitempty"`
	Repository      string `yaml:"repository,omitempty"`
	NamespaceURI    string `yaml:"namespace_uri,omitempty"`
	NamespacePrefix string `yaml:"namespace_prefix,omitempty"`
}

func (i *ImportDefinitionBody) UnmarshalYAML(unmarshal func(interface{}) error) error {
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
