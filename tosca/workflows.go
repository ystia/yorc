package tosca

// An Workflow is the representation of a TOSCA Workflow
//
// Currently Workflows are not part of the TOSCA specification
type Workflow struct {
	Steps map[string]Step `yaml:"steps,omitempty" json:"steps,omitempty"`
}

// An Step is the representation of a TOSCA Workflow Step
//
// Currently Workflows are not part of the TOSCA specification
type Step struct {
	Target             string     `yaml:"target,omitempty" json:"target,omitempty"`
	TargetRelationShip string     `yaml:"target_relationship,omitempty" json:"target_relationship,omitempty"`
	Activities         []Activity `yaml:"activities" json:"activities"`
	OnSuccess          []string   `yaml:"on_success,omitempty" json:"on_success,omitempty"`
	OperationHost      string     `yaml:"operation_host,omitempty" json:"operation_host,omitempty"`
}

// An Activity is the representation of a TOSCA Workflow Step Activity
//
// Currently Workflows are not part of the TOSCA specification
type Activity struct {
	SetState      string `yaml:"set_state,omitempty" json:"set_state,omitempty"`
	Delegate      string `yaml:"delegate,omitempty" json:"delegate,omitempty"`
	CallOperation string `yaml:"call_operation,omitempty" json:"call_operation,omitempty"`
	Inline        string `yaml:"inline,omitempty" json:"inline,omitempty"`
}
