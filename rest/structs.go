package rest

import "novaforge.bull.com/starlings-janus/janus/deployments"

type Deployment struct {
	Id     string     `json:"id"`
	Status string     `json:"status"`
	Links  []AtomLink `json:"links"`
}

type Output struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

const (
	LINK_REL_SELF       string = "self"
	LINK_REL_DEPLOYMENT string = "deployment"
	LINK_REL_NODE       string = "node"
	LINK_REL_INSTANCE   string = "instance"
	LINK_REL_OUTPUT     string = "output"
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
	Name   string     `json:"name"`
	Status string     `json:"status"`
	Links  []AtomLink `json:"links"`
}

type NodeInstance struct {
	Id     string     `json:"id"`
	Status string     `json:"status"`
	Links  []AtomLink `json:"links"`
}

type OutputsCollection struct {
	Outputs []AtomLink `json:"outputs,omitempty"`
}
