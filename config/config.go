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

// DefaultConsulTLSHandshakeTimeout is the default maximum duration to wait for
// a TLS handshake.
// See consulutil package for more details
const DefaultConsulTLSHandshakeTimeout = 50 * time.Second

// DefaultWorkersNumber is the default number of workers in the Yorc server
const DefaultWorkersNumber int = 30

// DefaultHTTPPort is the default port number for the HTTP REST API
const DefaultHTTPPort int = 8800

// DefaultHTTPAddress is the default listening address for the HTTP REST API
const DefaultHTTPAddress string = "0.0.0.0"

// DefaultPluginDir is the default path for the plugin directory
const DefaultPluginDir = "plugins"

// DefaultPluginsHealthCheckTimeInterval is the default time interval for plugins health check
const DefaultPluginsHealthCheckTimeInterval = 1 * time.Minute

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

// DefaultPurgedDeploymentsEvictionTimeout is the default timeout after which final events and logs for a purged deployment are actually deleted.
const DefaultPurgedDeploymentsEvictionTimeout = 30 * time.Minute

// DefaultAnsibleJobMonInterval is the default monitoring interval for Jobs handled by Ansible
const DefaultAnsibleJobMonInterval = 15 * time.Second

// DefaultTasksDispatcherLongPollWaitTime is the default wait time when long polling for executions tasks to dispatch to workers
const DefaultTasksDispatcherLongPollWaitTime = 1 * time.Minute

// DefaultTasksDispatcherLockWaitTime is the default wait time for acquiring a lock for an execution task
const DefaultTasksDispatcherLockWaitTime = 50 * time.Millisecond

// DefaultTasksDispatcherMetricsRefreshTime is the default refresh time for the Tasks dispatcher metrics
const DefaultTasksDispatcherMetricsRefreshTime = 5 * time.Minute

// DefaultUpgradesConcurrencyLimit is the default limit of concurrency used in Upgrade processes
const DefaultUpgradesConcurrencyLimit = 1000

// DefaultSSHConnectionTimeout is the default timeout for SSH connections
const DefaultSSHConnectionTimeout = 10 * time.Second

// DefaultSSHConnectionRetryBackoff is the default duration before retring connect for SSH connections
const DefaultSSHConnectionRetryBackoff = 1 * time.Second

// DefaultSSHConnectionMaxRetries is the default maximum number of retries before giving up (number of attempts is number of retries + 1)
const DefaultSSHConnectionMaxRetries uint64 = 3

// Configuration holds config information filled by Cobra and Viper (see commands package for more information)
type Configuration struct {
	Ansible                          Ansible       `yaml:"ansible,omitempty" mapstructure:"ansible"`
	PluginsDirectory                 string        `yaml:"plugins_directory,omitempty" mapstructure:"plugins_directory"`
	PluginsHealthCheckTimeInterval   time.Duration `yaml:"plugins_health_check_time_interval,omitempty" mapstructure:"plugins_health_check_time_interval"`
	WorkingDirectory                 string        `yaml:"working_directory,omitempty" mapstructure:"working_directory"`
	WorkersNumber                    int           `yaml:"workers_number,omitempty" mapstructure:"workers_number"`
	ServerGracefulShutdownTimeout    time.Duration `yaml:"server_graceful_shutdown_timeout,omitempty" mapstructure:"server_graceful_shutdown_timeout"`
	HTTPPort                         int           `yaml:"http_port,omitempty" mapstructure:"http_port"`
	HTTPAddress                      string        `yaml:"http_address,omitempty" mapstructure:"http_address"`
	KeyFile                          string        `yaml:"key_file,omitempty" mapstructure:"key_file"`
	CertFile                         string        `yaml:"cert_file,omitempty" mapstructure:"cert_file"`
	CAFile                           string        `yaml:"ca_file,omitempty" mapstructure:"ca_file"`
	CAPath                           string        `yaml:"ca_path,omitempty" mapstructure:"ca_path"`
	SSLVerify                        bool          `yaml:"ssl_verify,omitempty" mapstructure:"ssl_verify"`
	ResourcesPrefix                  string        `yaml:"resources_prefix,omitempty" mapstructure:"resources_prefix"`
	Consul                           Consul        `yaml:"consul,omitempty" mapstructure:"consul"`
	Telemetry                        Telemetry     `yaml:"telemetry,omitempty" mapstructure:"telemetry"`
	LocationsFilePath                string        `yaml:"locations_file_path,omitempty" mapstructure:"locations_file_path"`
	Vault                            DynamicMap    `yaml:"vault,omitempty" mapstructure:"vault"`
	WfStepGracefulTerminationTimeout time.Duration `yaml:"wf_step_graceful_termination_timeout,omitempty" mapstructure:"wf_step_graceful_termination_timeout"`
	PurgedDeploymentsEvictionTimeout time.Duration `yaml:"purged_deployments_eviction_timeout,omitempty" mapstructure:"purged_deployments_eviction_timeout"`
	ServerID                         string        `yaml:"server_id,omitempty" mapstructure:"server_id"`
	Terraform                        Terraform     `yaml:"terraform,omitempty" mapstructure:"terraform"`
	DisableSSHAgent                  bool          `yaml:"disable_ssh_agent,omitempty" mapstructure:"disable_ssh_agent"`
	Tasks                            Tasks         `yaml:"tasks,omitempty" mapstructure:"tasks"`
	Storage                          Storage       `yaml:"storage,omitempty" mapstructure:"storage"`
	UpgradeConcurrencyLimit          int           `yaml:"concurrency_limit_for_upgrades,omitempty" mapstructure:"concurrency_limit_for_upgrades"`
	SSHConnectionTimeout             time.Duration `yaml:"ssh_connection_timeout,omitempty" mapstructure:"ssh_connection_timeout"`
	SSHConnectionRetryBackoff        time.Duration `yaml:"ssh_connection_retry_backoff,omitempty" mapstructure:"ssh_connection_retry_backoff"`
	SSHConnectionMaxRetries          uint64        `yaml:"ssh_connection_max_retries,omitempty" mapstructure:"ssh_connection_max_retries"`
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
	UseOpenSSH              bool                         `yaml:"use_openssh,omitempty" mapstructure:"use_openssh" json:"use_open_ssh,omitempty"`
	DebugExec               bool                         `yaml:"debug,omitempty" mapstructure:"debug" json:"debug_exec,omitempty"`
	ConnectionRetries       int                          `yaml:"connection_retries,omitempty" mapstructure:"connection_retries" json:"connection_retries,omitempty"`
	OperationRemoteBaseDir  string                       `yaml:"operation_remote_base_dir,omitempty" mapstructure:"operation_remote_base_dir" json:"operation_remote_base_dir,omitempty"`
	KeepOperationRemotePath bool                         `yaml:"keep_operation_remote_path,omitempty" mapstructure:"keep_operation_remote_path" json:"keep_operation_remote_path,omitempty"`
	KeepGeneratedRecipes    bool                         `yaml:"keep_generated_recipes,omitempty" mapstructure:"keep_generated_recipes" json:"keep_generated_recipes,omitempty"`
	ArchiveArtifacts        bool                         `yaml:"archive_artifacts,omitempty" mapstructure:"archive_artifacts" json:"archive_artifacts,omitempty"`
	CacheFacts              bool                         `yaml:"cache_facts,omitempty" mapstructure:"cache_facts" json:"cache_facts,omitempty"`
	HostedOperations        HostedOperations             `yaml:"hosted_operations,omitempty" mapstructure:"hosted_operations" json:"hosted_operations,omitempty"`
	JobsChecksPeriod        time.Duration                `yaml:"job_monitoring_time_interval,omitempty" mapstructure:"job_monitoring_time_interval" json:"job_monitoring_time_interval,omitempty"`
	Config                  map[string]map[string]string `yaml:"config,omitempty" mapstructure:"config"`
	Inventory               map[string][]string          `yaml:"inventory,omitempty" mapstructure:"inventory"`
}

// Consul configuration
type Consul struct {
	Token               string        `yaml:"token,omitempty" mapstructure:"token"`
	Datacenter          string        `yaml:"datacenter,omitempty" mapstructure:"datacenter"`
	Address             string        `yaml:"address,omitempty" mapstructure:"address"`
	Key                 string        `yaml:"key_file,omitempty" mapstructure:"key_file"`
	Cert                string        `yaml:"cert_file,omitempty" mapstructure:"cert_file"`
	CA                  string        `yaml:"ca_cert,omitempty" mapstructure:"ca_cert"`
	CAPath              string        `yaml:"ca_path,omitempty" mapstructure:"ca_path"`
	SSL                 bool          `yaml:"ssl,omitempty" mapstructure:"ssl"`
	SSLVerify           bool          `yaml:"ssl_verify,omitempty" mapstructure:"ssl_verify"`
	PubMaxRoutines      int           `yaml:"publisher_max_routines,omitempty" mapstructure:"publisher_max_routines"`
	TLSHandshakeTimeout time.Duration `yaml:"tls_handshake_timeout,omitempty" mapstructure:"tls_handshake_timeout"`
}

// Telemetry holds the configuration for the telemetry service
type Telemetry struct {
	StatsdAddress           string `yaml:"statsd_address,omitempty" mapstructure:"statsd_address"`
	StatsiteAddress         string `yaml:"statsite_address,omitempty" mapstructure:"statsite_address"`
	PrometheusEndpoint      bool   `yaml:"expose_prometheus_endpoint,omitempty" mapstructure:"expose_prometheus_endpoint"`
	ServiceName             string `yaml:"service_name,omitempty" mapstructure:"service_name"`
	DisableHostName         bool   `yaml:"disable_hostname,omitempty" mapstructure:"disable_hostname"`
	DisableGoRuntimeMetrics bool   `yaml:"disable_go_runtime_metrics,omitempty" mapstructure:"disable_go_runtime_metrics"`
}

// Terraform configuration
type Terraform struct {
	PluginsDir                       string `yaml:"plugins_dir,omitempty" mapstructure:"plugins_dir"`
	ConsulPluginVersionConstraint    string `yaml:"consul_plugin_version_constraint,omitempty" mapstructure:"consul_plugin_version_constraint"`
	AWSPluginVersionConstraint       string `yaml:"aws_plugin_version_constraint,omitempty" mapstructure:"aws_plugin_version_constraint"`
	GooglePluginVersionConstraint    string `yaml:"google_plugin_version_constraint,omitempty" mapstructure:"google_plugin_version_constraint"`
	OpenStackPluginVersionConstraint string `yaml:"openstack_plugin_version_constraint,omitempty" mapstructure:"openstack_plugin_version_constraint"`
	KeepGeneratedFiles               bool   `yaml:"keep_generated_files,omitempty" mapstructure:"keep_generated_files"`
}

// Tasks processing configuration
type Tasks struct {
	Dispatcher Dispatcher `yaml:"dispatcher,omitempty" mapstructure:"dispatcher" json:"dispatcher,omitempty"`
}

// Dispatcher configuration
type Dispatcher struct {
	LongPollWaitTime   time.Duration `yaml:"long_poll_wait_time,omitempty" mapstructure:"long_poll_wait_time" json:"long_poll_wait_time,omitempty"`
	LockWaitTime       time.Duration `yaml:"lock_wait_time,omitempty" mapstructure:"lock_wait_time" json:"lock_wait_time,omitempty"`
	MetricsRefreshTime time.Duration `yaml:"metrics_refresh_time,omitempty" mapstructure:"metrics_refresh_time" json:"metrics_refresh_time,omitempty"`
}

// Storage configuration
type Storage struct {
	Reset             bool       `yaml:"reset,omitempty" json:"reset,omitempty" mapstructure:"reset"`
	Stores            []Store    `yaml:"stores,omitempty" json:"stores,omitempty" mapstructure:"stores"`
	DefaultProperties DynamicMap `yaml:"default_properties,omitempty" json:"default_properties,omitempty" mapstructure:"default_properties"`
}

// Store configuration
type Store struct {
	Name                  string     `yaml:"name" json:"name" mapstructure:"name"`
	MigrateDataFromConsul bool       `yaml:"migrate_data_from_consul,omitempty" json:"migrate_data_from_consul,omitempty"  mapstructure:"migrate_data_from_consul"`
	Implementation        string     `yaml:"implementation" json:"implementation" mapstructure:"implementation"` // not an enum as it may be extended by plugins
	Types                 []string   `yaml:"types" json:"types" mapstructure:"types"`
	Properties            DynamicMap `yaml:"properties,omitempty" json:"properties,omitempty" mapstructure:"properties"`
}

// DynamicMap allows to store configuration parameters that are not known in advance.
// This is particularly useful when configuration parameters may be defined in a plugin such for locations.
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

// GetIntOrDefault returns the value of the given key casted into an int.
// The given default value is returned if not found.
func (dm DynamicMap) GetIntOrDefault(name string, defaultValue int) int {
	if !dm.IsSet(name) {
		return defaultValue
	}
	return cast.ToInt(dm.Get(name))
}

// GetInt64OrDefault returns the value of the given key casted into an int64.
// The given default value is returned if not found.
func (dm DynamicMap) GetInt64OrDefault(name string, defaultValue int64) int64 {
	if !dm.IsSet(name) {
		return defaultValue
	}
	return cast.ToInt64(dm.Get(name))
}

// GetInt64 returns the value of the given key casted into an int64.
// 0 is returned if not found.
func (dm DynamicMap) GetInt64(name string) int64 {
	return cast.ToInt64(dm.Get(name))
}

// GetUint64OrDefault returns the value of the given key casted into an uint64.
// The given default value is returned if not found.
func (dm DynamicMap) GetUint64OrDefault(name string, defaultValue uint64) uint64 {
	if !dm.IsSet(name) {
		return defaultValue
	}
	return cast.ToUint64(dm.Get(name))
}

// GetUint64 returns the value of the given key casted into an uint64.
// 0 is returned if not found.
func (dm DynamicMap) GetUint64(name string) uint64 {
	return cast.ToUint64(dm.Get(name))
}

// GetDuration returns the value of the given key casted into a Duration.
// A 0 duration is returned if not found.
func (dm DynamicMap) GetDuration(name string) time.Duration {
	return cast.ToDuration(dm.Get(name))
}

// GetDurationOrDefault returns the value of the given key casted into a Duration.
// The given default value is returned if not found.
func (dm DynamicMap) GetDurationOrDefault(name string, defaultValue time.Duration) time.Duration {
	if !dm.IsSet(name) {
		return defaultValue
	}
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
	// Disable allows to disable configuration templates usage
	Disable()
	// Enable allows to enable configuration templates usage
	Enable()
}

type configTemplateResolver struct {
	templateFunctions template.FuncMap
	disable           bool
}

// SetTemplatesFunctions allows to set configuration templates
func (ctr *configTemplateResolver) SetTemplatesFunctions(fm template.FuncMap) {
	ctr.templateFunctions = fm
}

// Disable allows to disable configuration templates usage
func (ctr *configTemplateResolver) Disable() {
	ctr.disable = true
}

// Enable allows to enable configuration templates usage
func (ctr *configTemplateResolver) Enable() {
	ctr.disable = false
}

// ResolveValueWithTemplates returns a value corresponding to some template if the templates usage is not disabled
func (ctr *configTemplateResolver) ResolveValueWithTemplates(key string, value interface{}) interface{} {
	if value == nil {
		return nil
	}
	// if template disabled return the value as it is
	if ctr.disable {
		return value
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
