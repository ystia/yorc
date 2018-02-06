package rest

//go:generate go-enum -f=structs.go --lower

import (
	"bytes"
	"encoding/json"

	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/prov/hostspool"
	"novaforge.bull.com/starlings-janus/janus/registry"
	"novaforge.bull.com/starlings-janus/janus/tosca"
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
	// JanusIndexHeader is the name of the HTTP header containing the last index for long polling endpoints
	JanusIndexHeader string = "X-Janus-Index"
)

const (
	// JanusDeploymentIDPattern is the allowed pattern for Janus deployments IDs
	JanusDeploymentIDPattern string = "^[-_0-9a-zA-Z]+$"

	// Disable this for now as it doesn't have a concrete impact for now
	// JanusDeploymentIDMaxLength is the maximum allowed length for Janus deployments IDs
	//JanusDeploymentIDMaxLength int = 36
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

// Deployment is the representation of a Janus deployment
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

// DeploymentsCollection is a collection of deployments links
//
// Links are all of type LinkRelDeployment.
type DeploymentsCollection struct {
	Deployments []AtomLink `json:"deployments"`
}

// EventsCollection is a collection of instances status change events
type EventsCollection struct {
	Events    []events.StatusUpdate `json:"events"`
	LastIndex uint64                `json:"last_index"`
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

// Task is the representation of a Janus' task
type Task struct {
	ID       string `json:"id"`
	TargetID string `json:"target_id"`
	Type     string `json:"type"`
	Status   string `json:"status"`
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
	NodeName          string            `json:"node"`
	CustomCommandName string            `json:"name"`
	Inputs            map[string]string `json:"inputs"`
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

// HostRequest represents a request for creating or updating a host in the hosts pool
type HostRequest struct {
	Connection *hostspool.Connection `json:"connection,omitempty"`
	Tags       []MapEntry            `json:"tags,omitempty"`
}

// HostsCollection is a collection of hosts registered in the host pool links
//
// Links are all of type LinkRelHost.
type HostsCollection struct {
	Hosts []AtomLink `json:"hosts"`
}

// Host is a host in the host pool representation
//
// Links are all of type LinkRelSelf.
type Host struct {
	hostspool.Host
	Links []AtomLink `json:"links"`
}

// RegistryDelegatesCollection is the collection of Delegates executors registered in the Janus registry
type RegistryDelegatesCollection struct {
	Delegates []registry.DelegateMatch `json:"delegates"`
}

// RegistryImplementationsCollection is the collection of Operation executors registered in the Janus registry
type RegistryImplementationsCollection struct {
	Implementations []registry.OperationExecMatch `json:"implementations"`
}

// RegistryDefinitionsCollection is the collection of TOSCA Definitions registered in the Janus registry
type RegistryDefinitionsCollection struct {
	Definitions []registry.Definition `json:"definitions"`
}

// RegistryVaultsCollection is the collection of Vaults Clients Builders registered in the Janus registry
type RegistryVaultsCollection struct {
	VaultClientBuilders []registry.VaultClientBuilder `json:"vaults"`
}
