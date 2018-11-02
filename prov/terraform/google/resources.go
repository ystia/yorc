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

package google

// ComputeInstance represents a Google Compute Engine Virtual Machine
// See https://www.terraform.io/docs/providers/google/r/compute_instance.html
// for the latest documentation.
type ComputeInstance struct {
	Name              string             `json:"name"`
	MachineType       string             `json:"machine_type"`
	Zone              string             `json:"zone"`
	BootDisk          BootDisk           `json:"boot_disk,omitempty"`
	NetworkInterfaces []NetworkInterface `json:"network_interface"`
	Description       string             `json:"description,omitempty"`
	Labels            map[string]string  `json:"labels,omitempty"`
	Metadata          map[string]string  `json:"metadata,omitempty"`
	Scheduling        Scheduling         `json:"scheduling,omitempty"`
	// ServiceAccounts is an array of at most one element
	ServiceAccounts []ServiceAccount `json:"service_account,omitempty"`
	Tags            []string         `json:"tags,omitempty"`
	ScratchDisks    []ScratchDisk    `json:"scratch_disk,omitempty"`
}

// BootDisk represents the required boot disk for compute instance
type BootDisk struct {
	AutoDelete bool `json:"auto_delete,omitempty"`
	// InitializeParams is an array of at most one element
	InitializeParams InitializeParams `json:"initialize_params,omitempty"`
}

// InitializeParams represents the initialize params
type InitializeParams struct {
	Image string `json:"image,omitempty"`
}

// NetworkInterface represents a network to attach to the instance
type NetworkInterface struct {
	Network           string         `json:"network,omitempty"`
	Subnetwork        string         `json:"subnetwork,omitempty"`
	SubNetworkProject string         `json:"subnetwork_project,omitempty"`
	Address           string         `json:"address,omitempty"`
	AccessConfigs     []AccessConfig `json:"access_config,omitempty"`
}

// AccessConfig provides IPs via which this instance can be accessed via the Internet
type AccessConfig struct {
	NatIP               string `json:"nat_ip,omitempty"`
	NetworkTier         string `json:"network_tier,omitempty"`
	PublicPtrDomainName string `json:"public_ptr_domain_name ,omitempty"`
}

// ServiceAccount to attach to the instance
type ServiceAccount struct {
	Email  string   `json:"email,omitempty"`
	Scopes []string `json:"scopes,omitempty"`
}

// Scheduling strategy to use
type Scheduling struct {
	Preemptible bool `json:"preemptible,omitempty"`
}

// ComputeAddress represents a Google compute static IP address
// See https://www.terraform.io/docs/providers/google/r/compute_address.html for more information
type ComputeAddress struct {
	Name        string            `json:"name"`
	Region      string            `json:"region"`
	Address     string            `json:"address,omitempty"`
	AddressType string            `json:"address_type,omitempty"`
	Description string            `json:"description,omitempty"`
	NetworkTier string            `json:"network_tier,omitempty"`
	SubNetwork  string            `json:"subnetwork,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Project     string            `json:"project,omitempty"`
}

// EncryptionKey represents a Google encryption key
type EncryptionKey struct {
	Raw    string `json:"raw_key,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}

// ScratchDisk represents an additional Compute instance local scratch disk
type ScratchDisk struct {
	Interface string `json:"interface,omitempty"`
}

// PersistentDisk represents a Google persistent disk
// See https://www.terraform.io/docs/providers/google/r/compute_disk.html
type PersistentDisk struct {
	Name                        string            `json:"name"`
	Size                        int               `json:"size,omitempty"`
	Description                 string            `json:"description,omitempty"`
	Type                        string            `json:"type,omitempty"`
	Labels                      map[string]string `json:"labels,omitempty"`
	Zone                        string            `json:"zone,omitempty"`
	DiskEncryptionKey           *EncryptionKey    `json:"disk_encryption_key,omitempty"`
	SourceSnapshot              string            `json:"snapshot,omitempty"`
	SourceSnapshotEncryptionKey *EncryptionKey    `json:"source_snapshot_encryption_key,omitempty"`
	SourceImage                 string            `json:"image,omitempty"`
	SourceImageEncryptionKey    *EncryptionKey    `json:"source_image_encryption_key,omitempty"`
}

// ComputeAttachedDisk represents compute instance's attached disk
// See https://www.terraform.io/docs/providers/google/r/compute_attached_disk.html
type ComputeAttachedDisk struct {
	Instance   string `json:"instance"`
	Disk       string `json:"disk"`
	DeviceName string `json:"device_name,omitempty"`
	Mode       string `json:"mode,omitempty"`
	Zone       string `json:"zone,omitempty"`
}

// PrivateNetwork represents a Google private network
// See https://www.terraform.io/docs/providers/google/r/compute_network.html
type PrivateNetwork struct {
	Name                  string `json:"name"`
	AutoCreateSubNetworks bool   `json:"auto_create_subnetworks"`
	RoutingMode           string `json:"routing_mode,omitempty"`
	Description           string `json:"description,omitempty"`
	Project               string `json:"project,omitempty"`
}

// SubNetwork represents a Google sub-network
// See https://www.terraform.io/docs/providers/google/r/compute_subnetwork.html
type SubNetwork struct {
	Name                  string    `json:"name"`
	Network               string    `json:"network"`
	IPCIDRRange           string    `json:"ip_cidr_range"`
	Description           string    `json:"description,omitempty"`
	EnableFlowLogs        bool      `json:"enable_flow_logs"`
	PrivateIPGoogleAccess bool      `json:"private_ip_google_access"`
	Region                string    `json:"region,omitempty"`
	Project               string    `json:"project,omitempty"`
	SecondaryIPRanges     []IPRange `json:"secondary_ip_range,omitempty"`
}

// IPRange Represents an IP Range
type IPRange struct {
	Name        string `json:"range_name"`
	IPCIDRRange string `json:"ip_cidr_range"`
}

// Firewall represents a firewall resource
// See https://www.terraform.io/docs/providers/google/r/compute_firewall.html
type Firewall struct {
	Name         string      `json:"name"`
	Network      string      `json:"network"`
	Allow        []AllowRule `json:"allow,omitempty"`
	SourceRanges []string    `json:"source_ranges,omitempty"`
}

// AllowRule represents an allowing firewall rule
type AllowRule struct {
	Protocol string   `json:"protocol"`
	Ports    []string `json:"ports,omitempty"`
}
