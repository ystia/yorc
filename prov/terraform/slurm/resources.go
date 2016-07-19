package slurm

type ComputeInstance struct {
	Name             string           `json:"name,omitempty"`
	GpuType          string           `json:"gputype,omitempty"`

	Provisioners     map[string]interface{} `json:"provisioner,omitempty"`

}

type Volume struct {
	VolumeId string    `json:"volume_id"`
	Device   string    `json:"device,omitempty"`
}

type ComputeNetwork struct {
	UUID          string `json:"uuid,omitempty"`
	Name          string `json:"name,omitempty"`
	Port          string `json:"port,omitempty"`
	FixedIpV4     string `json:"fixed_ip_v4,omitempty"`
	FloatingIp    string `json:"floating_ip,omitempty"`
	AccessNetwork string `json:"access_network,omitempty"`
}

type BlockStorageVolume struct {
	Region           string   `json:"region"`
	Size             int         `json:"size"`
	Name             string     `json:"name,omitempty"`
	Description      string     `json:"description,omitempty"`
	AvailabilityZone string           `json:"availability_zone,omitempty"`
}
