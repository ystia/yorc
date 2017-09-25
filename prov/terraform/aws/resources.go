package aws

// A ComputeInstance represent an AWS compute
type ComputeInstance struct {
	ImageID         string      `json:"ami,omitempty"`
	InstanceType    string      `json:"instance_type,omitempty"`
	Placement       string      `json:"availability_zone,omitempty"`
	SecurityGroups  []string    `json:"security_groups,omitempty"`
	KeyName         string      `json:"key_name,omitempty"`
	Tags            Tags        `json:"tags,omitempty"`
	ElasticIps      []string    `json:"-"`
	RootBlockDevice BlockDevice `json:"root_block_device,omitempty"`

	Provisioners map[string]interface{} `json:"provisioner,omitempty"`
}

// A BlockDevice represents an AWS device block volume
type BlockDevice struct {
	DeleteOnTermination bool `json:"delete_on_termination"`
}

// Tags represent a mapping of tags assigned to the Instance.
type Tags struct {
	Name string `json:"Name,omitempty"`
}

// ElasticIP represents the AWS Elastic IP resource
type ElasticIP struct {
}

// ElasticIPAssociation represents the ElasticIP/ComputeInstance association
// A way to associate/disassociate Elastic IPs from AWS instances
type ElasticIPAssociation struct {
	InstanceID   string `json:"instance_id,omitempty"`
	AllocationID string `json:"allocation_id,omitempty"`
	PublicIP     string `json:"public_ip,omitempty"`
}
