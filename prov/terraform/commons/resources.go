package commons

type Infrastructure struct {
	Variable map[string]interface{} `json:"variable,omitempty"`
	Provider map[string]interface{} `json:"provider,omitempty"`
	Resource map[string]interface{} `json:"resource,omitempty"`
	Output   map[string]interface{} `json:"output,omitempty"`
}

type ConsulKeys struct {
	Datacenter string      `json:"datacenter,omitempty"`
	Token      string      `json:"token,omitempty"`
	Keys       []ConsulKey `json:"key"`
}

type ConsulKey struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Default string `json:"default,omitempty"`
	Value   string `json:"value,omitempty"`
	Delete  bool   `json:"delete,omitempty"`
}
