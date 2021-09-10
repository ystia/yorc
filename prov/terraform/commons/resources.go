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

package commons

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/provutil"
	"github.com/ystia/yorc/v4/helper/sshutil"
	"github.com/ystia/yorc/v4/log"
)

const (
	// DefaultSSHPrivateKeyFilePath is the default SSH private Key file path
	// used to connect to provisioned resources
	DefaultSSHPrivateKeyFilePath = "~/.ssh/yorc.pem"
	// NullPluginVersionConstraint is the Terraform null plugin version constraint
	NullPluginVersionConstraint = "~> 1.0"
	// DefaultConsulProviderAddress is Default Address to use  for Terraform Consul Provider
	DefaultConsulProviderAddress = "127.0.0.1:8500"
)

// An Infrastructure is the top-level element of a Terraform infrastructure definition
type Infrastructure struct {
	Terraform map[string]interface{} `json:"terraform,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Variable  map[string]interface{} `json:"variable,omitempty"`
	Provider  map[string]interface{} `json:"provider,omitempty"`
	Resource  map[string]interface{} `json:"resource,omitempty"`
	Output    map[string]*Output     `json:"output,omitempty"`
}

// The ConsulKeys can be used as 'resource' to writes or 'data' to read sets of individual values into Consul.
type ConsulKeys struct {
	Resource
	Datacenter string      `json:"datacenter,omitempty"`
	Token      string      `json:"token,omitempty"`
	Keys       []ConsulKey `json:"key"`
}

// A Resource is the base type for terraform resources
type Resource struct {
	Count        int                      `json:"count,omitempty"`
	DependsOn    []string                 `json:"depends_on,omitempty"`
	Connection   *Connection              `json:"connection,omitempty"`
	Provisioners []map[string]interface{} `json:"provisioner,omitempty"`
}

// A ConsulKey can be used in a ConsulKeys 'resource' to writes or a ConsulKeys 'data' to read an individual Key/Value pair into Consul
type ConsulKey struct {
	Path string `json:"path"`

	// Should only be use in datasource (read) mode, this is the name to use to access this key within the terraform interpolation syntax
	Name string `json:"name,omitempty"`
	// Should only be use in datasource (read) mode, default value if the key is not found

	Default string `json:"default,omitempty"`
	// Should only be use in resource (write) mode, the value to set to the key

	Value string `json:"value,omitempty"`

	// Should only be use in resource (write) mode, deletes the key
	Delete bool `json:"delete,omitempty"`
}

// The RemoteExec provisioner invokes a script on a remote resource after it is created.
//
// The remote-exec provisioner supports both ssh and winrm type connections.
type RemoteExec struct {
	Connection *Connection `json:"connection,omitempty"`
	Inline     []string    `json:"inline,omitempty"`
	Script     string      `json:"script,omitempty"`
	Scripts    []string    `json:"scripts,omitempty"`
}

// LocalExec allows to invoke a local executable after a resource is created. This invokes a process on the machine running Terraform, not on the resource
type LocalExec struct {
	Command     string                 `json:"command"`
	WorkingDir  string                 `json:"working_dir,omitempty"`
	Interpreter string                 `json:"interpreter,omitempty"`
	Environment map[string]interface{} `json:"environment,omitempty"`
}

// A Connection allows to overwrite the way Terraform connects to a resource
type Connection struct {
	ConnType   string `json:"type,omitempty"`
	User       string `json:"user,omitempty"`
	Password   string `json:"password,omitempty"`
	Host       string `json:"host,omitempty"`
	Port       string `json:"port,omitempty"`
	Timeout    string `json:"timeout,omitempty"` // defaults to "5m"
	PrivateKey string `json:"private_key,omitempty"`

	Agent bool `json:"agent,omitempty"`

	// The following values are only supported by ssh
	BastionHost       string `json:"bastion_host,omitempty"`
	BastionPort       string `json:"bastion_port,omitempty"`
	BastionUser       string `json:"bastion_user,omitempty"`
	BastionPassword   string `json:"bastion_password,omitempty"`
	BastionPrivateKey string `json:"bastion_private_key,omitempty"`
}

// An Output allows to define a terraform output value
type Output struct {
	// Value is the value of the output. This can be a string, list, or map.
	// This usually includes an interpolation since outputs that are static aren't usually useful.
	Value     interface{} `json:"value"`
	Sensitive bool        `json:"sensitive,omitempty"`
}

// AddResource allows to add a Resource to a defined Infrastructure
func AddResource(infrastructure *Infrastructure, resourceType, resourceName string, resource interface{}) {
	if len(infrastructure.Resource) != 0 {
		if infrastructure.Resource[resourceType] != nil && len(infrastructure.Resource[resourceType].(map[string]interface{})) != 0 {
			resourcesMap := infrastructure.Resource[resourceType].(map[string]interface{})
			resourcesMap[resourceName] = resource
		} else {
			resourcesMap := make(map[string]interface{})
			resourcesMap[resourceName] = resource
			infrastructure.Resource[resourceType] = resourcesMap
		}

	} else {
		resourcesMap := make(map[string]interface{})
		infrastructure.Resource = resourcesMap
		resourcesMap = make(map[string]interface{})
		resourcesMap[resourceName] = resource
		infrastructure.Resource[resourceType] = resourcesMap
	}
}

// AddOutput allows to add an Output to a defined Infrastructure
func AddOutput(infrastructure *Infrastructure, outputName string, output *Output) {
	if infrastructure.Output == nil {
		infrastructure.Output = make(map[string]*Output)
	}
	infrastructure.Output[outputName] = output
}

// GetConnInfoFromEndpointCredentials allow to retrieve user and private key path for connection needs from endpoint credentials
func GetConnInfoFromEndpointCredentials(ctx context.Context, deploymentID, nodeName string) (string, *sshutil.PrivateKey, error) {
	user, err := deployments.GetCapabilityPropertyValue(ctx, deploymentID, nodeName, "endpoint", "credentials", "user")
	if err != nil {
		return "", nil, err
	} else if user == nil || user.RawString() == "" {
		return "", nil, errors.Errorf("Missing mandatory parameter 'user' node type for %s", nodeName)
	}
	keys, err := sshutil.GetKeysFromCredentialsAttribute(ctx, deploymentID, nodeName, "0", "endpoint")
	if err != nil {
		return "", nil, err
	}

	var pk *sshutil.PrivateKey
	if len(keys) == 0 {
		pk, err = sshutil.GetDefaultKey()
		if err != nil {
			return "", nil, err
		}
		keys["default"] = pk
	} else {
		pk = sshutil.SelectPrivateKeyOnName(keys, false)

	}

	keysList := make([]*sshutil.PrivateKey, 0, len(keys))
	for _, k := range keys {
		keysList = append(keysList, k)
	}

	addKeysToContextualSSHAgent(ctx, keysList)

	return user.RawString(), pk, nil
}

// AddConnectionCheckResource builds a null specific resource to check SSH connection with SSH key passed via env variable
func AddConnectionCheckResource(ctx context.Context, deploymentID, nodeName string, infrastructure *Infrastructure,
	user string, privateKey *sshutil.PrivateKey, accessIP, resourceName string, env *[]string) error {
	// Define private_key variable
	infrastructure.Variable = make(map[string]interface{})
	infrastructure.Variable["private_key"] = struct{}{}

	// Add env TF variable for private key
	*env = append(*env, fmt.Sprintf("%s=%s", "TF_VAR_private_key", string(privateKey.Content)))

	conn := &Connection{
		User:       user,
		Host:       accessIP,
		PrivateKey: "${var.private_key}",
		Timeout:    "15m",
	}

	bast, err := provutil.GetInstanceBastionHost(ctx, deploymentID, nodeName)
	if err != nil {
		return err
	}

	if bast != nil {
		conn.BastionHost = bast.Host
		conn.BastionPort = bast.Port
		conn.BastionUser = bast.User
		conn.BastionPassword = bast.Password

		var bastPk *sshutil.PrivateKey
		if bast != nil {
			bastPk = sshutil.SelectPrivateKeyOnName(bast.PrivateKeys, false)
			if bastPk == nil {
				// If no key is explicitly defined, use the same as for the instance.
				bastPk = privateKey
			}
		}

		if bastPk != nil {
			infrastructure.Variable["bastion_private_key"] = struct{}{}
			*env = append(*env, fmt.Sprintf("TF_VAR_bastion_private_key=%s", string(bastPk.Content)))
			conn.BastionPrivateKey = "${var.bastion_private_key}"
		} else if bast.Password == "" {
			return errors.New("bastion host configuration is missing credentials")
		}
	}

	// Build null Resource
	nullResource := Resource{}
	re := RemoteExec{Inline: []string{`echo "connected"`}, Connection: conn}
	nullResource.Provisioners = make([]map[string]interface{}, 0)
	provMap := make(map[string]interface{})
	provMap["remote-exec"] = re
	nullResource.Provisioners = append(nullResource.Provisioners, provMap)

	AddResource(infrastructure, "null_resource", resourceName+"-ConnectionCheck", &nullResource)
	return nil
}

// GetBackendConfiguration returns the Terraform Backend configuration
// to store the state in the Consul KV store at a given path
func GetBackendConfiguration(path string, cfg config.Configuration) map[string]interface{} {

	consulAddress := DefaultConsulProviderAddress
	if cfg.Consul.Address != "" {
		consulAddress = cfg.Consul.Address
	}
	consulScheme := "http"
	if cfg.Consul.SSL {
		consulScheme = "https"
	}

	log.Debugf("Terraform Consul backend configuration address %s", consulAddress)
	log.Debugf("Terraform Consul backend configuration scheme %s", consulScheme)
	log.Debugf("Terraform Consul backend configuration use ca file %s", cfg.Consul.CA)
	log.Debugf("Terraform Consul backend configuration cert file %s", cfg.Consul.Key)
	log.Debugf("Terraform Consul backend configuration key file %s", cfg.Consul.Cert)

	return map[string]interface{}{
		"backend": map[string]interface{}{
			"consul": map[string]interface{}{
				"path":      path,
				"address":   consulAddress,
				"scheme":    consulScheme,
				"ca_file":   cfg.Consul.CA,
				"cert_file": cfg.Consul.Cert,
				"key_file":  cfg.Consul.Key,
			},
		},
	}
}

// GetConsulProviderfiguration returns the Terraform Consul Provider configuration
func GetConsulProviderfiguration(cfg config.Configuration) map[string]interface{} {

	consulAddress := DefaultConsulProviderAddress
	if cfg.Consul.Address != "" {
		consulAddress = cfg.Consul.Address
	}
	consulScheme := "http"
	if cfg.Consul.SSL {
		consulScheme = "https"
	}

	return map[string]interface{}{
		"version":   cfg.Terraform.ConsulPluginVersionConstraint,
		"address":   consulAddress,
		"scheme":    consulScheme,
		"ca_file":   cfg.Consul.CA,
		"cert_file": cfg.Consul.Cert,
		"key_file":  cfg.Consul.Key,
	}
}
