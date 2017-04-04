package commons

// An Infrastructure is the top-level element of a Terraform infrastructure definition
type Infrastructure struct {
	Data     map[string]interface{} `json:"data"`
	Variable map[string]interface{} `json:"variable,omitempty"`
	Provider map[string]interface{} `json:"provider,omitempty"`
	Resource map[string]interface{} `json:"resource,omitempty"`
	Output   map[string]interface{} `json:"output,omitempty"`
}

// The ConsulKeys resource writes sets of individual values into Consul.
type ConsulKeys struct {
	Datacenter string      `json:"datacenter,omitempty"`
	Token      string      `json:"token,omitempty"`
	Keys       []ConsulKey `json:"key"`
}

// A ConsulKey is an individual Key/Value pair to be store into Consul
type ConsulKey struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Default string `json:"default,omitempty"`
	Value   string `json:"value,omitempty"`
	Delete  bool   `json:"delete,omitempty"`
}

// The RemoteExec provisioner invokes a script on a remote resource after it is created.
//
// The remote-exec provisioner supports both ssh and winrm type connections.
type RemoteExec struct {
	Connection Connection `json:"connection,omitempty"`
	Inline     []string   `json:"inline,omitempty"`
	Script     string     `json:"script,omitempty"`
	Scripts    []string   `json:"scripts,omitempty"`
}

// A Connection allows to overwrite the way Terraform connects to a resource
type Connection struct {
	ConnType   string `json:"type,omitempty"`
	User       string `json:"user,omitempty"`
	Password   string `json:"password,omitempty"`
	Host       string `json:"host,omitempty"`
	Port       string `json:"port,omitempty"`
	Timeout    string `json:"timeout,omitempty"` // defaults to "5m"
	PrivateKey string `json:"private_key,omitempty"`
}
