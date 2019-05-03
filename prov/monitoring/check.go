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
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/deployments"
	"github.com/ystia/yorc/v3/events"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/tosca"
)

// NewCheck allows to instantiate a Check
func NewCheck(deploymentID, nodeName, instance string) *Check {
	return &Check{ID: buildID(deploymentID, nodeName, instance), Report: CheckReport{DeploymentID: deploymentID, NodeName: nodeName, Instance: instance}}
}

// NewCheckFromID allows to instantiate a new Check from an pre-existing ID
func NewCheckFromID(checkID string) (*Check, error) {
	tab := strings.Split(checkID, ":")
	if len(tab) != 3 {
		return nil, errors.Errorf("Malformed check ID :%q", checkID)
	}
	return &Check{ID: checkID, Report: CheckReport{DeploymentID: tab[0], NodeName: tab[1], Instance: tab[2]}}, nil
}

// Start allows to start running a TCP check
func (c *Check) Start() {
	c.stopLock.Lock()
	defer c.stopLock.Unlock()

	// Instantiate ctx for check
	lof := events.LogOptionalFields{
		events.InstanceID: c.Report.Instance,
		events.NodeID:     c.Report.NodeName,
	}
	c.ctx = events.NewContext(context.Background(), lof)

	// timeout is defined arbitrary as half interval to avoid overlap
	c.timeout = c.TimeInterval / 2
	// instantiate channel to close the internal routine
	c.chStop = make(chan struct{})
	c.stop = false

	// check if initially the node can be monitored according to its node state
	if c.isNodeStateOKForMonitoring() {
		c.enable()
	}

	go func() {
		// time interval for polling node state needs to be inferior than time interval for checks to detect when checks must be disabled
		ticker := time.NewTicker(c.TimeInterval / 2)
		for {
			select {
			case <-c.chStop:
				ticker.Stop()
				log.Debugf("Stop monitoring check with id:%s", c.ID)
				c.disable()
				return
			case <-ticker.C:
				if !c.isNodeStateOKForMonitoring() {
					// Disable check
					c.disable()
					continue
				}
				// Enable check
				c.enable()
			}
		}
	}()
}

func (c *Check) isNodeStateOKForMonitoring() bool {
	instanceState, err := deployments.GetInstanceState(defaultMonManager.cc.KV(), c.Report.DeploymentID, c.Report.NodeName, c.Report.Instance)
	if err != nil || instanceState != tosca.NodeStateStarted && instanceState != tosca.NodeStateError {
		return false
	}
	return true
}

// Stop allows to stop a TCP check
func (c *Check) Stop() {
	c.stopLock.Lock()
	defer c.stopLock.Unlock()

	if !c.stop {
		c.stop = true
		close(c.chStop)
	}
}

func (c *Check) disable() {
	if c.enabled {
		c.enabled = false
		close(c.chDisable)
	}
}

func (c *Check) enable() {
	if !c.enabled {
		// instantiate channel to close the check ticker
		c.chDisable = make(chan struct{})
		c.enabled = true
		go c.run()
	}
}

func (c *Check) run() {
	log.Debugf("Running check:%+v", c)
	ticker := time.NewTicker(c.TimeInterval)
	for {
		select {
		case <-c.chDisable:
			log.Debugf("Disable check with id:%s", c.ID)
			ticker.Stop()
			return
		case <-ticker.C:
			status, mess := c.execution.execute(c.timeout)
			c.updateStatus(status, mess)
		}
	}
}

func (c *Check) exist() bool {
	checkPath := path.Join(consulutil.MonitoringKVPrefix, "reports", c.ID, "status")
	KVPair, _, err := defaultMonManager.cc.KV().Get(checkPath, nil)
	if err != nil {
		log.Println("[WARN] Failed to get check due to error:%+v", err)
		return false
	}
	if KVPair == nil || len(KVPair.Value) == 0 {
		return false
	}
	return true
}

func (c *Check) updateStatus(status CheckStatus, message string) {
	if c.Report.Status != status {
		// Be sure check isn't currently being removed before check has been stopped
		if !c.exist() {
			return
		}

		log.Debugf("Update check status from %q to %q", c.Report.Status.String(), status.String())
		err := consulutil.StoreConsulKeyAsString(path.Join(consulutil.MonitoringKVPrefix, "reports", c.ID, "status"), status.String())
		if err != nil {
			log.Printf("[WARN] TCP check updating status failed for check ID:%q due to error:%+v", c.ID, err)
		}
		c.Report.Status = status
		c.notify(message)
	}
}

func (c *Check) notify(additionalMessage string) {
	var nodeState tosca.NodeState
	var eventLevel events.LogLevel
	var statusChangeMess string

	switch c.Report.Status {
	case CheckStatusPASSING:
		// Back to normal
		nodeState = tosca.NodeStateStarted
		eventLevel = events.LogLevelINFO
		statusChangeMess = fmt.Sprintf("Monitoring Check is back to normal for node (%s-%s)", c.Report.NodeName, c.Report.Instance)
	case CheckStatusCRITICAL:
		// Node in ERROR
		nodeState = tosca.NodeStateError
		eventLevel = events.LogLevelERROR
		statusChangeMess = fmt.Sprintf("Monitoring Check returned a failure for node (%s-%s)", c.Report.NodeName, c.Report.Instance)
	case CheckStatusWARNING:
		// TODO maybe a warning state should be more appropriate
		nodeState = tosca.NodeStateError
		eventLevel = events.LogLevelWARN
		statusChangeMess = fmt.Sprintf("Monitoring Check returned a warning for node (%s-%s)", c.Report.NodeName, c.Report.Instance)
	}

	if additionalMessage != "" {
		events.WithContextOptionalFields(c.ctx).NewLogEntry(eventLevel, c.Report.DeploymentID).Registerf(additionalMessage)
	}
	events.WithContextOptionalFields(c.ctx).NewLogEntry(eventLevel, c.Report.DeploymentID).Registerf(statusChangeMess)

	// Update the node state
	if err := deployments.SetInstanceStateWithContextualLogs(c.ctx, defaultMonManager.cc.KV(), c.Report.DeploymentID, c.Report.NodeName, c.Report.Instance, nodeState); err != nil {
		log.Printf("[WARN] Unable to update node state due to error:%+v", err)
	}
}

func buildID(deploymentID, nodeName, instance string) string {
	return fmt.Sprintf("%s:%s:%s", deploymentID, nodeName, instance)
}
