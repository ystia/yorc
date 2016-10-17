package tosca

type ParameterDefinition struct {
	PropertyDefinition
	Value ValueAssignment `yaml:"value,omitempty"`
}
