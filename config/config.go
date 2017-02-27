// Package config defines configuration structures
package config

// DefaultConsulPubMaxRoutines is the default maximum number of parallel goroutines used to store keys/values in Consul
//
// See consulutil package for more details
const DefaultConsulPubMaxRoutines int = 500

// DefaultWorkersNumber is the default number of workers in the Janus server
const DefaultWorkersNumber int = 3

// Configuration holds config information filled by Cobra and Viper (see commands package for more information)
type Configuration struct {
	WorkingDirectory        string   `json:"working_directory,omitempty"`
	WorkersNumber           int      `json:"workers_number,omitempty"`
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
