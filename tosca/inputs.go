package tosca

import "novaforge.bull.com/starlings-janus/janus/log"

type Input struct {
	ValueAssign *ValueAssignment
	PropDef     *PropertyDefinition
}

func (i *Input) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// an Input is either a Property definition or a ValueAssignment
	i.ValueAssign = nil
	i.PropDef = nil
	log.Debug("Try to parse input as a PropertyDefinition")
	var v *ValueAssignment = new(ValueAssignment)
	if err := unmarshal(v); err == nil {
		i.ValueAssign = v
		return nil
	}
	log.Debug("Try to parse input as a ValueAssignment")
	var p *PropertyDefinition = new(PropertyDefinition)
	if err := unmarshal(p); err != nil {
		return err
	}
	i.PropDef = p
	return nil
}
