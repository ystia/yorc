package deployments

//go:generate stringer -type=DeploymentStatus structs.go

type DeploymentStatus int

const (
	INITIAL DeploymentStatus = iota
	DEPLOYMENT_IN_PROGRESS
	DEPLOYED
	UNDEPLOYMENT_IN_PROGRESS
	UNDEPLOYED
	DEPLOYMENT_FAILED
	UNDEPLOYMENT_FAILED
)

const DeploymentKVPrefix string = "_janus/deployments"

type Event struct {
	Timestamp string `json:"timestamp"`
	Node      string `json:"node"`
	Status    string `json:"status"`
}
