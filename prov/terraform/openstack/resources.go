// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openstack

import (
	"github.com/ystia/yorc/v4/prov/terraform/commons"
)

const (
	defaultOSRegion             = "RegionOne"
	infrastructureType          = "openstack"
	bootVolumeTOSCAAttr         = "boot_volume"
	uuidTOSCAKey                = "uuid"
	sourceTOSCAKey              = "source"
	destinationTOSCAKey         = "destination"
	sizeTOSCAKey                = "size"
	volumeTypeTOSCAKey          = "volume_type"
	deleteOnTerminationTOSCAKey = "delete_on_termination"
)

// A ComputeInstance represent an OpenStack compute
type ComputeInstance struct {
	Region           string            `json:"region"`
	Name             string            `json:"name,omitempty"`
	ImageID          string            `json:"image_id,omitempty"`
	ImageName        string            `json:"image_name,omitempty"`
	BootVolume       *BootVolume       `json:"block_device,omitempty"`
	FlavorID         string            `json:"flavor_id,omitempty"`
	FlavorName       string            `json:"flavor_name,omitempty"`
	FloatingIP       string            `json:"floating_ip,omitempty"`
	SecurityGroups   []string          `json:"security_groups,omitempty"`
	AvailabilityZone string            `json:"availability_zone,omitempty"`
	Networks         []ComputeNetwork  `json:"network,omitempty"`
	KeyPair          string            `json:"key_pair,omitempty"`
	SchedulerHints   SchedulerHints    `json:"scheduler_hints,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	UserData         string            `json:"user_data,omitempty"`
	commons.Resource
}

// BootVolume used by a Compute Instance and its terraform json attributes
type BootVolume struct {
	UUID                string `json:"uuid,,omitempty"`
	Source              string `json:"source_type"`
	Destination         string `json:"destination_type,omitempty"`
	Size                int    `json:"volume_size,omitempty"`
	VolumeType          string `json:"volume_type,omitempty"`
	DeleteOnTermination bool   `json:"delete_on_termination,omitempty"`
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
type ComputeFloatingIPAssociate struct {
	Region     string `json:"region"`
	FloatingIP string `json:"floating_ip"`
	InstanceID string `json:"instance_id"`
	FixedIP    string `json:"fixed_ip,omitempty"`
}

// A ComputeVolumeAttach attaches a volume to an instance.
type ComputeVolumeAttach struct {
	Region     string `json:"region"`
	VolumeID   string `json:"volume_id"`
	InstanceID string `json:"instance_id"`
	Device     string `json:"device,omitempty"`
}

// ServerGroup represents an OpenStack Server group
// https://www.terraform.io/docs/providers/openstack/r/compute_servergroup_v2.html
type ServerGroup struct {
	Name     string   `json:"name"`
	Policies []string `json:"policies"`
}

// SchedulerHints represents a scheduler_hints block for computeInstance
type SchedulerHints struct {
	Group string `json:"group"`
}
