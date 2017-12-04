// Package config defines configuration structures
package config

import (
	"bytes"
	"strings"
	"text/template"
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
	OperationRemoteBaseDir           string
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
	Infrastructures                  map[string]GenericConfigMap
	Vault                            GenericConfigMap
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

// ResolveDynamicConfiguration resolves configuration parameters defined as
// Go templates.
func (c Configuration) ResolveDynamicConfiguration(fm template.FuncMap) {
	for _, gcm := range c.Infrastructures {
		gcm.resolveTemplates(fm)
	}
}

// GenericConfigMap allows to store configuration parameters that are not known in advance.
// This is particularly useful when configration parameters may be defined in a plugin such for infrastructures.
//
// It has methods to automatically cast data to the desired type.
type GenericConfigMap map[string]interface{}

// IsSet checks if a given configuration key is defined
func (gcm GenericConfigMap) IsSet(name string) bool {
	_, ok := gcm[name]
	return ok
}

// Get returns the raw value of a given configuration key
func (gcm GenericConfigMap) Get(name string) interface{} {
	return gcm[name]
}

// GetString returns the value of the given key casted into a string.
// An empty string is returned if not found.
func (gcm GenericConfigMap) GetString(name string) string {
	return cast.ToString(gcm[name])
}

// GetStringOrDefault returns the value of the given key casted into a string.
// The given default value is returned if not found or not a valid string.
func (gcm GenericConfigMap) GetStringOrDefault(name, defaultValue string) string {
	v, ok := gcm[name]
	if !ok {
		return defaultValue
	}
	if res, err := cast.ToStringE(v); err == nil {
		return res
	}
	return defaultValue
}

// GetBool returns the value of the given key casted into a boolean.
// False is returned if not found.
func (gcm GenericConfigMap) GetBool(name string) bool {
	return cast.ToBool(gcm[name])
}

// GetStringSlice returns the value of the given key casted into a slice of string.
// If the corresponding raw value is a string, it is  splited on comas.
// A nil or empty slice is returned if not found.
func (gcm GenericConfigMap) GetStringSlice(name string) []string {
	val := gcm[name]
	switch v := val.(type) {
	case string:
		return strings.Split(v, ",")
	default:
		return cast.ToStringSlice(gcm[name])
	}
}

// GetInt returns the value of the given key casted into an int.
// 0 is returned if not found.
func (gcm GenericConfigMap) GetInt(name string) int {
	return cast.ToInt(gcm[name])
}

// GetDuration returns the value of the given key casted into a Duration.
// A 0 duration is returned if not found.
func (gcm GenericConfigMap) GetDuration(name string) time.Duration {
	return cast.ToDuration(gcm[name])
}

func (gcm GenericConfigMap) resolveTemplates(fm template.FuncMap) {
	// TODO here we resolve data in oneshot for all
	// but it may be more intersting to resolve it only when needed
	// as a secret may expire or be renewed.
	for key, value := range gcm {
		s, err := cast.ToStringE(value)
		if err != nil {
			continue
		}

		t, err := template.New("GenericConfigMap").Funcs(fm).Parse(s)
		if err != nil {
			continue
		}
		b := bytes.Buffer{}
		err = t.Execute(&b, nil)
		if err != nil {
			continue
		}
		gcm[key] = b.String()
	}
}
