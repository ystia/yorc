package rest

import "novaforge.bull.com/starlings-janus/janus/events"

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
)

const (
	// JanusIndexHeader is the name of the HTTP header containing the last index for long polling endpoints
	JanusIndexHeader string = "X-Janus-Index"
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
	Events    []events.InstanceStatus `json:"events"`
	LastIndex uint64                  `json:"last_index"`
}

// LogsCollection is a collection of logs events
type LogsCollection struct {
	Logs      []events.LogEntry `json:"logs"`
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
