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

package rest

//go:generate go-enum -f=structs.go --lower

import (
	"bytes"
	"encoding/json"

	"github.com/ystia/yorc/prov/hostspool"
	"github.com/ystia/yorc/registry"
	"github.com/ystia/yorc/tosca"
)

const (
	// LinkRelSelf defines the AtomLink Rel attribute for relationships of the "self"
	LinkRelSelf string = "self"
	// LinkRelDeployment defines the AtomLink Rel attribute for relationships of the "deployment"
	LinkRelDeployment string = "deployment"
	// LinkRelNode defines the AtomLink Rel attribute for relationships of the "node"
	LinkRelNode string = "node"
	// LinkRelInstance defines the AtomLink Rel attribute for relationships of the "instance"
	LinkRelInstance string = "instance"
	// LinkRelOutput defines the AtomLink Rel attribute for relationships of the "output"
	LinkRelOutput string = "output"
	// LinkRelTask defines the AtomLink Rel attribute for relationships of the "task"
	LinkRelTask string = "task"
	// LinkRelAttribute defines the AtomLink Rel attribute for relationships of the "attribute"
	LinkRelAttribute string = "attribute"
	// LinkRelWorkflow defines the AtomLink Rel attribute for relationships of the "attribute"
	LinkRelWorkflow string = "workflow"
	// LinkRelHost defines the AtomLink Rel attribute for relationships of the "host" (for hostspool)
	LinkRelHost string = "host"
)

const (
	// YorcIndexHeader is the name of the HTTP header containing the last index for long polling endpoints
	YorcIndexHeader string = "X-Yorc-Index"
)

const (
	// YorcDeploymentIDPattern is the allowed pattern for Yorc deployments IDs
	YorcDeploymentIDPattern string = "^[-_0-9a-zA-Z]+$"

	// Disable this for now as it doesn't have a concrete impact for now
	// YorcDeploymentIDMaxLength is the maximum allowed length for Yorc deployments IDs
	//YorcDeploymentIDMaxLength int = 36
)

// An AtomLink is defined in the Atom specification (https://tools.ietf.org/html/rfc4287#section-4.2.7) it allows to reference REST endpoints
// in the HATEOAS model
type AtomLink struct {
	Rel      string `json:"rel"`
	Href     string `json:"href"`
	LinkType string `json:"type"`
}

func newAtomLink(rel, href string) AtomLink {
	return AtomLink{Rel: rel, Href: href, LinkType: "application/json"}
}

// Health of a Yorc instance
type Health struct {
	Value string `json:"value"`
}

// Deployment is the representation of a Yorc deployment
//
// Deployment's links may be of type LinkRelSelf, LinkRelNode, LinkRelTask, LinkRelOutput.
type Deployment struct {
	ID     string     `json:"id"`
	Status string     `json:"status"`
	Links  []AtomLink `json:"links"`
}

// Output is the representation of a deployment output
type Output struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// DeploymentsCollection is a collection of Deployment
//
// Links are all of type LinkRelDeployment.
type DeploymentsCollection struct {
	Deployments []Deployment `json:"deployments"`
}

// EventsCollection is a collection of instances status change events
type EventsCollection struct {
	Events    []json.RawMessage `json:"events"`
	LastIndex uint64            `json:"last_index"`
}

// LogsCollection is a collection of logs events
type LogsCollection struct {
	Logs      []json.RawMessage `json:"logs"`
	LastIndex uint64            `json:"last_index"`
}

// Node is the representation of a TOSCA node
//
// Node's links are of type LinkRelSelf, LinkRelDeployment and LinkRelInstance.
type Node struct {
	Name  string     `json:"name"`
	Links []AtomLink `json:"links"`
}

// NodeInstance is the representation of a TOSCA node
//
// Node's links are of type LinkRelSelf, LinkRelDeployment, LinkRelNode and LinkRelAttribute.
type NodeInstance struct {
	ID     string     `json:"id"`
	Status string     `json:"status"`
	Links  []AtomLink `json:"links"`
}

// OutputsCollection is a collection of deployment's outputs links
type OutputsCollection struct {
	Outputs []AtomLink `json:"outputs,omitempty"`
}

// Task is the representation of a Yorc' task
type Task struct {
	ID        string          `json:"id"`
	TargetID  string          `json:"target_id"`
	Type      string          `json:"type"`
	Status    string          `json:"status"`
	ResultSet json.RawMessage `json:"result_set,omitempty"`
}

// TasksCollection is the collection of task's links
type TasksCollection struct {
	Tasks []AtomLink `json:"tasks,omitempty"`
}

// TaskRequest is the representation of a request to process a new task
type TaskRequest struct {
	Type string `json:"type"`
}

// AttributesCollection is a collection of node instance's attributes links
type AttributesCollection struct {
	Attributes []AtomLink `json:"attributes,omitempty"`
}

// Attribute is the representation of an TOSCA node instance attribute
type Attribute struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// CustomCommandRequest is the representation of a request to process a Custom Command
type CustomCommandRequest struct {
	NodeName          string                            `json:"node"`
	CustomCommandName string                            `json:"name"`
	InterfaceName     string                            `json:"interface,omitempty"`
	Inputs            map[string]*tosca.ValueAssignment `json:"inputs"`
}

// WorkflowsCollection is a collection of workflows links
//
// Links are all of type LinkRelWorkflow.
type WorkflowsCollection struct {
	Workflows []AtomLink `json:"workflows"`
}

// Workflow is a workflow representation.
type Workflow struct {
	Name string `json:"name"`
	tosca.Workflow
}

// MapEntryOperation is an enumeration of valid values for a MapEntry.Op field
// ENUM(
// Add,
// Remove
// )
type MapEntryOperation int

// MarshalJSON is used to represent this enumeration as a string instead of an int
func (o MapEntryOperation) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(o.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON is used to read this enumeration from a string
func (o *MapEntryOperation) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	*o, err = ParseMapEntryOperation(s)
	return err
}

// A MapEntry allows to manipulate a Map collection by performing operations (Op) on entries.
// It is inspired (with many simplifications) by the JSON Patch RFC https://tools.ietf.org/html/rfc6902
type MapEntry struct {
	// Op is the operation for this entry. The default if omitted is "add". An "add" operation acts like a remplace if the entry already exists
	Op MapEntryOperation `json:"op,omitempty"`
	// Name is the map entry key name
	Name string `json:"name"`
	// Value is the value of the map entry. Optional for a "remove" operation
	Value string `json:"value,omitempty"`
}

// HostConfig represents the configuration of a host in the Hosts Pool
type HostConfig struct {
	Name       string
	Connection hostspool.Connection `json:"connection,omitempty" yaml:"connection,omitempty"`
	Labels     map[string]string    `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// HostsPoolRequest represents a request for applying a Hosts Pool configuration
type HostsPoolRequest struct {
	Hosts []HostConfig `json:"hosts"`
}

// HostRequest represents a request for creating or updating a host in the hosts pool
type HostRequest struct {
	Connection *hostspool.Connection `json:"connection,omitempty"`
	Labels     []MapEntry            `json:"labels,omitempty"`
}

// HostsCollection is a collection of hosts registered in the host pool links
//
// Links are all of type LinkRelHost.
type HostsCollection struct {
	Checkpoint uint64     `json:"checkpoint,omitempty"`
	Hosts      []AtomLink `json:"hosts"`
	Warnings   []string   `json:"warnings,omitempty"`
}

// Host is a host in the host pool representation
//
// Links are all of type LinkRelSelf.
type Host struct {
	hostspool.Host
	Links []AtomLink `json:"links"`
}

// RegistryDelegatesCollection is the collection of Delegates executors registered in the Yorc registry
type RegistryDelegatesCollection struct {
	Delegates []registry.DelegateMatch `json:"delegates"`
}

// RegistryImplementationsCollection is the collection of Operation executors registered in the Yorc registry
type RegistryImplementationsCollection struct {
	Implementations []registry.OperationExecMatch `json:"implementations"`
}

// RegistryDefinitionsCollection is the collection of TOSCA Definitions registered in the Yorc registry
type RegistryDefinitionsCollection struct {
	Definitions []registry.Definition `json:"definitions"`
}

// RegistryVaultsCollection is the collection of Vaults Clients Builders registered in the Yorc registry
type RegistryVaultsCollection struct {
	VaultClientBuilders []registry.VaultClientBuilder `json:"vaults"`
}

// RegistryInfraUsageCollectorsCollection is the collection of infrastructure usage collectors registered in the Yorc registry
type RegistryInfraUsageCollectorsCollection struct {
	InfraUsageCollectors []registry.InfraUsageCollector `json:"infrastructure_usage_collectors"`
}
