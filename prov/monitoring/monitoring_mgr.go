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
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/tasks/workflow"
	"github.com/ystia/yorc/tasks/workflow/builder"
	"github.com/ystia/yorc/tosca"
)

var defaultMonManager *monitoringMgr

func init() {
	workflow.RegisterPreActivityHook(removeMonitoringHook)
	workflow.RegisterPostActivityHook(addMonitoringHook)
}

type monitoringMgr struct {
	cc               *api.Client
	chStopMonitoring chan struct{}
	chShutdown       chan struct{}
	isMonitoring     bool
	isMonitoringLock sync.Mutex
	checks           map[string]*Check
	serviceKey       string
	cfg              config.Configuration
}

// Start allows to instantiate a default Monitoring Manager and to start monitoring checks
func Start(cfg config.Configuration, cc *api.Client) {
	defaultMonManager = &monitoringMgr{
		cc:           cc,
		chShutdown:   make(chan struct{}),
		isMonitoring: false,
		serviceKey:   path.Join(consulutil.YorcServicePrefix, "/monitoring/leader"),
		cfg:          cfg,
	}

	// Watch leader election for monitoring service
	go consulutil.WatchLeaderElection(defaultMonManager.cc, defaultMonManager.serviceKey, defaultMonManager.chShutdown, defaultMonManager.startMonitoring, defaultMonManager.stopMonitoring)
}

// Stop allows to stop managing monitoring checks
func Stop() {
	// Stop Monitoring checks
	defaultMonManager.stopMonitoring()

	// Stop watch leader election
	close(defaultMonManager.chShutdown)
}

func handleError(err error) {
	err = errors.Wrap(err, "[WARN] Error during polling monitoring checks")
	log.Print(err)
	log.Debugf("%+v", err)
}

func (mgr *monitoringMgr) stopMonitoring() {
	if defaultMonManager.isMonitoring {
		log.Debugf("Monitoring service is about to be stopped")
		close(defaultMonManager.chStopMonitoring)
		defaultMonManager.isMonitoringLock.Lock()
		defaultMonManager.isMonitoring = false
		defaultMonManager.isMonitoringLock.Unlock()

		// Stop all running checks
		for _, check := range defaultMonManager.checks {
			check.Stop()
		}
	}
}

func (mgr *monitoringMgr) startMonitoring() {
	if mgr.isMonitoring {
		log.Println("Monitoring is already running.")
		return
	}
	log.Debugf("Monitoring service is now running.")

	mgr.isMonitoringLock.Lock()
	mgr.isMonitoring = true
	mgr.isMonitoringLock.Unlock()
	mgr.chStopMonitoring = make(chan struct{})
	mgr.checks = make(map[string]*Check)
	var waitIndex uint64
	go func() {
		for {
			select {
			case <-mgr.chStopMonitoring:
				log.Debugf("Ending monitoring has been requested: stop it now.")
				return
			case <-mgr.chShutdown:
				log.Debugf("Shutdown has been sent: stop monitoring checks now.")
				return
			default:
			}

			q := &api.QueryOptions{WaitIndex: waitIndex}
			checks, rMeta, err := mgr.cc.KV().Keys(path.Join(consulutil.MonitoringKVPrefix, "checks")+"/", "/", q)
			log.Debugf("Found %d checks", len(checks))
			if err != nil {
				handleError(err)
				continue
			}
			if waitIndex == rMeta.LastIndex {
				// long pool ended due to a timeout
				// there is no new items go back to the pooling
				continue
			}
			waitIndex = rMeta.LastIndex
			log.Debugf("Monitoring Wait index: %d", waitIndex)
			for _, key := range checks {
				id := path.Base(key)
				check, err := NewCheckFromID(id)
				if err != nil {
					handleError(err)
					continue
				}

				// Handle check unregistration
				kvp, _, err := mgr.cc.KV().Get(path.Join(key, ".unregisterFlag"), nil)
				if err != nil {
					handleError(err)
					continue
				}
				if kvp != nil && len(kvp.Value) > 0 && strings.ToLower(string(kvp.Value)) == "true" {
					log.Debugf("Check with id:%q has been requested to be stopped and unregister", id)
					checkToStop, is := mgr.checks[id]
					if is {
						// Stop the check execution
						checkToStop.Stop()
						// Remove it from the manager checks
						delete(mgr.checks, id)
						// Unregister it definitively
						mgr.unregisterCheck(id)
					}
					continue
				}

				kvp, _, err = mgr.cc.KV().Get(path.Join(key, "interval"), nil)
				if err != nil {
					handleError(err)
					continue
				}
				if kvp != nil && len(kvp.Value) > 0 {
					d, err := time.ParseDuration(string(kvp.Value))
					if err != nil {
						handleError(err)
						continue
					}
					check.TimeInterval = d
				}
				kvp, _, err = mgr.cc.KV().Get(path.Join(key, "address"), nil)
				if err != nil {
					handleError(err)
					continue
				}
				if kvp != nil && len(kvp.Value) > 0 {
					check.TCPAddress = string(kvp.Value)
				}

				reportPath := path.Join(consulutil.MonitoringKVPrefix, "reports", id)
				kvp, _, err = mgr.cc.KV().Get(path.Join(reportPath, "status"), nil)
				if err != nil {
					handleError(err)
					continue
				}
				if kvp != nil && len(kvp.Value) > 0 {
					check.Report.Status, err = ParseCheckStatus(string(kvp.Value))
					if err != nil {
						handleError(err)
						continue
					}
				}

				// Store the check if not already present and start it
				_, is := mgr.checks[id]
				if !is {
					mgr.checks[check.ID] = check
					check.Start()
				}
			}
		}
	}()
}

func addMonitoringHook(ctx context.Context, cfg config.Configuration, taskID, deploymentID, target string, activity builder.Activity) {
	// Monitoring check are added after (post-hook):
	// - Delegate activity and install operation
	// - SetState activity and node state "Started"

	switch {
	case activity.Type() == builder.ActivityTypeDelegate && strings.ToLower(activity.Value()) == "install",
		activity.Type() == builder.ActivityTypeSetState && activity.Value() == tosca.NodeStateStarted.String():

		// Check if monitoring is required
		isMonitorReq, monitoringInterval, err := defaultMonManager.isMonitoringRequired(deploymentID, target)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).
				Registerf("Failed to check if monitoring is required for node name:%q due to: %v", target, err)
			return
		}
		if !isMonitorReq {
			return
		}

		instances, err := tasks.GetInstances(defaultMonManager.cc.KV(), taskID, deploymentID, target)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).
				Registerf("Failed to retrieve instances for node name:%q due to: %v", target, err)
			return
		}

		for _, instance := range instances {
			ipAddress, err := deployments.GetInstanceAttributeValue(defaultMonManager.cc.KV(), deploymentID, target, instance, "ip_address")
			if err != nil {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).
					Registerf("Failed to retrieve ip_address for node name:%q due to: %v", target, err)
				return
			}
			if ipAddress == nil || ipAddress.RawString() == "" {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).
					Registerf("No attribute ip_address has been found for nodeName:%q, instance:%q with deploymentID:%q", target, instance, deploymentID)
				return
			}

			if err := defaultMonManager.registerCheck(deploymentID, target, instance, ipAddress.RawString(), 22, monitoringInterval); err != nil {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).
					Registerf("Failed to register check for node name:%q due to: %v", target, err)
				return
			}
		}
	}
}

func removeMonitoringHook(ctx context.Context, cfg config.Configuration, taskID, deploymentID, target string, activity builder.Activity) {
	// Monitoring check are removed before (pre-hook):
	// - Delegate activity and uninstall operation
	// - SetState activity and node state "Deleted"
	switch {
	case activity.Type() == builder.ActivityTypeDelegate && strings.ToLower(activity.Value()) == "uninstall",
		activity.Type() == builder.ActivityTypeSetState && activity.Value() == tosca.NodeStateDeleted.String():

		// Check if monitoring has been required
		isMonitorReq, _, err := defaultMonManager.isMonitoringRequired(deploymentID, target)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).
				Registerf("Failed to check if monitoring is required for node name:%q due to: %v", target, err)
			return
		}
		if !isMonitorReq {
			return
		}

		instances, err := tasks.GetInstances(defaultMonManager.cc.KV(), taskID, deploymentID, target)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).
				Registerf("Failed to retrieve instances for node name:%q due to: %v", target, err)
			return
		}

		for _, instance := range instances {
			if err := defaultMonManager.flagCheckForRemoval(deploymentID, target, instance); err != nil {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).
					Registerf("Failed to unregister check for node name:%q due to: %v", target, err)
				return
			}
		}
	}
}

func (mgr *monitoringMgr) isMonitoringRequired(deploymentID, nodeName string) (bool, time.Duration, error) {
	// Check if the node is a compute
	var isCompute bool
	isCompute, err := deployments.IsNodeDerivedFrom(mgr.cc.KV(), deploymentID, nodeName, "tosca.nodes.Compute")
	if err != nil {
		return false, 0, err
	}
	if !isCompute {
		return false, 0, nil
	}

	// monitoring_time_interval must be set to positive value
	found, val, err := deployments.GetNodeMetadata(mgr.cc.KV(), deploymentID, nodeName, "monitoring_time_interval")
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

// registerCheck allows to register a check
func (mgr *monitoringMgr) registerCheck(deploymentID, nodeName, instance, ipAddress string, port int, interval time.Duration) error {
	id := buildID(deploymentID, nodeName, instance)
	log.Debugf("Register check with id:%q, iPAddress:%q, port:%d, interval:%d", id, ipAddress, port, interval)
	tcpAddr := fmt.Sprintf("%s:%d", ipAddress, port)

	// Check is registered in a transaction to ensure to be read in its wholeness
	checkPath := path.Join(consulutil.MonitoringKVPrefix, "checks", id)
	checkReportPath := path.Join(consulutil.MonitoringKVPrefix, "reports", id)

	checkOps := api.KVTxnOps{
		&api.KVTxnOp{
			Verb: api.KVCheckNotExists,
			Key:  path.Join(checkPath, "address"),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(checkPath, "address"),
			Value: []byte(tcpAddr),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(checkPath, "interval"),
			Value: []byte(interval.String()),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(checkReportPath, "status"),
			Value: []byte(CheckStatusPASSING.String()),
		},
	}

	ok, response, _, err := mgr.cc.KV().Txn(checkOps, nil)
	if err != nil {
		return errors.Wrapf(err, "Failed to add check with id:%q", id)
	}
	if !ok {
		// Check the response
		errs := make([]string, 0)
		for _, e := range response.Errors {
			if e.OpIndex == 0 {
				return errors.Wrapf(err, "Check with id:%q and TCP address:%q already exists", id, tcpAddr)
			}
			errs = append(errs, e.What)
		}
		return errors.Errorf("Failed to add check with id:%q due to:%s", id, strings.Join(errs, ", "))
	}
	return nil
}

// flagCheckForRemoval allows to remove a check report and flag a check in order to remove it
func (mgr *monitoringMgr) flagCheckForRemoval(deploymentID, nodeName, instance string) error {
	id := buildID(deploymentID, nodeName, instance)
	log.Debugf("PreUnregisterCheck check with id:%q", id)
	checkPath := path.Join(consulutil.MonitoringKVPrefix, "checks", id)
	return consulutil.StoreConsulKeyAsString(path.Join(checkPath, ".unregisterFlag"), "true")
}

// unregisterCheck allows to unregister a check and its related report
func (mgr *monitoringMgr) unregisterCheck(id string) error {
	log.Debugf("Removing check with id:%q", id)
	checkPath := path.Join(consulutil.MonitoringKVPrefix, "checks", id)
	checkReportPath := path.Join(consulutil.MonitoringKVPrefix, "reports", id)
	rmOps := api.KVTxnOps{
		&api.KVTxnOp{
			Verb: api.KVDeleteTree,
			Key:  checkPath,
		},
		&api.KVTxnOp{
			Verb: api.KVDeleteTree,
			Key:  checkReportPath,
		},
	}

	ok, response, _, err := mgr.cc.KV().Txn(rmOps, nil)
	if err != nil {
		return errors.Wrapf(err, "Failed to remove check and report for id:%q", id)
	}
	if !ok {
		// Check the response
		errs := make([]string, 0)
		for _, e := range response.Errors {
			errs = append(errs, e.What)
		}
		return errors.Errorf("Failed to remove check and report for id:%q due to:%s", id, strings.Join(errs, ", "))
	}
	return nil
}

// CheckFilterFunc defines a filter function for CheckReport
type CheckFilterFunc func(CheckReport) bool

// listCheckReports can return a filtered checks reports list if defined filter function. Otherwise, it returns the full check reports.
func (mgr *monitoringMgr) listCheckReports(f CheckFilterFunc) ([]CheckReport, error) {
	log.Debugf("List check reports")
	keys, _, err := mgr.cc.KV().Keys(path.Join(consulutil.MonitoringKVPrefix, "reports")+"/", "/", nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	checkReports := make([]CheckReport, 0)
	for _, key := range keys {
		id := path.Base(key)
		check, err := NewCheckFromID(id)
		if err != nil {
			return nil, err
		}

		kvp, _, err := mgr.cc.KV().Get(path.Join(key, "status"), nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp != nil && len(kvp.Value) > 0 {
			check.Report.Status, err = ParseCheckStatus(string(kvp.Value))
			if err != nil {
				return nil, err
			}
		}
		checkReports = append(checkReports, check.Report)
	}
	return filter(checkReports, f), nil
}

func filter(tab []CheckReport, f CheckFilterFunc) []CheckReport {
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
