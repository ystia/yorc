package jconfig

type Configuration struct {
	Auth_url            string `json:"auth_url,omitempty"`
	Tenant_id           string `json:"tenant_id,omitempty"`
	Tenant_name         string `json:"tenant_name,omitempty"`
	User_name           string `json:"user_name,omitempty"`
	Password            string `json:"password,omitempty"`
	Region              string `json:"region,omitempty"`
	External_gateway    string `json:"external_gateway,omitempty"`
	Public_network_name string `json:"public_network_name,omitempty"`
	Prefix              string `json:"prefix,omitempty"`
	Keystone_user       string `json:"keystone_user,omitempty"`
	Keystone_password   string `json:"keystone_password,omitempty"`
	Keystone_tenant     string `json:"keystone_tenant,omitempty"`
	Keystone_url        string `json:"keystone_url,omitempty"`
	Consul_token        string `json:"consul_token,omitempty"`
	Consul_datacenter   string `json:"consul_datacenter,omitempty"`
	Path                string `json:"path,omitempty"`
}
