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
	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/deployments"
	"github.com/ystia/yorc/v3/events"
	"github.com/ystia/yorc/v3/tasks"
	"github.com/ystia/yorc/v3/tasks/workflow"
	"github.com/ystia/yorc/v3/tasks/workflow/builder"
	"github.com/ystia/yorc/v3/tosca"
	"strings"
)

func init() {
	workflow.RegisterPreActivityHook(removeMonitoringHook)
	workflow.RegisterPostActivityHook(addMonitoringHook)
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

			if err := defaultMonManager.registerTCPCheck(deploymentID, target, instance, ipAddress.RawString(), 22, monitoringInterval); err != nil {
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
