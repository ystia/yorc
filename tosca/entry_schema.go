package tosca

// An EntrySchema is the representation of a TOSCA Entry Schema
type EntrySchema struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description,omitempty"`
	//Constraints []ConstraintClause `yaml:"constraints,omitempty"`
}
