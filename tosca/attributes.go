package tosca

// An AttributeDefinition is the representation of a TOSCA Attribute Definition
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ELEMENT_ATTRIBUTE_DEFN for more details
type AttributeDefinition struct {
	Type        string           `yaml:"type"`
	Description string           `yaml:"description,omitempty"`
	Default     *ValueAssignment `yaml:"default,omitempty"`
	Status      string           `yaml:"status,omitempty"`
	EntrySchema EntrySchema      `yaml:"entry_schema,omitempty"`
}

// UnmarshalYAML unmarshals a yaml into an AttributeDefinition
func (r *AttributeDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var ra struct {
		Type        string           `yaml:"type"`
		Description string           `yaml:"description,omitempty"`
		Default     *ValueAssignment `yaml:"default,omitempty"`
		Status      string           `yaml:"status,omitempty"`
		EntrySchema EntrySchema      `yaml:"entry_schema,omitempty"`
	}

	if err := unmarshal(&ra); err == nil && ra.Type != "" {
		r.Description = ra.Description
		r.Type = ra.Type
		r.Default = ra.Default
		r.Status = ra.Status
		r.EntrySchema = ra.EntrySchema
		return nil
	}

	var ras ValueAssignment
	if err := unmarshal(&ras); err != nil {
		return err
	}
	r.Type = ras.Type.String()
	r.Default = &ras
	return nil
}
