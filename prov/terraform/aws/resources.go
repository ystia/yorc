package aws

// A ComputeInstance represent an AWS compute
type ComputeInstance struct {
	ImageID        string   `json:"ami,omitempty"`
	InstanceType   string   `json:"instance_type,omitempty"`
	SecurityGroups []string `json:"security_groups,omitempty"`
	KeyName        string   `json:"key_name,omitempty"`
	Tags           Tags     `json:"tags,omitempty"`

	Provisioners map[string]interface{} `json:"provisioner,omitempty"`
}

// Tags represent a mapping of tags assigned to the Instance.
type Tags struct {
	Name string `json:"Name,omitempty"`
}
