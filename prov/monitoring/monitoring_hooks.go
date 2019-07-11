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
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tasks"
	"github.com/ystia/yorc/v4/tasks/workflow"
	"github.com/ystia/yorc/v4/tasks/workflow/builder"
	"github.com/ystia/yorc/v4/tosca"
	"strconv"
	"strings"
	"time"
)

func init() {
	workflow.RegisterPreActivityHook(removeMonitoringHook)
	workflow.RegisterPostActivityHook(addMonitoringHook)
}

const (
	httpMonitoring = "yorc.policies.monitoring.HTTPMonitoring"
	tcpMonitoring  = "yorc.policies.monitoring.TCPMonitoring"
	baseMonitoring = "yorc.policies.Monitoring"
)

func addMonitoringHook(ctx context.Context, cfg config.Configuration, taskID, deploymentID, target string, activity builder.Activity) {
	// Monitoring check are added after (post-hook):
	// - Delegate activity and install operation
	// - SetState activity and node state "Started"
	cc, err := cfg.GetConsulClient()
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, deploymentID).Registerf("Failed to check if monitoring is required for node name:%q due to: %v", target, err)
		return
	}
	kv := cc.KV()
	switch {
	case activity.Type() == builder.ActivityTypeDelegate && strings.ToLower(activity.Value()) == "install",
		activity.Type() == builder.ActivityTypeSetState && activity.Value() == tosca.NodeStateStarted.String():

		// Check if monitoring is required
		isMonitorReq, policyName, err := checkExistingMonitoringPolicy(kv, deploymentID, target)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).
				Registerf("Failed to check if monitoring is required for node name:%q due to: %v", target, err)
			return
		}
		if !isMonitorReq {
			return
		}

		err = addMonitoringPolicyForTarget(kv, taskID, deploymentID, target, policyName)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).
				Registerf("Failed to add monitoring policy for node name:%q due to: %v", target, err)
		}
	}
}

func removeMonitoringHook(ctx context.Context, cfg config.Configuration, taskID, deploymentID, target string, activity builder.Activity) {
	// Monitoring check are removed before (pre-hook):
	// - Delegate activity and uninstall operation
	// - SetState activity and node state "Deleted"
	cc, err := cfg.GetConsulClient()
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, deploymentID).Registerf("Failed to check if monitoring is required for node name:%q due to: %v", target, err)
		return
	}
	kv := cc.KV()
	switch {
	case activity.Type() == builder.ActivityTypeDelegate && strings.ToLower(activity.Value()) == "uninstall",
		activity.Type() == builder.ActivityTypeSetState && activity.Value() == tosca.NodeStateDeleted.String():

		// Check if monitoring has been required
		isMonitorReq, _, err := checkExistingMonitoringPolicy(kv, deploymentID, target)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).
				Registerf("Failed to check if monitoring is required for node name:%q due to: %v", target, err)
			return
		}
		if !isMonitorReq {
			return
		}

		instances, err := tasks.GetInstances(kv, taskID, deploymentID, target)
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

func checkExistingMonitoringPolicy(kv *api.KV, deploymentID, target string) (bool, string, error) {
	policies, err := deployments.GetPoliciesForTypeAndNode(kv, deploymentID, baseMonitoring, target)
	if err != nil {
		return false, "", err
	}
	if len(policies) == 0 {
		return false, "", nil
	}
	if len(policies) > 1 {
		return false, "", errors.Errorf("Found more than one monitoring policy to apply to node name:%q. No monitoring policy will be applied", target)
	}

	return true, policies[0], nil
}

func addMonitoringPolicyForTarget(kv *api.KV, taskID, deploymentID, target, policyName string) error {
	log.Debugf("Add monitoring policy:%q for deploymentID:%q, node name:%q", policyName, deploymentID, target)
	policyType, err := deployments.GetPolicyType(kv, deploymentID, policyName)
	if err != nil {
		return err
	}

	// Retrieve time_interval and port
	tiValue, err := deployments.GetPolicyPropertyValue(kv, deploymentID, policyName, "time_interval")
	if err != nil || tiValue == nil || tiValue.RawString() == "" {
		return errors.Errorf("Failed to retrieve time_interval for monitoring policy:%q due to: %v", policyName, err)
	}

	timeInterval, err := time.ParseDuration(tiValue.RawString())
	if err != nil {
		return errors.Errorf("Failed to retrieve time_interval as correct duration for monitoring policy:%q due to: %v", policyName, err)
	}
	portValue, err := deployments.GetPolicyPropertyValue(kv, deploymentID, policyName, "port")
	if err != nil || portValue == nil || portValue.RawString() == "" {
		return errors.Errorf("Failed to retrieve port for monitoring policy:%q due to: %v", policyName, err)
	}
	port, err := strconv.Atoi(portValue.RawString())
	if err != nil {
		return errors.Errorf("Failed to retrieve port as correct integer for monitoring policy:%q due to: %v", policyName, err)
	}
	instances, err := tasks.GetInstances(kv, taskID, deploymentID, target)
	if err != nil {
		return err
	}

	switch policyType {
	case httpMonitoring:
		return applyHTTPMonitoringPolicy(kv, policyName, deploymentID, target, timeInterval, port, instances)
	case tcpMonitoring:
		return applyTCPMonitoringPolicy(kv, deploymentID, target, timeInterval, port, instances)
	default:
		return errors.Errorf("Unsupported policy type:%q for policy:%q", policyType, policyName)
	}
}

func applyTCPMonitoringPolicy(kv *api.KV, deploymentID, target string, timeInterval time.Duration, port int, instances []string) error {
	for _, instance := range instances {
		ipAddress, err := retrieveIPAddress(kv, deploymentID, target, instance)
		if err != nil {
			return err
		}
		if err := defaultMonManager.registerTCPCheck(deploymentID, target, instance, ipAddress, port, timeInterval); err != nil {
			return errors.Errorf("Failed to register TCP check for node name:%q due to: %v", target, err)
		}
	}
	return nil
}

func applyHTTPMonitoringPolicy(kv *api.KV, policyName, deploymentID, target string, timeInterval time.Duration, port int, instances []string) error {
	for _, instance := range instances {
		ipAddress, err := retrieveIPAddress(kv, deploymentID, target, instance)
		if err != nil {
			return err
		}
		schemeValue, err := deployments.GetPolicyPropertyValue(kv, deploymentID, policyName, "scheme")
		if err != nil || schemeValue == nil || schemeValue.RawString() == "" {
			return errors.Errorf("Failed to retrieve scheme for monitoring policy:%q due to: %v", policyName, err)
		}
		var urlPath string
		pathValue, err := deployments.GetPolicyPropertyValue(kv, deploymentID, policyName, "path")
		if pathValue != nil && pathValue.RawString() != "" {
			urlPath = pathValue.RawString()
		}
		var d map[string]interface{}
		var headersMap map[string]string
		headersValue, err := deployments.GetPolicyPropertyValue(kv, deploymentID, policyName, "http_headers")
		if headersValue != nil && headersValue.RawString() != "" {
			var ok bool
			d, ok = headersValue.Value.(map[string]interface{})
			if !ok {
				return errors.New("failed to retrieve HTTP headers map from Tosca Value: not expected type")
			}

			headersMap = make(map[string]string, len(d))
			for k, v := range d {
				v, ok := v.(string)
				if !ok {
					return errors.Errorf("failed to retrieve string value from headers map from Tosca Value:%q not expected type", v)
				}
				headersMap[k] = v
			}
		}

		var tlsClientConfig map[string]string
		if schemeValue.RawString() == "https" {
			tlsClientConfig, err = retrieveTLSClientConfig(kv, policyName, deploymentID)
			if err != nil {
				return err
			}
		}

		if err := defaultMonManager.registerHTTPCheck(deploymentID, target, instance, ipAddress, schemeValue.RawString(), urlPath, port, headersMap, tlsClientConfig, timeInterval); err != nil {
			return errors.Errorf("Failed to register HTTP check for node name:%q due to: %v", target, err)
		}
	}
	return nil
}

func retrieveTLSClientConfig(kv *api.KV, policyName, deploymentID string) (map[string]string, error) {
	tlsClientConfig := make(map[string]string, 0)
	props := []string{"ca_cert", "ca_path", "client_cert", "client_key", "skip_verify"}
	for _, prop := range props {
		val, err := deployments.GetPolicyPropertyValue(kv, deploymentID, policyName, "tls_client", prop)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to retrieve TLS client config properties for policy:%q", policyName)
		}
		if val != nil && val.String() != "" {
			tlsClientConfig[prop] = val.String()
		}
	}
	return tlsClientConfig, nil
}

func retrieveIPAddress(kv *api.KV, deploymentID, target, instance string) (string, error) {
	ipAddress, err := deployments.GetInstanceAttributeValue(kv, deploymentID, target, instance, "ip_address")
	if err != nil {
		return "", errors.Errorf("Failed to retrieve ip_address for node name:%q due to: %v", target, err)
	}
	if ipAddress == nil || ipAddress.RawString() == "" {
		return "", errors.Errorf("No attribute ip_address has been found for nodeName:%q, instance:%q with deploymentID:%q", target, instance, deploymentID)
	}
	return ipAddress.RawString(), nil
}
