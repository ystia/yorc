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

package aws

// A ComputeInstance represent an AWS compute
type ComputeInstance struct {
	ImageID          string            `json:"ami,omitempty"`
	InstanceType     string            `json:"instance_type,omitempty"`
	AvailabilityZone string            `json:"availability_zone,omitempty"`
	PlacementGroup   string            `json:"placement_group,omitempty"`
	SecurityGroups   []string          `json:"security_groups,omitempty"`
	SubnetID         string            `json:"subnet_id,omitempty"`
	KeyName          string            `json:"key_name,omitempty"`
	Tags             Tags              `json:"tags,omitempty"`
	ElasticIps       []string          `json:"-"`
	RootBlockDevice  BlockDevice       `json:"root_block_device,omitempty"`
	NetworkInterface map[string]string `json:"network_interface,omitempty"`

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

// EBSVolume represents an EBS Volume
// see : https://www.terraform.io/docs/providers/aws/r/ebs_volume.html
type EBSVolume struct {
	AvailabilityZone string            `json:"availability_zone,omitempty"`
	Encrypted        bool              `json:"encrypted,omitempty"`
	Size             int               `json:"size,omitempty"`
	SnapshotID       string            `json:"snapshot_id,omitempty"`
	KMSKeyID         string            `json:"kms_key_id,omitempty"`
	Type             string            `json:"type,omitempty"`
	IOPS             string            `json:"iops,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
}

// VolumeAttachment provide a way to attach an EBS volume to an EC2 instance
// see : https://www.terraform.io/docs/providers/aws/r/volume_attachment.html
type VolumeAttachment struct {
	DeviceName string `json:"device_name,omitempty"`
	InstanceID string `json:"instance_id,omitempty"`
	VolumeID   string `json:"volume_id,omitempty"`
}

// VPC represents a virtual private cloud
// see : https://www.terraform.io/docs/providers/aws/r/vpc.html
type VPC struct {
	CidrBlock                    string            `json:"cidr_block,omitempty"`
	InstanceTenancy              string            `json:"instance_tenancy,omitempty"`
	EnableDNSSupport             bool              `json:"enable_dns_support,omitempty"`
	EnableDNSHostnames           bool              `json:"enable_dns_hostnames,omitempty"`
	EnableClassiclink            bool              `json:"enable_classiclink,omitempty"`
	EnableClassiclinkDNSSupport  bool              `json:"enable_classiclink_dns_support,omitempty"`
	AssignGeneratedIpv6CidrBlock bool              `json:"assign_generated_ipv6_cidr_block,omitempty"`
	Tags                         map[string]string `json:"tags,omitempty"`
}

// Subnet reprents a network in a VPC
// see : https://www.terraform.io/docs/providers/aws/r/subnet.html
type Subnet struct {
	AvailabilityZone            string            `json:"availability_zone,omitempty"`
	AvailabilityZoneID          string            `json:"availability_zone_id,omitempty"`
	CidrBlock                   string            `json:"cidr_block,omitempty"`
	Ipv6CidrBlock               string            `json:"ipv6_cidr_block,omitempty"`
	MapPublicIPOnLaunch         bool              `json:"map_public_ip_on_launch,omitempty"`
	AssignIpv6AddressOnCreation bool              `json:"assign_ipv6_address_on_creation,omitempty"`
	VPCId                       string            `json:"vpc_id,omitempty"`
	Tags                        map[string]string `json:"tags,omitempty"`
}

// NetworkInterface create an ENI (Elastic Network Interface)
// see : https://www.terraform.io/docs/providers/aws/r/network_interface.html#attachment
type NetworkInterface struct {
	SubnetID       string            `json:"subnet_id,omitempty"`
	Description    string            `json:"description,omitempty"`
	PrivateIps     string            `json:"private_ips,omitempty"`
	SecurityGroups []string          `json:"security_groups,omitempty"`
	Attachment     map[string]string `json:"attachment,omitempty"`
}

// SecurityRule provide a egress/ingress resource.
type SecurityRule struct {
	FromPort  string   `json:"from_port,omitempty"`
	ToPort    string   `json:"to_port,omitempty"`
	Protocol  string   `json:"protocol,omitempty"`
	CidrBlock []string `json:"cidr_blocks,omitempty"`
}

// SecurityGroups provides a security group resource
// see : https://www.terraform.io/docs/providers/aws/r/security_group.html
type SecurityGroups struct {
	VPCId   string       `json:"vpc_id,omitempty"`
	Egress  SecurityRule `json:"egress,omitempty"`
	Ingress SecurityRule `json:"ingress,omitempty"`
	Name    string       `json:"name,omitempty"`
}

// RouteTable provides a resource to create a VPC routing table.
// see : https://www.terraform.io/docs/providers/aws/r/route_table.html
type RouteTable struct {
	VPCId     string            `json:"vpc_id,omitempty"`
	Route     map[string]string `json:"route,omitempty"`
	DependsOn []string          `json:"depends_on,omitempty"`
}

// DefaultRouteTable provides a resource to manage a Default VPC Routing Table
// see : https://www.terraform.io/docs/providers/aws/r/default_route_table.html
type DefaultRouteTable struct {
	DefaultRouteTableID string            `json:"default_route_table_id,omitempty"`
	Route               map[string]string `json:"route,omitempty"`
	DependsOn           []string          `json:"depends_on,omitempty"`
}

// InternetGateway provides a resource to create a VPC Internet Gateway
// see : https://www.terraform.io/docs/providers/aws/r/internet_gateway.html
type InternetGateway struct {
	VPCId string `json:"vpc_id,omitempty"`
}
