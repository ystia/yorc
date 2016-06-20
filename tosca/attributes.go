package tosca

type AttributeDefinition struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description,omitempty"`
	Default     string `yaml:"default,omitempty"`
	Status      string `yaml:"status,omitempty"`
	//EntrySchema string `yaml:"entry_schema,omitempty"`
}
