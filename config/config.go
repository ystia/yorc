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

// Package config defines configuration structures
package config

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/cast"
)

// DefaultConsulPubMaxRoutines is the default maximum number of parallel goroutines used to store keys/values in Consul
//
// See consulutil package for more details
const DefaultConsulPubMaxRoutines int = 500

// DefaultWorkersNumber is the default number of workers in the Yorc server
const DefaultWorkersNumber int = 3

// DefaultHTTPPort is the default port number for the HTTP REST API
const DefaultHTTPPort int = 8800

// DefaultHTTPAddress is the default listening address for the HTTP REST API
const DefaultHTTPAddress string = "0.0.0.0"

// DefaultPluginDir is the default path for the plugin directory
const DefaultPluginDir = "plugins"

// DefaultServerGracefulShutdownTimeout is the default timeout for a graceful shutdown of a Yorc server before exiting
const DefaultServerGracefulShutdownTimeout = 5 * time.Minute

//DefaultKeepOperationRemotePath is set to false by default in order to remove path created to store operation artifacts on nodes.
const DefaultKeepOperationRemotePath = false

//DefaultArchiveArtifacts is set to false by default.
// When ArchiveArtifacts is true, destination hosts need tar to be installed,
// to be able to unarchive artifacts.
const DefaultArchiveArtifacts = false

// DefaultCacheFacts is set to false by default, meaning ansible facts are not cached by default
const DefaultCacheFacts = false

// DefaultWfStepGracefulTerminationTimeout is the default timeout for a graceful termination of a workflow step during concurrent workflow step failure
const DefaultWfStepGracefulTerminationTimeout = 2 * time.Minute

// DefaultTFConsulPluginVersionConstraint is the default Terraform Consul plugin version
const DefaultTFConsulPluginVersionConstraint = "~> 2.1"

// DefaultTFAWSPluginVersionConstraint is the default Terraform AWS plugin version
const DefaultTFAWSPluginVersionConstraint = "~> 1.36"

// DefaultTFOpenStackPluginVersionConstraint is the default Terraform OpenStack plugin version
const DefaultTFOpenStackPluginVersionConstraint = "~> 1.9"

// DefaultTFGooglePluginVersionConstraint is the default Terraform Google plugin version
const DefaultTFGooglePluginVersionConstraint = "~> 1.18"

// Configuration holds config information filled by Cobra and Viper (see commands package for more information)
type Configuration struct {
	Ansible                          Ansible               `mapstructure:"ansible"`
	PluginsDirectory                 string                `mapstructure:"plugins_directory"`
	WorkingDirectory                 string                `mapstructure:"working_directory"`
	WorkersNumber                    int                   `mapstructure:"workers_number"`
	ServerGracefulShutdownTimeout    time.Duration         `mapstructure:"server_graceful_shutdown_timeout"`
	HTTPPort                         int                   `mapstructure:"http_port"`
	HTTPAddress                      string                `mapstructure:"http_address"`
	KeyFile                          string                `mapstructure:"key_file"`
	CertFile                         string                `mapstructure:"cert_file"`
	CAFile                           string                `mapstructure:"ca_file"`
	CAPath                           string                `mapstructure:"ca_path"`
	SSLVerify                        bool                  `mapstructure:"ssl_verify"`
	ResourcesPrefix                  string                `mapstructure:"resources_prefix"`
	Consul                           Consul                `mapstructure:"consul"`
	Telemetry                        Telemetry             `mapstructure:"telemetry"`
	Infrastructures                  map[string]DynamicMap `mapstructure:"infrastructures"`
	Vault                            DynamicMap            `mapstructure:"vault"`
	WfStepGracefulTerminationTimeout time.Duration         `mapstructure:"wf_step_graceful_termination_timeout"`
	ServerID                         string                `mapstructure:"server_id"`
	Terraform                        Terraform             `mapstructure:"terraform"`
}

// DockerSandbox holds the configuration for a docker sandbox
type DockerSandbox struct {
	Image      string   `mapstructure:"image"`
	Command    []string `mapstructure:"command"`
	Entrypoint []string `mapstructure:"entrypoint"`
	Env        []string `mapstructure:"env"`
}

// HostedOperations holds the configuration for operations executed on the orechestrator host (eg. with an operation_host equals to ORECHESTRATOR)
type HostedOperations struct {
	UnsandboxedOperationsAllowed bool           `mapstructure:"unsandboxed_operations_allowed"`
	DefaultSandbox               *DockerSandbox `mapstructure:"default_sandbox"`
}

// Format implements fmt.Formatter to provide a custom formatter.
func (ho HostedOperations) Format(s fmt.State, verb rune) {
	io.WriteString(s, "{UnsandboxedOperationsAllowed:")
	fmt.Fprint(s, ho.UnsandboxedOperationsAllowed)
	io.WriteString(s, " DefaultSandbox:")
	fmt.Fprintf(s, "%+v", ho.DefaultSandbox)
	io.WriteString(s, "}")
}

// Ansible configuration
type Ansible struct {
	UseOpenSSH              bool             `mapstructure:"use_openssh"`
	DebugExec               bool             `mapstructure:"debug"`
	ConnectionRetries       int              `mapstructure:"connection_retries"`
	OperationRemoteBaseDir  string           `mapstructure:"operation_remote_base_dir"`
	KeepOperationRemotePath bool             `mapstructure:"keep_operation_remote_path"`
	ArchiveArtifacts        bool             `mapstructure:"archive_artifacts"`
	CacheFacts              bool             `mapstructure:"cache_facts"`
	HostedOperations        HostedOperations `mapstructure:"hosted_operations"`
}

// Consul configuration
type Consul struct {
	Token          string `mapstructure:"token"`
	Datacenter     string `mapstructure:"datacenter"`
	Address        string `mapstructure:"address"`
	Key            string `mapstructure:"key_file"`
	Cert           string `mapstructure:"cert_file"`
	CA             string `mapstructure:"ca_cert"`
	CAPath         string `mapstructure:"ca_path"`
	SSL            bool   `mapstructure:"ssl"`
	SSLVerify      bool   `mapstructure:"ssl_verify"`
	PubMaxRoutines int    `mapstructure:"publisher_max_routines"`
}

// Telemetry holds the configuration for the telemetry service
type Telemetry struct {
	StatsdAddress           string `mapstructure:"statsd_address"`
	StatsiteAddress         string `mapstructure:"statsite_address"`
	PrometheusEndpoint      bool   `mapstructure:"expose_prometheus_endpoint"`
	ServiceName             string `mapstructure:"service_name"`
	DisableHostName         bool   `mapstructure:"disable_hostname"`
	DisableGoRuntimeMetrics bool   `mapstructure:"disable_go_runtime_metrics"`
}

// Terraform configuration
type Terraform struct {
	PluginsDir                       string `mapstructure:"plugins_dir"`
	ConsulPluginVersionConstraint    string `mapstructure:"consul_plugin_version_constraint"`
	AWSPluginVersionConstraint       string `mapstructure:"aws_plugin_version_constraint"`
	GooglePluginVersionConstraint    string `mapstructure:"google_plugin_version_constraint"`
	OpenStackPluginVersionConstraint string `mapstructure:"openstack_plugin_version_constraint"`
}

// DynamicMap allows to store configuration parameters that are not known in advance.
// This is particularly useful when configuration parameters may be defined in a plugin such for infrastructures.
//
// It has methods to automatically cast data to the desired type.
type DynamicMap map[string]interface{}

// Keys returns registered keys in the dynamic map
func (dm DynamicMap) Keys() []string {
	keys := make([]string, 0, len(dm))
	for k := range dm {
		keys = append(keys, k)
	}
	return keys
}

// Set sets a value for a given key
func (dm DynamicMap) Set(name string, value interface{}) {
	dm[name] = value
}

// IsSet checks if a given configuration key is defined
func (dm DynamicMap) IsSet(name string) bool {
	_, ok := dm[name]
	return ok
}

// Get returns the raw value of a given configuration key
func (dm DynamicMap) Get(name string) interface{} {
	return DefaultConfigTemplateResolver.ResolveValueWithTemplates(name, dm[name])
}

// GetString returns the value of the given key casted into a string.
// An empty string is returned if not found.
func (dm DynamicMap) GetString(name string) string {
	return cast.ToString(dm.Get(name))
}

// GetStringOrDefault returns the value of the given key casted into a string.
// The given default value is returned if not found or not a valid string.
func (dm DynamicMap) GetStringOrDefault(name, defaultValue string) string {
	if !dm.IsSet(name) {
		return defaultValue
	}
	if res, err := cast.ToStringE(dm.Get(name)); err == nil {
		return res
	}
	return defaultValue
}

// GetBool returns the value of the given key casted into a boolean.
// False is returned if not found.
func (dm DynamicMap) GetBool(name string) bool {
	return cast.ToBool(dm.Get(name))
}

// GetStringSlice returns the value of the given key casted into a slice of string.
// If the corresponding raw value is a string, it is split on comas.
// A nil or empty slice is returned if not found.
func (dm DynamicMap) GetStringSlice(name string) []string {
	val := dm.Get(name)
	switch v := val.(type) {
	case string:
		return strings.Split(v, ",")
	default:
		return cast.ToStringSlice(val)
	}
}

// GetInt returns the value of the given key casted into an int.
// 0 is returned if not found.
func (dm DynamicMap) GetInt(name string) int {
	return cast.ToInt(dm.Get(name))
}

// GetDuration returns the value of the given key casted into a Duration.
// A 0 duration is returned if not found.
func (dm DynamicMap) GetDuration(name string) time.Duration {
	return cast.ToDuration(dm.Get(name))
}

// DefaultConfigTemplateResolver is the default resolver for configuration templates
var DefaultConfigTemplateResolver TemplateResolver = &configTemplateResolver{}

// TemplateResolver allows to resolve templates in DynamicMap
type TemplateResolver interface {
	// SetTemplatesFunctions allows to define custom template functions
	SetTemplatesFunctions(fm template.FuncMap)
	// ResolveValueWithTemplates resolves a template
	ResolveValueWithTemplates(key string, value interface{}) interface{}
}

type configTemplateResolver struct {
	templateFunctions template.FuncMap
}

func (ctr *configTemplateResolver) SetTemplatesFunctions(fm template.FuncMap) {
	ctr.templateFunctions = fm
}

func (ctr *configTemplateResolver) ResolveValueWithTemplates(key string, value interface{}) interface{} {
	if value == nil {
		return nil
	}
	// templates should be strings
	s, err := cast.ToStringE(value)
	if err != nil {
		return value
	}
	// templates should contains delimiters
	if strings.Contains(s, "{{") && strings.Contains(s, "}}") {
		t, err := template.New("GenericConfigMap").Funcs(ctr.templateFunctions).Parse(s)
		if err != nil {
			log.Printf("[Warning] template parsing failed for key: %q value: %q: %v", key, s, err)
			return value
		}
		b := bytes.Buffer{}
		err = t.Execute(&b, nil)
		if err != nil {
			log.Printf("[Warning] template execution failed for key: %q value: %q: %v", key, s, err)
			return value
		}
		return b.String()
	}
	return value
}

// Client holds config information filled by Cobra and Viper (see commands package for more information)
// for the CLI (in short not the server) part of Yorc
type Client struct {
	YorcAPI       string `mapstructure:"yorc_api"`
	SSLEnabled    bool   `mapstructure:"ssl_enabled"`
	SkipTLSVerify bool   `mapstructure:"skip_tls_verify"`
	KeyFile       string `mapstructure:"key_file"`
	CertFile      string `mapstructure:"cert_file"`
	CAFile        string `mapstructure:"ca_file"`
	CAPath        string `mapstructure:"ca_path"`
}
