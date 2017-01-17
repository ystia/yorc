package deployments

//go:generate stringer -type=DeploymentStatus structs.go

// DeploymentStatus represent the current status of a deployment
type DeploymentStatus int

const (
	startOfDepStatusConst DeploymentStatus = iota // Do not remove this line and define new const after it. It is used to get const value from string
	// INITIAL is the initial status of a deployment when it has not yet started
	INITIAL
	// DEPLOYMENT_IN_PROGRESS deployment is in progress
	DEPLOYMENT_IN_PROGRESS
	// DEPLOYED deployment is deployed without error
	DEPLOYED
	// UNDEPLOYMENT_IN_PROGRESS undeployment is in progress
	UNDEPLOYMENT_IN_PROGRESS
	// UNDEPLOYED deployment is no longer deployed
	UNDEPLOYED
	// DEPLOYMENT_FAILED deployment encountered an error
	DEPLOYMENT_FAILED
	// UNDEPLOYMENT_FAILED undeployment encountered an error
	UNDEPLOYMENT_FAILED
	// SCALING_IN_PROGRESS instances are currently added or removed to the deployment
	SCALING_IN_PROGRESS

	endOfDepStatusConst // Do not remove this line and define new const before it. It is used to get const value from string
)

// InfraLogPrefix Consul KV prefix for infrastructure logs
const InfraLogPrefix = "infrastructure"

// SoftwareLogPrefix Consul KV prefix for software provisioning logs
const SoftwareLogPrefix = "software"

// EngineLogPrefix Consul KV prefix for janus engine logs
const EngineLogPrefix = "engine"

// Event represents node instance status change event
type Event struct {
	Timestamp string `json:"timestamp"`
	Node      string `json:"node"`
	Instance  string `json:"instance"`
	Status    string `json:"status"`
}

// Logs represents a log message entry
type Logs struct {
	Timestamp string `json:"timestamp"`
	Logs      string `json:"logs"`
}
