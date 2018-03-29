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
	"strconv"
	"strings"

	"github.com/pkg/errors"
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

// An Host holds information on an Host as it is known by the hostspool
type Host struct {
	Name          string            `json:"name,omitempty"`
	Connection    Connection        `json:"connection"`
	Status        HostStatus        `json:"status"`
	Shareable     bool              `json:"shareable"`
	Message       string            `json:"reason,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	AllocationsNb int               `json:"allocations_nb,omitempty"`
}

// hostResources represents host main resources (cpus, disk, ram)
type hostResources struct {
	cpus     int64
	memSize  int64
	diskSize int64
}
