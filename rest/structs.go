package rest

import "novaforge.bull.com/starlings-janus/janus/deployments"

type Deployment struct {
	ID     string     `json:"id"`
	Status string     `json:"status"`
	Links  []AtomLink `json:"links"`
}

type Output struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

const (
	LinkRelSelf       string = "self"
	LinkRelDeployment string = "deployment"
	LinkRelNode       string = "node"
	LinkRelInstance   string = "instance"
	LinkRelOutput     string = "output"
	LinkRelTask       string = "task"
	LinkRelAttribute  string = "attribute"
)

const (
	JanusIndexHeader string = "X-Janus-Index"
)

type AtomLink struct {
	Rel      string `json:"rel"`
	Href     string `json:"href"`
	LinkType string `json:"type"`
}

func newAtomLink(rel, href string) AtomLink {
	return AtomLink{Rel: rel, Href: href, LinkType: "application/json"}
}

type DeploymentsCollection struct {
	Deployments []AtomLink `json:"deployments"`
}

type EventsCollection struct {
	Events    []deployments.Event `json:"events"`
	LastIndex uint64              `json:"last_index"`
}

type LogsCollection struct {
	Logs      []deployments.Logs `json:"logs"`
	LastIndex uint64             `json:"last_index"`
}

type Node struct {
	Name  string     `json:"name"`
	Links []AtomLink `json:"links"`
}

type NodeInstance struct {
	ID     string     `json:"id"`
	Status string     `json:"status"`
	Links  []AtomLink `json:"links"`
}

type OutputsCollection struct {
	Outputs []AtomLink `json:"outputs,omitempty"`
}

type Task struct {
	ID       string `json:"id"`
	TargetID string `json:"target_id"`
	Type     string `json:"type"`
	Status   string `json:"status"`
}

type TaskRequest struct {
	Type string `json:"type"`
}

type AttributesCollection struct {
	Attributes []AtomLink `json:"attributes,omitempty"`
}

type Attribute struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}


type InputsPropertyDef struct {
	NodeName          string            `json:"node"`
	CustomCommandName string            `json:"name"`
	Inputs            map[string]string `json:"inputs"`
}
