package openstack

type ComputeInstance struct {
	Region           string           `json:"region"`
	Name             string           `json:"name,omitempty"`
	ImageId          string           `json:"image_id,omitempty"`
	ImageName        string           `json:"image_name,omitempty"`
	FlavorId         string           `json:"flavor_id,omitempty"`
	FlavorName       string           `json:"flavor_name,omitempty"`
	FloatingIp       string           `json:"floating_ip,omitempty"`
	SecurityGroups   []string         `json:"security_groups,omitempty"`
	AvailabilityZone string           `json:"availability_zone,omitempty"`
	Networks         []ComputeNetwork `json:"network,omitempty"`
	KeyPair          string           `json:"key_pair,omitempty"`

	Provisioners map[string]interface{} `json:"provisioner,omitempty"`
}

type ComputeNetwork struct {
	UUID          string `json:"uuid,omitempty"`
	Name          string `json:"name,omitempty"`
	Port          string `json:"port,omitempty"`
	FixedIpV4     string `json:"fixed_ip_v4,omitempty"`
	FloatingIp    string `json:"floating_ip,omitempty"`
	AccessNetwork string `json:"access_network,omitempty"`
}
