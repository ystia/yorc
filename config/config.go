// Package config defines configuration structures
package config

import (
	"strings"
	"time"

	"github.com/spf13/cast"
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
	KeepOperationRemotePath       bool          `json:"keep_operation_remote_path,omitempty"`
	ResourcesPrefix               string        `json:"os_prefix,omitempty"`
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
	Telemetry                     Telemetry     `json:"telemetry,omitempty"`
	Infrastructures               map[string]InfrastructureConfig
}

// Telemetry holds the configuration for the telemetry service
type Telemetry struct {
	StatsdAddress           string `json:"statsd_address,omitempty"`
	StatsiteAddress         string `json:"statsite_address,omitempty"`
	PrometheusEndpoint      bool
	ServiceName             string `json:"service_name,omitempty"`
	DisableHostName         bool
	DisableGoRuntimeMetrics bool
}

// InfrastructureConfig parameters for a given infrastructure.
//
// It has methods to automatically cast data to the desired type.
type InfrastructureConfig map[string]interface{}

// Get returns the raw value of a given configuration key
func (ic InfrastructureConfig) Get(name string) interface{} {
	return ic[name]
}

// GetString returns the value of the given key casted into a string.
// An empty string is returned if not found.
func (ic InfrastructureConfig) GetString(name string) string {
	return cast.ToString(ic[name])
}

// GetStringOrDefault returns the value of the given key casted into a string.
// The given default value is returned if not found.
func (ic InfrastructureConfig) GetStringOrDefault(name, defaultValue string) string {
	if res := ic.GetString(name); res != "" {
		return res
	}
	return defaultValue
}

// GetBool returns the value of the given key casted into a boolean.
// False is returned if not found.
func (ic InfrastructureConfig) GetBool(name string) bool {
	return cast.ToBool(ic[name])
}

// GetStringSlice returns the value of the given key casted into a slice of string.
// If the corresponding raw value is a string, it is  splited on comas.
// A nil or empty slice is returned if not found.
func (ic InfrastructureConfig) GetStringSlice(name string) []string {
	val := ic[name]
	switch v := val.(type) {
	case string:
		return strings.Split(v, ",")
	default:
		return cast.ToStringSlice(ic[name])
	}
}
