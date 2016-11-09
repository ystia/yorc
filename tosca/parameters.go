package tosca

type ParameterDefinition struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description,omitempty"`
	Required    bool   `yaml:"required,omitempty"`
	Default     string `yaml:"default,omitempty"`
	Status      string `yaml:"status,omitempty"`
	//Constraints []ConstraintClause `yaml:"constraints,omitempty"`
	//EntrySchema string `yaml:"entry_schema,omitempty"`
	Value ValueAssignment `yaml:"value,omitempty"`
}
