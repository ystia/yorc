// Package config defines configuration structures
package config

import (
	"time"
)

// DefaultConsulPubMaxRoutines is the default maximum number of parallel goroutines used to store keys/values in Consul
//
// See consulutil package for more details
const DefaultConsulPubMaxRoutines int = 500

// DefaultWorkersNumber is the default number of workers in the Janus server
const DefaultWorkersNumber int = 3

// DefaultHTTPPort is the default port number for the HTTP REST API
const DefaultHTTPPort int = 8800

// DefaultHTTPAddress is the default listening address for the HTTP REST API
const DefaultHTTPAddress string = "0.0.0.0"

// DefaultPluginDir is the default path for the plugin directory
const DefaultPluginDir = "plugins"

// DefaultServerGracefulShutdownTimeout is the default timeout for a graceful shutdown of a Janus server before exiting
const DefaultServerGracefulShutdownTimeout = 5 * time.Minute

//DefaultKeepOperationRemotePath is set to true by default in order to remove path created to store operation artifacts on nodes.
const DefaultKeepOperationRemotePath = false

// Configuration holds config information filled by Cobra and Viper (see commands package for more information)
type Configuration struct {
	AnsibleUseOpenSSH             bool
	AnsibleDebugExec              bool
	PluginsDirectory              string        `json:"plugins_directory,omitempty"`
	WorkingDirectory              string        `json:"working_directory,omitempty"`
	WorkersNumber                 int           `json:"workers_number,omitempty"`
	ServerGracefulShutdownTimeout time.Duration `json:"server_graceful_shutdown_timeout"`
	HTTPPort                      int           `json:"http_port,omitempty"`
	HTTPAddress                   string        `json:"http_address,omitempty"`
	KeyFile                       string        `json:"key_file,omitempty"`
	CertFile                      string        `json:"cert_file,omitempty"`
	OSAuthURL                     string        `json:"os_auth_url,omitempty"`
	OSTenantID                    string        `json:"os_tenant_id,omitempty"`
	OSTenantName                  string        `json:"os_tenant_name,omitempty"`
	OSUserName                    string        `json:"os_user_name,omitempty"`
	OSPassword                    string        `json:"os_password,omitempty"`
	OSRegion                      string        `json:"os_region,omitempty"`
	OSInsecure                    string        `json:"os_insecure,omitempty"`
	OSCACert                      string        `json:"os_cacert_file,omitempty"`
	OSCert                        string        `json:"os_cert,omitempty"`
	OSKey                         string        `json:"os_key,omitempty"`
	KeepOperationRemotePath       bool          `json:"keep_operation_remote_path,omitempty"`
	ResourcesPrefix               string        `json:"os_prefix,omitempty"`
	OSPrivateNetworkName          string        `json:"os_private_network_name,omitempty"`
	OSPublicNetworkName           string        `json:"os_public_network_name,omitempty"`
	OSDefaultSecurityGroups       []string      `json:"os_default_security_groups,omitempty"`
	ConsulToken                   string        `json:"consul_token,omitempty"`
	ConsulDatacenter              string        `json:"consul_datacenter,omitempty"`
	ConsulAddress                 string        `json:"consul_address,omitempty"`
	ConsulKey                     string        `json:"consul_key_file,omitempty"`
	ConsulCert                    string        `json:"consul_cert_file,omitempty"`
	ConsulCA                      string        `json:"consul_ca_cert,omitempty"`
	ConsulCAPath                  string        `json:"consul_ca_path,omitempty"`
	ConsulSSL                     bool          `json:"consul_ssl,omitempty"`
	ConsulSSLVerify               bool          `json:"consul_ssl_verify,omitempty"`
	ConsulPubMaxRoutines          int           `json:"rest_consul_publisher_max_routines,omitempty"`
	KubemasterIP                  string        `json:"kube_ip,omitpempty"`
}
