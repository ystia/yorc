package rest

import "novaforge.bull.com/starlings-janus/janus/deployments"

type Deployment struct {
	Id     string `json:"id"`
	Status string `json:"status"`
}

type Output struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

const (
	LINK_REL_DEPLOYMENT string = "deployment"
)

const (
	LINK_REL_OUTPUT string = "output"
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

type OutputsCollection struct {
	Outputs []AtomLink `json:"outputs,omitempty"`
}
