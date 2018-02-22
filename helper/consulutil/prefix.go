package consulutil

const yorcPrefix string = "_yorc"

// DeploymentKVPrefix is the prefix in Consul KV store for deployments
const DeploymentKVPrefix string = yorcPrefix + "/deployments"

// TasksPrefix is the prefix in Consul KV store for tasks
const TasksPrefix = yorcPrefix + "/tasks"

// TasksLocksPrefix is the prefix in Consul KV store for tasks locks
const TasksLocksPrefix = yorcPrefix + "/tasks-locks"

// WorkflowsPrefix is the prefix in Consul KV store for workflows runtime data
const WorkflowsPrefix = yorcPrefix + "/workflows"

// EventsPrefix is the prefix in Consul KV store for events concerning all the deployments
const EventsPrefix = yorcPrefix + "/events"

// LogsPrefix is the prefix on KV store for logs concerning all the deployments
const LogsPrefix = yorcPrefix + "/logs"

// HostsPoolPrefix is the prefix on KV store for the hosts pool service
const HostsPoolPrefix = yorcPrefix + "/hosts_pool"
