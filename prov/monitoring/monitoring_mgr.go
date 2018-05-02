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
	"github.com/ystia/yorc/tasks/workflow"
	"github.com/ystia/yorc/tosca"
	"strconv"
	"strings"
	"sync"
	"time"
)

var defaultMonManager *monitoringMgr

func init() {
	workflow.RegisterPostActivityHook(computeMonitoringHook)
}

// Start allows to instantiate a default Monitoring Manager and to start polling Consul agent checks
func Start(cc *api.Client, cfg config.Configuration) error {
	defaultMonManager = &monitoringMgr{
		cc:                    cc,
		checksNb:              0,
		checksPollingDuration: cfg.Consul.HealthCheckPollingInterval,
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

func computeMonitoringHook(ctx context.Context, cfg config.Configuration, taskID, deploymentID, target string, activity workflow.Activity) {
	if activity.Type() != workflow.ActivityTypeDelegate && activity.Type() != workflow.ActivityTypeSetState {
		return
	}
	if activity.Type() == workflow.ActivityTypeSetState && (activity.Value() != tosca.NodeStateStarted.String() && activity.Value() != tosca.NodeStateDeleted.String()) {
		return
	}
	// Get Consul Client
	cc, err := cfg.GetConsulClient()
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).
			Registerf("Failed to retrieve consul client when handling compute monitoring for node name:%q due to: %v", target, err)
		return
	}

	// Check if monitoring is required
	isMonitorReq, monitoringInterval, err := isMonitoringRequired(cc, deploymentID, target)
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).
			Registerf("Failed to check if monitoring is required for node name:%q due to: %v", target, err)
		return
	}
	if !isMonitorReq {
		return
	}

	if err := handleMonitoring(ctx, cc, taskID, deploymentID, target, activity, monitoringInterval); err != nil {
		events.WithContextOptionalFields(ctx).
			NewLogEntry(events.WARN, deploymentID).Registerf("Health check monitoring hasn't be handled correctly due to:%+v", err)
	}
}

func handleMonitoring(ctx context.Context, cc *api.Client, taskID, deploymentID, nodeName string, activity workflow.Activity, monitoringInterval time.Duration) error {
	log.Debugf("Handle monitoring for deploymentID:%q, nodeName:%q", deploymentID, nodeName)
	instances, err := tasks.GetInstances(cc.KV(), taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}

	// Monitoring compute is run on:
	// - Delegate activity (checks are registered during install and unregistered during uninstall operation)
	// - SetState activity (checks are registered on node state "Started" and unregistered on node state "Deleted")
	switch {
	case activity.Type() == workflow.ActivityTypeDelegate && strings.ToLower(activity.Value()) == "install",
		activity.Type() == workflow.ActivityTypeSetState && activity.Value() == tosca.NodeStateStarted.String():

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
	case activity.Type() == workflow.ActivityTypeDelegate && strings.ToLower(activity.Value()) == "uninstall",
		activity.Type() == workflow.ActivityTypeSetState && activity.Value() == tosca.NodeStateDeleted.String():

		for _, instance := range instances {
			id := defaultMonManager.buildCheckID(deploymentID, nodeName, instance)
			if err := defaultMonManager.removeHealthCheck(id); err != nil {
				return err
			}
		}
	}

	return nil
}

func isMonitoringRequired(cc *api.Client, deploymentID, nodeName string) (bool, time.Duration, error) {
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
		duration, err := time.ParseDuration(val + "s")
		if err != nil {
			return false, 0, err
		}
		return true, duration, nil
	}
	return false, 0, nil
}

// addHealthCheck allows to register a TCP consul health check
func (mgr *monitoringMgr) addHealthCheck(ctx context.Context, id, ipAddress string, port, interval time.Duration) error {
	log.Debugf("Adding health check with id:%q, iPAddress:%q, port:%d, interval:%d", id, ipAddress, port, interval)
	tcpAddr := fmt.Sprintf("%s:%d", ipAddress, port)
	check := &api.AgentCheckRegistration{
		ID:   id,
		Name: id,
		AgentServiceCheck: api.AgentServiceCheck{
			Interval: interval.String(),
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
				logOptFields, ok := events.FromContext(ctx)
				if ok {
					logOptFields[events.InstanceID] = cr.Instance
					ctx = events.NewContext(ctx, logOptFields)
				}
			} else {
				lof := events.LogOptionalFields{
					events.InstanceID: cr.Instance,
					events.NodeID:     cr.NodeName,
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
