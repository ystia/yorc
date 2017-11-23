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

// DefaultWfStepGracefulTerminationTimeout is the default timeout for a graceful termination of a workflow step during concurrent workflow step failure
const DefaultWfStepGracefulTerminationTimeout = 2 * time.Minute

// Configuration holds config information filled by Cobra and Viper (see commands package for more information)
type Configuration struct {
	AnsibleUseOpenSSH                bool
	AnsibleDebugExec                 bool
	AnsibleConnectionRetries         int
	PluginsDirectory                 string
	WorkingDirectory                 string
	WorkersNumber                    int
	ServerGracefulShutdownTimeout    time.Duration
	HTTPPort                         int
	HTTPAddress                      string
	KeyFile                          string
	CertFile                         string
	KeepOperationRemotePath          bool
	ResourcesPrefix                  string
	ConsulToken                      string
	ConsulDatacenter                 string
	ConsulAddress                    string
	ConsulKey                        string
	ConsulCert                       string
	ConsulCA                         string
	ConsulCAPath                     string
	ConsulSSL                        bool
	ConsulSSLVerify                  bool
	ConsulPubMaxRoutines             int
	Telemetry                        Telemetry
	Infrastructures                  map[string]InfrastructureConfig
	WfStepGracefulTerminationTimeout time.Duration
}

// Telemetry holds the configuration for the telemetry service
type Telemetry struct {
	StatsdAddress           string
	StatsiteAddress         string
	PrometheusEndpoint      bool
	ServiceName             string
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
