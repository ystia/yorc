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

package hostspool

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/dustin/go-humanize"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/labelsutil"
	"github.com/ystia/yorc/v4/tasks"
	"github.com/ystia/yorc/v4/tosca"
)

type defaultExecutor struct {
}

type operationParameters struct {
	taskID            string
	deploymentID      string
	nodeName          string
	delegateOperation string
	hpManager         Manager
}

// Mutex to ensure consistency of a host allocations and resources labels
var hostsPoolAllocMutex sync.Mutex

func (e *defaultExecutor) ExecDelegate(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	cc, err := cfg.GetConsulClient()
	if err != nil {
		return err
	}

	operationParams := operationParameters{
		taskID:            taskID,
		deploymentID:      deploymentID,
		nodeName:          nodeName,
		delegateOperation: delegateOperation,
		hpManager:         NewManager(cc),
	}
	return e.execDelegateHostsPool(ctx, cc, cfg, operationParams)
}

func (e *defaultExecutor) execDelegateHostsPool(
	ctx context.Context, cc *api.Client, cfg config.Configuration,
	op operationParameters) error {

	instances, err := tasks.GetInstances(cc.KV(), op.taskID, op.deploymentID, op.nodeName)
	if err != nil {
		return err
	}
	allocatedResources, err := e.getAllocatedResourcesFromHostCapabilities(cc.KV(),
		op.deploymentID, op.nodeName)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve allocated resources from host capabilities for node %q and deploymentID %q",
			op.nodeName, op.deploymentID)
	}

	switch strings.ToLower(op.delegateOperation) {
	case "install":
		setInstancesStateWithContextualLogs(ctx, cc, op, instances, tosca.NodeStateCreating)
		err = e.hostsPoolCreate(ctx, cc, cfg, op, allocatedResources)
		if err != nil {
			return err
		}
		setInstancesStateWithContextualLogs(ctx, cc, op, instances, tosca.NodeStateStarted)
	case "uninstall":
		setInstancesStateWithContextualLogs(ctx, cc, op, instances, tosca.NodeStateDeleting)
		err = e.hostsPoolDelete(ctx, cc, cfg, op, allocatedResources)
		if err != nil {
			return err
		}
		setInstancesStateWithContextualLogs(ctx, cc, op, instances, tosca.NodeStateDeleted)
	default:
		return errors.Errorf("operation %q not supported", op.delegateOperation)
	}
	return nil
}

func setInstancesStateWithContextualLogs(
	ctx context.Context,
	cc *api.Client,
	op operationParameters,
	instances []string,
	state tosca.NodeState) {

	for _, instance := range instances {
		deployments.SetInstanceStateWithContextualLogs(
			events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.InstanceID: instance}),
			cc.KV(), op.deploymentID, op.nodeName, instance, state)
	}
}

func (e *defaultExecutor) hostsPoolCreate(ctx context.Context,
	cc *api.Client, cfg config.Configuration,
	op operationParameters, allocatedResources map[string]string) error {

	jsonProp, err := deployments.GetNodePropertyValue(cc.KV(), op.deploymentID, op.nodeName, "filters")
	if err != nil {
		return err
	}
	var filtersString []string
	if jsonProp != nil && jsonProp.RawString() != "" {
		err = json.Unmarshal([]byte(jsonProp.RawString()), &filtersString)
		if err != nil {
			return errors.Wrapf(err, `failed to parse property "filter" for node %q as json %q`, op.nodeName, jsonProp.String())
		}
	}
	filters, err := createFiltersFromComputeCapabilities(cc.KV(), op.deploymentID, op.nodeName)
	if err != nil {
		return err
	}
	for i := range filtersString {
		f, err := labelsutil.CreateFilter(filtersString[i])
		if err != nil {
			return err
		}
		filters = append(filters, f)
	}

	shareable := false
	if s, err := deployments.GetNodePropertyValue(cc.KV(), op.deploymentID, op.nodeName, "shareable"); err != nil {
		return err
	} else if s != nil && s.RawString() != "" {
		shareable, err = strconv.ParseBool(s.RawString())
		if err != nil {
			return err
		}
	}

	instances, err := tasks.GetInstances(cc.KV(), op.taskID, op.deploymentID, op.nodeName)
	if err != nil {
		return err
	}

	return e.allocateHostsToInstances(ctx, instances, shareable, filters, op, allocatedResources)
}

func (e *defaultExecutor) allocateHostsToInstances(
	originalCtx context.Context,
	instances []string,
	shareable bool,
	filters []labelsutil.Filter,
	op operationParameters,
	allocatedResources map[string]string) error {

	for _, instance := range instances {
		ctx := events.AddLogOptionalFields(originalCtx, events.LogOptionalFields{events.InstanceID: instance})

		allocation := &Allocation{
			NodeName:     op.nodeName,
			Instance:     instance,
			DeploymentID: op.deploymentID,
			Shareable:    shareable,
			Resources:    allocatedResources}

		// Protecting the allocation and update of resources labels by a mutex, to
		// ensure no other worker will attempt to over-allocate resources of a
		// host if another worker has allocated but not yet updated resources labels
		hostsPoolAllocMutex.Lock()
		hostname, warnings, err := op.hpManager.Allocate(allocation, filters...)
		if err == nil {
			err = op.hpManager.UpdateResourcesLabels(hostname, allocatedResources, subtract, updateResourcesLabels)
		}
		hostsPoolAllocMutex.Unlock()

		for _, warn := range warnings {
			events.WithContextOptionalFields(ctx).
				NewLogEntry(events.LogLevelWARN, op.deploymentID).Registerf(`%v`, warn)
		}
		if err != nil {
			return err
		}
		err = deployments.SetInstanceAttribute(op.deploymentID, op.nodeName, instance, "hostname", hostname)
		if err != nil {
			return err
		}
		host, err := op.hpManager.GetHost(hostname)
		if err != nil {
			return err
		}

		err = e.updateConnectionSettings(ctx, op, host, hostname, instance)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *defaultExecutor) updateConnectionSettings(
	ctx context.Context, op operationParameters, host Host, hostname, instance string) error {

	err := deployments.SetInstanceCapabilityAttribute(op.deploymentID, op.nodeName,
		instance, tosca.ComputeNodeEndpointCapabilityName, tosca.EndpointCapabilityIPAddressAttribute,
		host.Connection.Host)
	if err != nil {
		return err
	}
	credentials := map[string]interface{}{"user": host.Connection.User}
	if host.Connection.Password != "" {
		credentials["token"] = host.Connection.Password
	}
	if host.Connection.PrivateKey != "" {
		credentials["keys"] = []string{host.Connection.PrivateKey}
	}
	err = deployments.SetInstanceCapabilityAttributeComplex(op.deploymentID,
		op.nodeName, instance, tosca.ComputeNodeEndpointCapabilityName, "credentials", credentials)
	if err != nil {
		return err
	}

	privateAddress, ok := host.Labels[tosca.ComputeNodePrivateAddressAttributeName]
	if !ok {
		privateAddress = host.Connection.Host
		events.WithContextOptionalFields(ctx).
			NewLogEntry(events.LogLevelWARN, op.deploymentID).Registerf(
			`no "%q label for host %q, will use the address from the connection section`,
			tosca.ComputeNodePrivateAddressAttributeName, hostname)
	}
	err = setInstanceAttributesValue(op, instance, privateAddress,
		[]string{tosca.EndpointCapabilityIPAddressAttribute, tosca.ComputeNodePrivateAddressAttributeName})

	if publicAddress, ok := host.Labels[tosca.ComputeNodePublicAddressAttributeName]; ok {
		// For compatibility with components referencing a host public_ip_address,
		// defining an attribute public_ip_address as well
		err = setInstanceAttributesValue(op, instance, privateAddress,
			[]string{tosca.ComputeNodePublicAddressAttributeName, "public_ip_address"})

		err = deployments.SetInstanceAttribute(op.deploymentID, op.nodeName,
			instance, tosca.ComputeNodePublicAddressAttributeName, publicAddress)
		if err != nil {
			return err
		}
	}

	if host.Connection.Port != 0 {
		err = deployments.SetInstanceCapabilityAttribute(op.deploymentID, op.nodeName,
			instance, tosca.ComputeNodeEndpointCapabilityName, tosca.EndpointCapabilityPortProperty,
			strconv.FormatUint(host.Connection.Port, 10))
		if err != nil {
			return err
		}
	}

	return setInstanceAttributesFromLabels(op, instance, host.Labels)
}

func setInstanceAttributesValue(op operationParameters, instance, value string, attributes []string) error {
	for _, attr := range attributes {
		err := deployments.SetInstanceAttribute(op.deploymentID, op.nodeName, instance,
			attr, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func setInstanceAttributesFromLabels(op operationParameters, instance string, labels map[string]string) error {
	for label, value := range labels {
		err := setAttributeFromLabel(op.deploymentID, op.nodeName, instance,
			label, value, tosca.ComputeNodeNetworksAttributeName, tosca.NetworkNameProperty)
		if err != nil {
			return err
		}
		err = setAttributeFromLabel(op.deploymentID, op.nodeName, instance,
			label, value, tosca.ComputeNodeNetworksAttributeName, tosca.NetworkIDProperty)
		if err != nil {
			return err
		}
		// This is bad as we split value even if we are not sure that it matches
		err = setAttributeFromLabel(op.deploymentID, op.nodeName, instance,
			label, strings.Split(value, ","), tosca.ComputeNodeNetworksAttributeName,
			tosca.NetworkAddressesProperty)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *defaultExecutor) getAllocatedResourcesFromHostCapabilities(kv *api.KV, deploymentID, nodeName string) (map[string]string, error) {
	res := make(map[string]string, 0)
	p, err := deployments.GetCapabilityPropertyValue(kv, deploymentID, nodeName, "host", "num_cpus")
	if err != nil {
		return nil, err
	}
	if p != nil && p.RawString() != "" {
		res["host.num_cpus"] = p.RawString()
	}

	p, err = deployments.GetCapabilityPropertyValue(kv, deploymentID, nodeName, "host", "mem_size")
	if err != nil {
		return nil, err
	}
	if p != nil && p.RawString() != "" {
		res["host.mem_size"] = p.RawString()
	}

	p, err = deployments.GetCapabilityPropertyValue(kv, deploymentID, nodeName, "host", "disk_size")
	if err != nil {
		return nil, err
	}
	if p != nil && p.RawString() != "" {
		res["host.disk_size"] = p.RawString()
	}
	return res, nil
}

func appendCapabilityFilter(kv *api.KV, deploymentID, nodeName, capName, propName, op string, filters []labelsutil.Filter) ([]labelsutil.Filter, error) {
	p, err := deployments.GetCapabilityPropertyValue(kv, deploymentID, nodeName, capName, propName)
	if err != nil {
		return filters, err
	}

	hasProp, propDataType, err := deployments.GetCapabilityPropertyType(kv, deploymentID,
		nodeName, capName, propName)
	if err != nil {
		return filters, err
	}

	if p != nil && p.RawString() != "" {
		var sb strings.Builder
		sb.WriteString(capName)
		sb.WriteString(".")
		sb.WriteString(propName)
		sb.WriteString(" ")
		sb.WriteString(op)
		sb.WriteString(" ")
		if hasProp && propDataType == "string" {
			// Strings need to be quoted in filters
			sb.WriteString("'")
			sb.WriteString(p.RawString())
			sb.WriteString("'")

		} else {
			sb.WriteString(p.RawString())
		}

		f, err := labelsutil.CreateFilter(sb.String())
		if err != nil {
			return filters, err
		}
		return append(filters, f), nil
	}
	return filters, nil
}

func createFiltersFromComputeCapabilities(kv *api.KV, deploymentID, nodeName string) ([]labelsutil.Filter, error) {
	var err error
	filters := make([]labelsutil.Filter, 0)
	filters, err = appendCapabilityFilter(kv, deploymentID, nodeName, "host", "num_cpus", ">=", filters)
	if err != nil {
		return nil, err
	}
	filters, err = appendCapabilityFilter(kv, deploymentID, nodeName, "host", "cpu_frequency", ">=", filters)
	if err != nil {
		return nil, err
	}
	filters, err = appendCapabilityFilter(kv, deploymentID, nodeName, "host", "disk_size", ">=", filters)
	if err != nil {
		return nil, err
	}
	filters, err = appendCapabilityFilter(kv, deploymentID, nodeName, "host", "mem_size", ">=", filters)
	if err != nil {
		return nil, err
	}
	filters, err = appendCapabilityFilter(kv, deploymentID, nodeName, "os", "architecture", "=", filters)
	if err != nil {
		return nil, err
	}
	filters, err = appendCapabilityFilter(kv, deploymentID, nodeName, "os", "type", "=", filters)
	if err != nil {
		return nil, err
	}
	filters, err = appendCapabilityFilter(kv, deploymentID, nodeName, "os", "distribution", "=", filters)
	if err != nil {
		return nil, err
	}
	filters, err = appendCapabilityFilter(kv, deploymentID, nodeName, "os", "version", "=", filters)
	if err != nil {
		return nil, err
	}
	return filters, nil
}

func (e *defaultExecutor) hostsPoolDelete(originalCtx context.Context, cc *api.Client,
	cfg config.Configuration, op operationParameters, allocatedResources map[string]string) error {
	instances, err := tasks.GetInstances(cc.KV(), op.taskID, op.deploymentID, op.nodeName)
	if err != nil {
		return err
	}
	var errs error
	for _, instance := range instances {
		ctx := events.AddLogOptionalFields(originalCtx, events.LogOptionalFields{events.InstanceID: instance})
		hostname, err := deployments.GetInstanceAttributeValue(
			cc.KV(), op.deploymentID, op.nodeName, instance, "hostname")
		if err != nil {
			errs = multierror.Append(errs, err)
		}
		if hostname == nil || hostname.RawString() == "" {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, op.deploymentID).Registerf(
				"instance %q of node %q has no registered hostname. This may be due to an error at creation time.",
				instance, op.nodeName)
			continue
		}
		allocation := &Allocation{NodeName: op.nodeName, Instance: instance, DeploymentID: op.deploymentID}
		err = op.hpManager.Release(hostname.RawString(), allocation)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
		return op.hpManager.UpdateResourcesLabels(hostname.RawString(), allocatedResources, add, updateResourcesLabels)

	}
	return errors.Wrap(errs, "errors encountered during hosts pool node release. Some hosts maybe not properly released.")
}

func setAttributeFromLabel(deploymentID, nodeName, instance, label string, value interface{}, prefix, suffix string) error {
	if strings.HasPrefix(label, prefix+".") && strings.HasSuffix(label, "."+suffix) {
		attrName := strings.Replace(strings.Replace(label, prefix+".", prefix+"/", -1), "."+suffix, "/"+suffix, -1)
		err := deployments.SetInstanceAttributeComplex(deploymentID, nodeName, instance, attrName, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateResourcesLabels(origin map[string]string, diff map[string]string, operation func(a int64, b int64) int64) (map[string]string, error) {
	labels := make(map[string]string)

	// Host Resources Labels can only be updated when deployment resources requirement is described
	if cpusDiffStr, ok := diff["host.num_cpus"]; ok {
		if cpusOriginStr, ok := origin["host.num_cpus"]; ok {
			cpusOrigin, err := strconv.Atoi(cpusOriginStr)
			if err != nil {
				return nil, err
			}
			cpusDiff, err := strconv.Atoi(cpusDiffStr)
			if err != nil {
				return nil, err
			}

			res := operation(int64(cpusOrigin), int64(cpusDiff))
			labels["host.num_cpus"] = strconv.Itoa(int(res))
		}
	}

	if memDiffStr, ok := diff["host.mem_size"]; ok {
		if memOriginStr, ok := origin["host.mem_size"]; ok {
			memOrigin, err := humanize.ParseBytes(memOriginStr)
			if err != nil {
				return nil, err
			}
			memDiff, err := humanize.ParseBytes(memDiffStr)
			if err != nil {
				return nil, err
			}

			res := operation(int64(memOrigin), int64(memDiff))
			labels["host.mem_size"] = formatBytes(res, isIECformat(memOriginStr))
		}
	}

	if diskDiffStr, ok := diff["host.disk_size"]; ok {
		if diskOriginStr, ok := origin["host.disk_size"]; ok {
			diskOrigin, err := humanize.ParseBytes(diskOriginStr)
			if err != nil {
				return nil, err
			}
			diskDiff, err := humanize.ParseBytes(diskDiffStr)
			if err != nil {
				return nil, err
			}

			res := operation(int64(diskOrigin), int64(diskDiff))
			labels["host.disk_size"] = formatBytes(res, isIECformat(diskOriginStr))
		}
	}

	return labels, nil
}

func add(valA int64, valB int64) int64 {
	return valA + valB
}

func subtract(valA int64, valB int64) int64 {
	return valA - valB
}

func formatBytes(value int64, isIEC bool) string {
	if isIEC {
		return humanize.IBytes(uint64(value))
	}
	return humanize.Bytes(uint64(value))
}

func isIECformat(value string) bool {
	if value != "" && strings.HasSuffix(value, "iB") {
		return true
	}
	return false
}
