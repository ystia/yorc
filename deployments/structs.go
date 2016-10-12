package deployments

//go:generate stringer -type=DeploymentStatus structs.go

type DeploymentStatus int

const (
	startOfDepStatusConst DeploymentStatus = iota // Do not remove this line and define new const after it. It is used to get const value from string
	INITIAL
	DEPLOYMENT_IN_PROGRESS
	DEPLOYED
	UNDEPLOYMENT_IN_PROGRESS
	UNDEPLOYED
	DEPLOYMENT_FAILED
	UNDEPLOYMENT_FAILED

	endOfDepStatusConst // Do not remove this line and define new const before it. It is used to get const value from string
)

const DeploymentKVPrefix string = "_janus/deployments"

type Event struct {
	Timestamp string `json:"timestamp"`
	Node      string `json:"node"`
	Status    string `json:"status"`
}

type Status struct {
	Status string `json:"status"`
}

type Logs struct {
	Timestamp string `json:"timestamp"`
	Logs      string `json:"logs"`
}
