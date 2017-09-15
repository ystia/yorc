package commons

// An Infrastructure is the top-level element of a Terraform infrastructure definition
type Infrastructure struct {
	Terraform map[string]interface{} `json:"terraform,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Variable  map[string]interface{} `json:"variable,omitempty"`
	Provider  map[string]interface{} `json:"provider,omitempty"`
	Resource  map[string]interface{} `json:"resource,omitempty"`
	Output    map[string]*Output     `json:"output,omitempty"`
}

// The ConsulKeys can be used as 'resource' to writes or 'data' to read sets of individual values into Consul.
type ConsulKeys struct {
	Datacenter string      `json:"datacenter,omitempty"`
	Token      string      `json:"token,omitempty"`
	Keys       []ConsulKey `json:"key"`
}

// A Resource is the base type for terraform resources
type Resource struct {
	Connection   *Connection              `json:"connection,omitempty"`
	Provisioners []map[string]interface{} `json:"provisioner,omitempty"`
}

// A ConsulKey can be used in a ConsulKeys 'resource' to writes or a ConsulKeys 'data' to read an individual Key/Value pair into Consul
type ConsulKey struct {
	Path string `json:"path"`

	// Should only be use in datasource (read) mode, this is the name to use to access this key within the terraform interpolation syntax
	Name string `json:"name,omitempty"`
	// Should only be use in datasource (read) mode, default value if the key is not found

	Default string `json:"default,omitempty"`
	// Should only be use in resource (write) mode, the value to set to the key

	Value string `json:"value,omitempty"`

	// Should only be use in resource (write) mode, deletes the key
	Delete bool `json:"delete,omitempty"`
}

// The RemoteExec provisioner invokes a script on a remote resource after it is created.
//
// The remote-exec provisioner supports both ssh and winrm type connections.
type RemoteExec struct {
	Connection *Connection `json:"connection,omitempty"`
	Inline     []string    `json:"inline,omitempty"`
	Script     string      `json:"script,omitempty"`
	Scripts    []string    `json:"scripts,omitempty"`
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

// An Output allows to define a terraform output value
type Output struct {
	// Value is the value of the output. This can be a string, list, or map.
	// This usually includes an interpolation since outputs that are static aren't usually useful.
	Value     interface{} `json:"value"`
	Sensitive bool        `json:"sensitive,omitempty"`
}

func AddResource(infrastructure *Infrastructure, resourceType, resourceName string, resource interface{}) {
	if len(infrastructure.Resource) != 0 {
		if infrastructure.Resource[resourceType] != nil && len(infrastructure.Resource[resourceType].(map[string]interface{})) != 0 {
			resourcesMap := infrastructure.Resource[resourceType].(map[string]interface{})
			resourcesMap[resourceName] = resource
		} else {
			resourcesMap := make(map[string]interface{})
			resourcesMap[resourceName] = resource
			infrastructure.Resource[resourceType] = resourcesMap
		}

	} else {
		resourcesMap := make(map[string]interface{})
		infrastructure.Resource = resourcesMap
		resourcesMap = make(map[string]interface{})
		resourcesMap[resourceName] = resource
		infrastructure.Resource[resourceType] = resourcesMap
	}
}

func AddOutput(infrastructure *Infrastructure, outputName string, output *Output) {
	if infrastructure.Output == nil {
		infrastructure.Output = make(map[string]*Output)
	}
	infrastructure.Output[outputName] = output
}
