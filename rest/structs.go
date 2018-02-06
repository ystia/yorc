package rest

import "novaforge.bull.com/starlings-janus/janus/events"
import "novaforge.bull.com/starlings-janus/janus/tosca"
import (
	"encoding/json"

	"novaforge.bull.com/starlings-janus/janus/registry"
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
	ID        string            `json:"id"`
	TargetID  string            `json:"target_id"`
	Type      string            `json:"type"`
	Status    string            `json:"status"`
	ResultSet map[string]string `json:"result_set,omitempty"`
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

// RegistryInfraUsageCollectorsCollection is the collection of infrastructure usage collectors registered in the Janus registry
type RegistryInfraUsageCollectorsCollection struct {
	InfrastructureUsageCollectors []registry.InfrastructureUsageCollector `json:"infrastructure_usage_collectors"`
}
