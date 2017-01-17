package events

// InfraLogPrefix Consul KV prefix for infrastructure logs
const InfraLogPrefix = "infrastructure"

// SoftwareLogPrefix Consul KV prefix for software provisioning logs
const SoftwareLogPrefix = "software"

// EngineLogPrefix Consul KV prefix for janus engine logs
const EngineLogPrefix = "engine"

// InstanceStatus represents node instance status change event
type InstanceStatus struct {
	Timestamp string `json:"timestamp"`
	Node      string `json:"node"`
	Instance  string `json:"instance"`
	Status    string `json:"status"`
}

// LogEntry represents a log message entry
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Logs      string `json:"logs"`
}
