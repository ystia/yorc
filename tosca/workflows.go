package tosca

type Workflow struct {
	Steps map[string]Step `yaml:",omitempty"`
}

type Step struct {
	Node      string
	Activity  Activity
	OnSuccess []string `yaml:"on-success,omitempty"`
}

type Activity struct {
	SetState      string `yaml:"set_state,omitempty"`
	Delegate      string `yaml:",omitempty"`
	CallOperation string `yaml:"call_operation,omitempty"`
}
