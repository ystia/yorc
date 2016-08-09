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

type ConsulKeyIP struct {
	Pool string `json:"pool"`
}

type RemoteExec struct {
	Connection Connection `json:"connection,omitempty"`
	Inline     []string   `json:"inline,omitempty"`
	Script     string     `json:"script,omitempty"`
	Scripts    []string   `json:"scripts,omitempty"`
}

type Connection struct {
	ConnType   string `json:"type,omitempty"`
	User       string `json:"user,omitempty"`
	Password   string `json:"password,omitempty"`
	Host       string `json:"host,omitempty"`
	Port       string `json:"port,omitempty"`
	Timeout    string `json:"timeout,omitempty"` // defaults to "5m"
	PrivateKey string `json:"private_key,omitempty"`
}
