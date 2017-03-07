package tosca

// An Workflow is the representation of a TOSCA Workflow
//
// Currently Workflows are not part of the TOSCA specification
type Workflow struct {
	Steps map[string]Step `yaml:",omitempty"`
}

// An Step is the representation of a TOSCA Workflow Step
//
// Currently Workflows are not part of the TOSCA specification
type Step struct {
	Node      string
	Activity  Activity
	OnSuccess []string `yaml:"on-success,omitempty"`
}

// An Activity is the representation of a TOSCA Workflow Step Activity
//
// Currently Workflows are not part of the TOSCA specification
type Activity struct {
	SetState      string `yaml:"set_state,omitempty"`
	Delegate      string `yaml:",omitempty"`
	CallOperation string `yaml:"call_operation,omitempty"`
}
