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

	Volumes []Volume `json:"volume,omitempty"`
}

type Volume struct {
	VolumeId string `json:"volume_id"`
	Device   string `json:"device,omitempty"`
}

type ComputeNetwork struct {
	UUID          string `json:"uuid,omitempty"`
	Name          string `json:"name,omitempty"`
	Port          string `json:"port,omitempty"`
	FixedIpV4     string `json:"fixed_ip_v4,omitempty"`
	FloatingIp    string `json:"floating_ip,omitempty"`
	AccessNetwork bool   `json:"access_network,omitempty"`
}

type BlockStorageVolume struct {
	Region           string `json:"region"`
	Size             int    `json:"size"`
	Name             string `json:"name,omitempty"`
	Description      string `json:"description,omitempty"`
	AvailabilityZone string `json:"availability_zone,omitempty"`
}

type FloatingIP struct {
	Pool string `json:"pool,omitempty"`
}

type Network struct {
	Region     string `json:"region,omitempty"`
	Name       string `json:"name,omitempty"`
	Shared     string `json:"shared,omitempty"`
	AdminState string `json:"admin_state_up,omitempty"`
}

type Subnet struct {
	Region          string          `json:"region"`
	NetworkID       string          `json:"network_id"`
	CIDR            string          `json:"cidr"`
	IPVersion       int             `json:"ip_version,omitempty"`
	Name            string          `json:"name,omitempty"`
	GatewayIP       string          `json:"gateway_ip,omitempty"`
	AllocationPools *AllocationPool `json:"allocation_pools,omitempty"`
	EnableDHCP      bool            `json:"enable_dhcp,omitempty"`
}

type AllocationPool struct {
	Start string `json:"start"`
	End   string `json:"end"`
}
