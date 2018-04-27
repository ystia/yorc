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
	"github.com/ystia/yorc/tosca"
	"strconv"
	"strings"
	"sync"
	"time"
)

var defaultMonManager *monitoringMgr

// Start allows to instantiate a default Monitoring Manager and to start polling Consul agent checks
func Start(cc *api.Client) error {
	defaultMonManager = &monitoringMgr{
		cc:                    cc,
		checksNb:              0,
		checksPollingDuration: 5 * time.Second,
		chEndPolling:          make(chan struct{}),
	}

	checks, err := defaultMonManager.listCheckReports(nil)
	if err != nil {
		return errors.Wrapf(err, "Unable to start monitoring")
	}
	// Start checks polling only if any health check is registered
	defaultMonManager.checksNb = len(checks)
	if defaultMonManager.checksNb > 0 {
		defaultMonManager.startCheckReportsPolling(nil)
	}
	return nil
}

// Stop allows to stop polling Consul agent checks
func Stop() {
	if defaultMonManager != nil && defaultMonManager.checksNb > 0 {
		close(defaultMonManager.chEndPolling)
	}
}

type monitoringMgr struct {
	cc                    *api.Client
	checksNbLock          sync.RWMutex
	checksNb              int
	checksPollingDuration time.Duration
	chEndPolling          chan struct{}
}

// ExecDelegateFunc represents an alias to the exec delegate function in the aim of being decorated for monitoring
type ExecDelegateFunc func(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error

// MonitoredExecDelegate decorates an execDelegate function with the monitoring handling
func MonitoredExecDelegate(f ExecDelegateFunc) ExecDelegateFunc {
	return func(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
		err := f(ctx, cfg, taskID, deploymentID, nodeName, delegateOperation)
		if err == nil {
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

			if err := handleMonitoring(ctx, cc, taskID, deploymentID, nodeName, delegateOperation); err != nil {
				events.WithContextOptionalFields(ctx).
					NewLogEntry(events.WARN, deploymentID).Registerf("[Warning] Monitoring hasn't be handled due to error:%+v", err)
			}
		}
		return err
	}
}

func handleMonitoring(ctx context.Context, cc *api.Client, taskID, deploymentID, nodeName, delegateOperation string) error {
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

			id := defaultMonManager.buildCheckID(deploymentID, nodeName, instance)
			if err := defaultMonManager.addHealthCheck(ctx, id, ipAddress, 22, monitoringInterval); err != nil {
				return err
			}
		}
		return nil
	case "uninstall":
		for _, instance := range instances {
			id := defaultMonManager.buildCheckID(deploymentID, nodeName, instance)
			if err := defaultMonManager.removeHealthCheck(id); err != nil {
				return err
			}
		}
	default:
		return errors.Errorf("Operation %q not supported", delegateOperation)
	}
	return nil
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

// addHealthCheck allows to register a TCP consul health check
func (mgr *monitoringMgr) addHealthCheck(ctx context.Context, id, ipAddress string, port, interval int) error {
	log.Debugf("Adding health check with id:%q, iPAddress:%q, port:%d, interval:%d", id, ipAddress, port, interval)
	tcpAddr := fmt.Sprintf("%s:%d", ipAddress, port)
	check := &api.AgentCheckRegistration{
		ID:   id,
		Name: id,
		AgentServiceCheck: api.AgentServiceCheck{
			Interval: strconv.Itoa(interval) + "s",
			TCP:      tcpAddr,
			Status:   "passing",
		},
	}

	if err := mgr.cc.Agent().CheckRegister(check); err != nil {
		return errors.Wrapf(err, "Failed to add health check with id:%q, ipAddress:%q, port:%d, interval:%d", id, ipAddress, port, interval)
	}

	mgr.checksNbLock.Lock()
	defer mgr.checksNbLock.Unlock()

	// Start checks polling if no initial checks
	if mgr.checksNb == 0 {
		mgr.chEndPolling = make(chan struct{})
		mgr.startCheckReportsPolling(ctx)
	}
	mgr.checksNb++
	return nil
}

// removeHealthCheck allows to unregister a TCP consul health check
func (mgr *monitoringMgr) removeHealthCheck(id string) error {
	log.Debugf("Removing health check with id:%q", id)

	if err := mgr.cc.Agent().CheckDeregister(id); err != nil {
		return errors.Wrapf(err, "Failed to remove health check with id:%q", id)
	}

	mgr.checksNbLock.Lock()
	defer mgr.checksNbLock.Unlock()
	mgr.checksNb--
	// Stop checks polling if no more checks
	if mgr.checksNb == 0 {
		close(mgr.chEndPolling)
	}
	return nil
}

// CheckReportFilterFunc defines a filter function for CheckReport
type CheckReportFilterFunc func(CheckReport) bool

// listCheckReports can return a filtered health checks list if defined filter function. Otherwise, it returns the full checks.
func (mgr monitoringMgr) listCheckReports(f CheckReportFilterFunc) ([]CheckReport, error) {
	checks, err := mgr.cc.Agent().Checks()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list health checks")
	}
	return filter(mgr.toCheckReport(checks), f), nil
}

func (mgr *monitoringMgr) buildCheckID(deploymentID, nodeName, instance string) string {
	return fmt.Sprintf("%s:%s:%s", deploymentID, nodeName, instance)
}

func (mgr *monitoringMgr) parseCheckID(checkID string) (*CheckReport, error) {
	tab := strings.Split(checkID, ":")
	if len(tab) != 3 {
		return nil, errors.Errorf("Failed to parse checkID:%q", checkID)
	}
	return &CheckReport{DeploymentID: tab[0], NodeName: tab[1], Instance: tab[2]}, nil
}

// startCheckReportsPolling polls Consul agent checks
func (mgr *monitoringMgr) startCheckReportsPolling(ctx context.Context) {
	ticker := time.NewTicker(mgr.checksPollingDuration)
	go func() {
		for {
			select {
			case <-mgr.chEndPolling:
				log.Debug("Ending polling has been requested: stop it now.")
				ticker.Stop()
				return
			case <-ticker.C:
				log.Debugf("Polling checks is running")
				if err := mgr.updateNodesState(ctx); err != nil {
					log.Printf("[ERROR] An error occurred during polling checks:%+v", err)
				}
			}
		}
	}()
}

func (mgr monitoringMgr) toCheckReport(agentChecks map[string]*api.AgentCheck) []CheckReport {
	res := make([]CheckReport, 0)

	for _, v := range agentChecks {
		cr, err := mgr.parseCheckID(v.CheckID)
		if err != nil {
			log.Printf("[WARNING] Failed to parse check report from checkID:%q. This check is ignored.", v.CheckID)
			continue
		}
		st, err := ParseCheckStatus(v.Status)
		if err != nil {
			log.Printf("[WARNING] Failed to parse check report status from status:%q. This check is ignored.", v.Status)
			continue
		}
		cr.Status = st
		res = append(res, *cr)
	}
	return res
}

func (mgr *monitoringMgr) updateNodesState(ctx context.Context) error {
	checkReports, err := mgr.listCheckReports(nil)
	if err != nil {
		return errors.Wrap(err, "Failed to update node states in function of monitoring health checks")
	}
	for _, cr := range checkReports {
		nodeStateBefore, err := deployments.GetInstanceState(mgr.cc.KV(), cr.DeploymentID, cr.NodeName, cr.Instance)
		if err != nil {
			return err
		}

		var nodeStateAfter tosca.NodeState
		if cr.Status != CheckStatusPASSING {
			nodeStateAfter = tosca.NodeStateError
		} else if cr.Status == CheckStatusPASSING {
			nodeStateAfter = tosca.NodeStateStarted
		}
		if nodeStateBefore != nodeStateAfter {
			if ctx != nil {
				logOptFields, _ := events.FromContext(ctx)
				logOptFields[events.InstanceID] = cr.Instance
				ctx = events.NewContext(ctx, logOptFields)
			} else {
				lof := events.LogOptionalFields{
					events.InstanceID: cr.NodeName,
					events.NodeID:     cr.Instance,
				}
				ctx = events.NewContext(context.Background(), lof)
			}

			// Log change on node state
			if nodeStateAfter == tosca.NodeStateError {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, cr.DeploymentID).Registerf("Health check monitoring returned a connection failure for node (%s-%s)", cr.NodeName, cr.Instance)
			} else if nodeStateAfter == tosca.NodeStateStarted {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.INFO, cr.DeploymentID).Registerf("Health check monitoring is back to normal for node (%s-%s)", cr.NodeName, cr.Instance)
			}
			// Update the node state
			if err := deployments.SetInstanceState(mgr.cc.KV(), cr.DeploymentID, cr.NodeName, cr.Instance, nodeStateAfter); err != nil {
				return err
			}
		}
	}
	return nil
}

func filter(tab []CheckReport, f CheckReportFilterFunc) []CheckReport {
	if f == nil {
		return tab
	}
	res := make([]CheckReport, 0)
	for _, v := range tab {
		if f(v) {
			res = append(res, v)
		}
	}
	return res
}
