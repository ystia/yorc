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

//DefaultKeepOperationRemotePath is set to true by default in order to remove path created to store operation artifacts on nodes.
const DefaultKeepOperationRemotePath = false

// DefaultWfStepGracefulTerminationTimeout is the default timeout for a graceful termination of a workflow step during concurrent workflow step failure
const DefaultWfStepGracefulTerminationTimeout = 2 * time.Minute

// Configuration holds config information filled by Cobra and Viper (see commands package for more information)
type Configuration struct {
	Ansible                          Ansible
	PluginsDirectory                 string
	WorkingDirectory                 string
	WorkersNumber                    int
	ServerGracefulShutdownTimeout    time.Duration
	HTTPPort                         int
	HTTPAddress                      string
	KeyFile                          string
	CertFile                         string
	CAFile                           string
	CAPath 							 string
	SSLEnabled 						 bool
	SSLVerify                        bool
	ResourcesPrefix                  string
	Consul                           Consul
	Telemetry                        Telemetry
	Infrastructures                  map[string]DynamicMap
	Vault                            DynamicMap
	WfStepGracefulTerminationTimeout time.Duration
}

// Ansible configuration
type Ansible struct {
	UseOpenSSH              bool
	DebugExec               bool
	ConnectionRetries       int
	OperationRemoteBaseDir  string
	KeepOperationRemotePath bool
}

// Consul configuration
type Consul struct {
	Token          string
	Datacenter     string
	Address        string
	Key            string
	Cert           string
	CA             string
	CAPath         string
	SSL            bool
	SSLVerify      bool
	PubMaxRoutines int
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
