package config

const DefaultConsulPubMaxRoutines int = 500

type Configuration struct {
	OSAuthURL               string   `json:"os_auth_url,omitempty"`
	OSTenantID              string   `json:"os_tenant_id,omitempty"`
	OSTenantName            string   `json:"os_tenant_name,omitempty"`
	OSUserName              string   `json:"os_user_name,omitempty"`
	OSPassword              string   `json:"os_password,omitempty"`
	OSRegion                string   `json:"os_region,omitempty"`
	ResourcesPrefix         string   `json:"os_prefix,omitempty"`
	OSPrivateNetworkName    string   `json:"os_private_network_name,omitempty"`
	OSPublicNetworkName     string   `json:"os_public_network_name,omitempty"`
	OSDefaultSecurityGroups []string `json:"os_default_security_groups,omitempty"`
	ConsulToken             string   `json:"consul_token,omitempty"`
	ConsulDatacenter        string   `json:"consul_datacenter,omitempty"`
	ConsulAddress           string   `json:"consul_address,omitempty"`
	ConsulPubMaxRoutines    int      `json:"rest_consul_publisher_max_routines,omitempty"`
}
