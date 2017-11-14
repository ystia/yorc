package tosca

// An PropertyDefinition is the representation of a TOSCA Property Definition
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.0/TOSCA-Simple-Profile-YAML-v1.0.html#DEFN_ELEMENT_PROPERTY_DEFN for more details
type PropertyDefinition struct {
	Type        string           `yaml:"type"`
	Description string           `yaml:"description,omitempty"`
	Required    *bool            `yaml:"required,omitempty"`
	Default     *ValueAssignment `yaml:"default,omitempty"`
	Status      string           `yaml:"status,omitempty"`
	//Constraints []ConstraintClause `yaml:"constraints,omitempty"`
	EntrySchema EntrySchema `yaml:"entry_schema,omitempty"`
}
