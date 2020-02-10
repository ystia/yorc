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

package hostspool

//go:generate go-enum -f=hostspool_structs.go --lower

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/helper/stringutil"
)

// HostStatus x ENUM(
// free,
// allocated,
// error
// )
type HostStatus int

// MarshalJSON is used to represent this enumeration as a string instead of an int
func (hs HostStatus) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(hs.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON is used to read this enumeration from a string
func (hs *HostStatus) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal HostStatus as string")
	}
	*hs, err = ParseHostStatus(strings.ToLower(s))
	return errors.Wrap(err, "failed to parse HostStatus from JSON input")
}

// TODO support winrm for windows hosts

// A Connection holds info used to connect to a host using SSH
type Connection struct {
	// The User that we should use for the connection. Defaults to root.
	User string `json:"user,omitempty" yaml:"user,omitempty"`
	// The Password that we should use for the connection. One of Password or PrivateKey is required. PrivateKey takes the precedence.
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	// The SSH Private Key that we should use for the connection. One of Password or PrivateKey is required. PrivateKey takes the precedence.
	// The mapstructure tag is needed for viper unmarshalling
	PrivateKey string `json:"private_key,omitempty"  yaml:"private_key,omitempty" mapstructure:"private_key"`
	// The address of the Host to connect to. Defaults to the hostname specified during the registration.
	Host string `json:"host,omitempty" yaml:"host,omitempty"`
	// The Port to connect to. Defaults to 22 if set to 0.
	Port uint64 `json:"port,omitempty" yaml:"port,omitempty"`
}

// String allows to stringify a connection
func (conn Connection) String() string {
	var pass, key string
	if conn.Password != "" {
		pass = "password: " + conn.Password + ", "
	}
	if conn.PrivateKey != "" {
		key = "private key: " + conn.PrivateKey + ", "
	}

	return "user: " + conn.User + ", " + pass + key + "host: " + conn.Host + ", " + "port: " + strconv.FormatUint(conn.Port, 10)
}

// A Pool holds information on a hosts pool
type Pool struct {
	Hosts []Host `json:"hosts,omitempty"`
}

// A PoolConfig holds information on hosts configurations of a pool
type PoolConfig struct {
	Hosts []HostConfig `json:"hosts,omitempty"`
}

// An Host holds information on an Host as it is known by the hostspool
type Host struct {
	Name        string            `json:"name,omitempty"`
	Connection  Connection        `json:"connection,omitempty"`
	Status      HostStatus        `json:"status,omitempty"`
	Message     string            `json:"reason,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Allocations []Allocation      `json:"allocations,omitempty"`
}

// An HostConfig holds information on an Host basic configuration
// It's a short version of Host representation
type HostConfig struct {
	Name       string            `json:"name,omitempty"`
	Connection Connection        `json:"connection,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// An Allocation describes the related allocation associated to a host pool
type Allocation struct {
	ID               string             `json:"id"`
	NodeName         string             `json:"node_name"`
	Instance         string             `json:"instance"`
	DeploymentID     string             `json:"deployment_id"`
	Shareable        bool               `json:"shareable"`
	Resources        map[string]string  `json:"resource_labels,omitempty"`
	GenericResources []*GenericResource `json:"generic_resource_labels,omitempty"`
	PlacementPolicy  string             `json:"placement_policy"`
}

func (alloc *Allocation) String() string {
	allocStr := fmt.Sprintf("deployment: %s,node-instance: %s-%s,shareable: %t, placement:%s", alloc.DeploymentID, alloc.NodeName, alloc.Instance, alloc.Shareable, stringutil.GetLastElement(alloc.PlacementPolicy, "."))
	if alloc.Resources != nil && len(alloc.Resources) > 0 {
		for k, v := range alloc.Resources {
			allocStr += "," + k + ": " + v
		}
	}

	return allocStr
}

func (alloc *Allocation) buildID() error {
	if alloc.NodeName == "" || alloc.Instance == "" || alloc.DeploymentID == "" {
		return errors.New("Node name, instance and deployment ID must be set")
	}
	if alloc.ID == "" {
		alloc.ID = buildAllocationID(alloc.DeploymentID, alloc.NodeName, alloc.Instance)
	}
	return nil
}

func buildAllocationID(deploymentID, nodeName, instance string) string {
	return url.QueryEscape(strings.Join([]string{deploymentID, nodeName, instance}, "-"))
}

// GenericResource represents a generic resource requirement
type GenericResource struct {
	// name of the generic resource
	Name string `json:"name"`
	// label used in allocations and hosts pool as host.generic_resource.<name>
	Label string `json:"label"`
	// allocation label value set once the generic resource is allocated/released
	Value string `json:"value"`
	// define if the generic resource can be only used by a single compute (consumable) or not
	NoConsumable bool `json:"no_consumable"`
	// list of ids of generic resources required for the allocation need. Either ids or nb are used
	ids []string
	// nb of generic resources required for the allocation need. either ids or nb are used
	nb int
}
