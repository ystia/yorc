package tosca

import "novaforge.bull.com/starlings-janus/janus/log"

// An Input is the representation of the input part of a TOSCA Operation Definition
//
// It could be either a Value Assignment or a Property Definition.
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.0/TOSCA-Simple-Profile-YAML-v1.0.html#DEFN_ELEMENT_OPERATION_DEF for more details
type Input struct {
	ValueAssign *ValueAssignment
	PropDef     *PropertyDefinition
}

// UnmarshalYAML unmarshals a yaml into an Input
func (i *Input) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// an Input is either a Property definition or a ValueAssignment
	i.ValueAssign = nil
	i.PropDef = nil
	log.Debug("Try to parse input as a PropertyDefinition")
	p := new(PropertyDefinition)
	if err := unmarshal(p); err == nil && p.Type != "" {
		i.PropDef = p
		return nil
	}
	log.Debug("Try to parse input as a ValueAssignment")
	v := new(ValueAssignment)
	if err := unmarshal(v); err != nil {
		return err
	}
	i.ValueAssign = v
	return nil
}
