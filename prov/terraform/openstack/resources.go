package openstack

import (
	"github.com/ystia/yorc/prov/terraform/commons"
)

const defaultOSRegion = "RegionOne"
const infrastructureName = "openstack"

// A ComputeInstance represent an OpenStack compute
type ComputeInstance struct {
	Region           string           `json:"region"`
	Name             string           `json:"name,omitempty"`
	ImageID          string           `json:"image_id,omitempty"`
	ImageName        string           `json:"image_name,omitempty"`
	FlavorID         string           `json:"flavor_id,omitempty"`
	FlavorName       string           `json:"flavor_name,omitempty"`
	FloatingIP       string           `json:"floating_ip,omitempty"`
	SecurityGroups   []string         `json:"security_groups,omitempty"`
	AvailabilityZone string           `json:"availability_zone,omitempty"`
	Networks         []ComputeNetwork `json:"network,omitempty"`
	KeyPair          string           `json:"key_pair,omitempty"`

	commons.Resource

	// Deprecated use ComputeVolumeAttach instead
	Volumes []Volume `json:"volume,omitempty"`
}

// A Volume represent an OpenStack volume (BlockStorage) attachment to a ComputeInstance
type Volume struct {
	VolumeID string `json:"volume_id"`
	Device   string `json:"device,omitempty"`
}

// A ComputeNetwork represent an OpenStack virtual network bound to a ComputeInstance
type ComputeNetwork struct {
	UUID          string `json:"uuid,omitempty"`
	Name          string `json:"name,omitempty"`
	Port          string `json:"port,omitempty"`
	FixedIPV4     string `json:"fixed_ip_v4,omitempty"`
	AccessNetwork bool   `json:"access_network,omitempty"`

	// Deprecated use ComputeFloatingIPAssociate instead
	FloatingIP string `json:"floating_ip,omitempty"`
}

// A BlockStorageVolume represent an OpenStack volume (BlockStorage)
type BlockStorageVolume struct {
	Region           string `json:"region"`
	Size             int    `json:"size"`
	Name             string `json:"name,omitempty"`
	Description      string `json:"description,omitempty"`
	AvailabilityZone string `json:"availability_zone,omitempty"`
}

// A FloatingIP represent an OpenStack Floating IP pool configuration
type FloatingIP struct {
	Pool string `json:"pool,omitempty"`
}

// A Network represent an OpenStack virtual network
type Network struct {
	Region     string `json:"region,omitempty"`
	Name       string `json:"name,omitempty"`
	Shared     string `json:"shared,omitempty"`
	AdminState string `json:"admin_state_up,omitempty"`
}

// A Subnet represent an OpenStack virtual subnetwork
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

// A AllocationPool represent an allocation pool for OpenStack virtual subnetwork
type AllocationPool struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// A ComputeFloatingIPAssociate associates a floating IP to an instance.
// This should be used instead of the floating_ip options in openstack_compute_instance_v2 now deprecated.
type ComputeFloatingIPAssociate struct {
	Region     string `json:"region"`
	FloatingIP string `json:"floating_ip"`
	InstanceID string `json:"instance_id"`
	FixedIP    string `json:"fixed_ip,omitempty"`
}

// A ComputeVolumeAttach attaches a volume to an instance.
// This should be used instead of the floating_ip options in openstack_compute_instance_v2 now deprecated.
type ComputeVolumeAttach struct {
	Region     string `json:"region"`
	VolumeID   string `json:"volume_id"`
	InstanceID string `json:"instance_id"`
	Device     string `json:"device,omitempty"`
}
