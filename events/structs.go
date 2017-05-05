package events

import "fmt"

// InfraLogPrefix Consul KV prefix for infrastructure logs
const InfraLogPrefix = "infrastructure"

// SoftwareLogPrefix Consul KV prefix for software provisioning logs
const SoftwareLogPrefix = "software"

// EngineLogPrefix Consul KV prefix for janus engine logs
const EngineLogPrefix = "engine"

// StatusUpdateType is the status update type
type StatusUpdateType uint64

const (
	// InstanceStatusChangeType is the StatusUpdate type for an instance state change event
	InstanceStatusChangeType StatusUpdateType = iota
	// DeploymentStatusChangeType is the StatusUpdate type for an deployment status change event
	DeploymentStatusChangeType
	// CustomCommandStatusChangeType is the StatusUpdate type for an custom command status change event
	CustomCommandStatusChangeType
	// ScalingStatusChangeType is the StatusUpdate type for an scaling status change event
	ScalingStatusChangeType
	// WorkflowStatusChangeType is the StatusUpdate type for an workflow status change event
	WorkflowStatusChangeType
)

// StatusUpdate represents status change event
type StatusUpdate struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Node      string `json:"node,omitempty"`
	Instance  string `json:"instance,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
	Status    string `json:"status"`
}

// LogEntry represents a log message entry
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Logs      string `json:"logs"`
}

const _StatusUpdateType_name = "instancedeploymentcustom-commandscalingworkflow"

var _StatusUpdateType_index = [...]uint8{0, 8, 18, 32, 39, 47}

func (i StatusUpdateType) String() string {
	if i >= StatusUpdateType(len(_StatusUpdateType_index)-1) {
		return fmt.Sprintf("StatusUpdateType(%d)", i)
	}
	return _StatusUpdateType_name[_StatusUpdateType_index[i]:_StatusUpdateType_index[i+1]]
}

var _StatusUpdateTypeNameToValue_map = map[string]StatusUpdateType{
	_StatusUpdateType_name[0:8]:   0,
	_StatusUpdateType_name[8:18]:  1,
	_StatusUpdateType_name[18:32]: 2,
	_StatusUpdateType_name[32:39]: 3,
	_StatusUpdateType_name[39:47]: 4,
}

func StatusUpdateTypeString(s string) (StatusUpdateType, error) {
	if val, ok := _StatusUpdateTypeNameToValue_map[s]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to StatusUpdateType values", s)
}
