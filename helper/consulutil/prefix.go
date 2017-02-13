package consulutil

const janusPrefix string = "_janus"

// DeploymentKVPrefix is the prefix in Consul KV store for deployments
const DeploymentKVPrefix string = janusPrefix + "/deployments"

// TasksPrefix is the prefix in Consul KV store for tasks
const TasksPrefix = janusPrefix + "/tasks"

// TasksLocksPrefix is the prefix in Consul KV store for tasks locks
const TasksLocksPrefix = janusPrefix + "/tasks-locks"

// WorkflowsPrefix is the prefix in Consul KV store for workflows runtime data
const WorkflowsPrefix = janusPrefix + "/workflows"
