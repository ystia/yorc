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

package monitoring

import (
	"context"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tasks"
	"strconv"
	"strings"
)

// Manager is responsible for adding and removing monitoring health checks
type Manager interface {
	AddHealthCheck(name, ipAddress string, port int, interval int) error
	RemoveHealthCheck(name string) error
	GetHealthChecks() (map[string]*api.AgentCheck, error)
}

// ExecDelegateFunc represents an alias to the exec delegate function in the aim of being decorated for monitoring
type ExecDelegateFunc func(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error

// MonitoredExecDelegate decorates an execDelegate function with the monitoring handling
func MonitoredExecDelegate(f ExecDelegateFunc) ExecDelegateFunc {
	return func(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
		err := f(ctx, cfg, taskID, deploymentID, nodeName, delegateOperation)
		if err == nil {
			log.Debugf("Check if monitoring is required...")
			// Fill log optional fields for log registration
			logOptFields, ok := events.FromContext(ctx)
			if !ok {
				return errors.New("Missing contextual log optional fields")
			}
			logOptFields[events.NodeID] = nodeName
			logOptFields[events.ExecutionID] = taskID
			logOptFields[events.OperationName] = delegateOperation
			logOptFields[events.InterfaceName] = "delegate"
			ctx = events.NewContext(ctx, logOptFields)

			// Get Consul Client
			cc, err := cfg.GetConsulClient()
			if err != nil {
				return err
			}

			if err := handleMonitoring(cc, taskID, deploymentID, nodeName, delegateOperation); err != nil {
				events.WithContextOptionalFields(ctx).
					NewLogEntry(events.WARN, deploymentID).Registerf("[Warning] Monitoring hasn't be handled due to error:%+v", err)
			}
		}
		return err
	}
}

func handleMonitoring(cc *api.Client, taskID, deploymentID, nodeName, delegateOperation string) error {
	// Check if monitoring is required
	isMonitorReq, monitoringInterval, err := isMonitoringRequired(cc, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if !isMonitorReq {
		return nil
	}
	log.Debugf("Handle monitoring for deploymentID:%q, nodeName:%q", deploymentID, nodeName)
	instances, err := tasks.GetInstances(cc.KV(), taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}

	monitoringMgr := NewManager(cc)
	// Check operation
	switch strings.ToLower(delegateOperation) {
	case "install":
		for _, instance := range instances {
			found, ipAddress, err := deployments.GetInstanceAttribute(cc.KV(), deploymentID, nodeName, instance, "ip_address")
			if err != nil {
				return err
			}
			if !found || ipAddress == "" {
				return errors.Errorf("No attribute ip_address has been found for nodeName:%q, instance:%q with deploymentID", nodeName, instance, deploymentID)
			}

			name := buildCheckID(deploymentID, nodeName, instance)
			if err := monitoringMgr.AddHealthCheck(name, ipAddress, 22, monitoringInterval); err != nil {
				return err
			}
		}
		return nil
	case "uninstall":
		for _, instance := range instances {
			name := buildCheckID(deploymentID, nodeName, instance)
			if err := monitoringMgr.RemoveHealthCheck(name); err != nil {
				return err
			}
		}
	}
	return errors.Errorf("operation %q not supported", delegateOperation)
}

func buildCheckID(deploymentID, nodeName, instance string) string {
	return fmt.Sprintf("%s_%s-%s", deploymentID, nodeName, instance)
}

func isMonitoringRequired(cc *api.Client, deploymentID, nodeName string) (bool, int, error) {
	// Check if the node is a compute
	var isCompute bool
	isCompute, err := deployments.IsNodeDerivedFrom(cc.KV(), deploymentID, nodeName, "tosca.nodes.Compute")
	if err != nil {
		return false, 0, err
	}
	if !isCompute {
		return false, 0, nil
	}

	// monitoring_time_interval must be set to positive value
	found, val, err := deployments.GetNodeMetadata(cc.KV(), deploymentID, nodeName, "monitoring_time_interval")
	if err != nil {
		return false, 0, err
	}
	if !found {
		return false, 0, nil
	}

	var t int
	if t, err = strconv.Atoi(val); err != nil {
		return false, 0, err
	}
	if t > 0 {
		return true, t, nil
	}
	return false, 0, nil
}

// NewManager instantiates a new monitoring manager
func NewManager(cc *api.Client) Manager {
	return monitoringMgr{cc: cc}
}

type monitoringMgr struct {
	cc *api.Client
}

// AddHealthCheck allows to register a TCP consul health check
func (mgr monitoringMgr) AddHealthCheck(name, ipAddress string, port, interval int) error {
	log.Debugf("Adding health check with name:%q, iPAddress:%q, port:%d, interval:%d", name, ipAddress, port, interval)
	tcpAddr := fmt.Sprintf("%s:%d", ipAddress, port)
	check := &api.AgentCheckRegistration{
		ID:   name,
		Name: name,
		AgentServiceCheck: api.AgentServiceCheck{
			Interval: strconv.Itoa(interval) + "s",
			TCP:      tcpAddr,
		},
	}

	if err := mgr.cc.Agent().CheckRegister(check); err != nil {
		return errors.Wrapf(err, "failed to add health check with name:%q, ipAddress:%q, port:%d, interval:%d", name, ipAddress, port, interval)
	}

	return nil
}

// RemoveHealthCheck allows to unregister a TCP consul health check
func (mgr monitoringMgr) RemoveHealthCheck(name string) error {
	log.Debugf("Removing health check with name:%q", name)

	if err := mgr.cc.Agent().CheckDeregister(name); err != nil {
		return errors.Wrapf(err, "failed to remove health check with name:%q", name)
	}
	return nil
}

// GetHealthChecks returns all health checks
func (mgr monitoringMgr) GetHealthChecks() (map[string]*api.AgentCheck, error) {
	log.Debug("Getting health checks")

	checks, err := mgr.cc.Agent().Checks()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get health checks")
	}
	return checks, nil
}
