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
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/tasks/workflow"
	"github.com/ystia/yorc/tosca"
	"path"
	"strconv"
	"strings"
	"time"
)

var defaultMonManager *monitoringMgr

func init() {
	workflow.RegisterPostActivityHook(computeMonitoringHook)
}

type monitoringMgr struct {
	cc     *api.Client
	chStop chan struct{}
	stop   bool
	checks map[string]*Check
}

// Start allows to instantiate a default Monitoring Manager and to start monitoring checks
func Start(cc *api.Client) error {
	defaultMonManager = &monitoringMgr{
		cc:     cc,
		chStop: make(chan struct{}),
		checks: make(map[string]*Check),
	}
	defaultMonManager.startMonitoring()
	return nil
}

// Stop allows to stop monitoring checks
func Stop() {
	if !defaultMonManager.stop {
		close(defaultMonManager.chStop)
		defaultMonManager.stop = true
	}

	// Stop all running checks
	for _, check := range defaultMonManager.checks {
		check.Stop()
	}
}

func handleError(err error) {
	err = errors.Wrap(err, "Error during polling monitoring checks")
	log.Print(err)
	log.Debugf("%+v", err)
}

func (mgr *monitoringMgr) startMonitoring() {
	var waitIndex uint64
	go func() {
		for {
			select {
			case <-mgr.chStop:
				log.Debug("Ending monitoring has been requested: stop it now.")
				return
			default:
			}

			q := &api.QueryOptions{WaitIndex: waitIndex, WaitTime: 100 * time.Millisecond}
			checks, rMeta, err := mgr.cc.KV().Keys(path.Join(consulutil.MonitoringKVPrefix, "checks")+"/", "/", q)
			log.Debugf("%d checks has been found", len(checks))
			if err != nil {
				handleError(err)
				continue
			}
			if waitIndex == rMeta.LastIndex {
				log.Debugf("No changes")
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
					if !is {
						handleError(errors.Errorf("Unable to find check with id:%q in order to stop and remove it", id))
						continue
					}
					// Stop the check execution
					checkToStop.Stop()
					// Remove it from the manager checks
					delete(mgr.checks, id)
					// Unregister it definitively
					mgr.unregisterCheck(id)
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

func computeMonitoringHook(ctx context.Context, cfg config.Configuration, taskID, deploymentID, target string, activity workflow.Activity) {
	// Monitoring compute is run on:
	// - Delegate activity (checks are registered during install and unregistered during uninstall operation)
	// - SetState activity (checks are registered on node state "Started" and unregistered on node state "Deleted")
	if activity.Type() != workflow.ActivityTypeDelegate && activity.Type() != workflow.ActivityTypeSetState {
		return
	}
	if activity.Type() == workflow.ActivityTypeSetState && (activity.Value() != tosca.NodeStateStarted.String() && activity.Value() != tosca.NodeStateDeleted.String()) {
		return
	}

	// Check if monitoring is required
	isMonitorReq, monitoringInterval, err := defaultMonManager.isMonitoringRequired(deploymentID, target)
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).
			Registerf("Failed to check if monitoring is required for node name:%q due to: %v", target, err)
		return
	}
	if !isMonitorReq {
		return
	}

	instances, err := tasks.GetInstances(defaultMonManager.cc.KV(), taskID, deploymentID, target)
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).
			Registerf("Failed to retrieve instances for node name:%q due to: %v", target, err)
		return
	}

	switch {
	case activity.Type() == workflow.ActivityTypeDelegate && strings.ToLower(activity.Value()) == "install",
		activity.Type() == workflow.ActivityTypeSetState && activity.Value() == tosca.NodeStateStarted.String():

		for _, instance := range instances {
			found, ipAddress, err := deployments.GetInstanceAttribute(defaultMonManager.cc.KV(), deploymentID, target, instance, "ip_address")
			if err != nil {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).
					Registerf("Failed to retrieve ip_address for node name:%q due to: %v", target, err)
				return
			}
			if !found || ipAddress == "" {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).
					Registerf("No attribute ip_address has been found for nodeName:%q, instance:%q with deploymentID", target, instance, deploymentID)
				return
			}

			if err := defaultMonManager.registerCheck(deploymentID, target, instance, ipAddress, 22, monitoringInterval); err != nil {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).
					Registerf("Failed to register check for node name:%q due to: %v", target, err)
				return
			}
		}
	case activity.Type() == workflow.ActivityTypeDelegate && strings.ToLower(activity.Value()) == "uninstall",
		activity.Type() == workflow.ActivityTypeSetState && activity.Value() == tosca.NodeStateDeleted.String():

		for _, instance := range instances {
			if err := defaultMonManager.flagCheckForRemoval(deploymentID, target, instance); err != nil {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).
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
	kvp := &api.KVPair{Key: path.Join(checkPath, ".unregisterFlag"), Value: []byte("true")}
	_, err := mgr.cc.KV().Put(kvp, nil)
	return errors.Wrap(err, "Failed to flag check for unregister it")
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
func (mgr monitoringMgr) listCheckReports(f CheckFilterFunc) ([]CheckReport, error) {
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
