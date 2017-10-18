package tosca

// An ParameterDefinition is the representation of a TOSCA Parameter Definition
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.0/TOSCA-Simple-Profile-YAML-v1.0.html#DEFN_ELEMENT_PARAMETER_DEF for more details
type ParameterDefinition struct {
	Type        string           `yaml:"type"`
	Description string           `yaml:"description,omitempty"`
	Required    bool             `yaml:"required,omitempty"`
	Default     *ValueAssignment `yaml:"default,omitempty"`
	Status      string           `yaml:"status,omitempty"`
	//Constraints []ConstraintClause `yaml:"constraints,omitempty"`
	EntrySchema EntrySchema      `yaml:"entry_schema,omitempty"`
	Value       *ValueAssignment `yaml:"value,omitempty"`
}
