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
	ImageID          string      `json:"ami,omitempty"`
	InstanceType     string      `json:"instance_type,omitempty"`
	AvailabilityZone string      `json:"availability_zone,omitempty"`
	PlacementGroup   string      `json:"placement_group,omitempty"`
	SecurityGroups   []string    `json:"security_groups,omitempty"`
	KeyName          string      `json:"key_name,omitempty"`
	Tags             Tags        `json:"tags,omitempty"`
	ElasticIps       []string    `json:"-"`
	RootBlockDevice  BlockDevice `json:"root_block_device,omitempty"`

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
	AvailabilityZone string `json:"availability_zone,omitempty"`
	Encrypted        bool   `json:"encrypted,omitempty"`
	Size             int    `json:"size,omitempty"`
	SnapshotID       string `json:"snapshot_id,omitempty"`
	KMSKeyID         string `json:"kms_key_id,omitempty"`
}
