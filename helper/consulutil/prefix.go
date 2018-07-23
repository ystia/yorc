// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package consulutil

const yorcPrefix string = "_yorc"

// DeploymentKVPrefix is the prefix in Consul KV store for deployments
const DeploymentKVPrefix string = yorcPrefix + "/deployments"

// TasksPrefix is the prefix in Consul KV store for tasks
const TasksPrefix = yorcPrefix + "/tasks"

// ExecutionsTaskPrefix is the prefix in Consul KV store for execution task
const ExecutionsTaskPrefix = yorcPrefix + "/executions"

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

// YorcManagementPrefix is the prefix on KV store for the orchestrator
// own management
const YorcManagementPrefix = yorcPrefix + "/.yorc_management"

// MonitoringKVPrefix is the prefix in Consul KV store for monitoring
const MonitoringKVPrefix string = yorcPrefix + "/monitoring"

// YorcServicePrefix is the prefix used for storing services keys
const YorcServicePrefix = yorcPrefix + "/service"
