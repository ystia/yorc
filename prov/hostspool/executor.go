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
	"strings"

	"github.com/ystia/yorc/helper/labelsutil"

	"github.com/hashicorp/go-multierror"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/dustin/go-humanize"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/tosca"
	"strconv"
)

type defaultExecutor struct {
	allocatedResources *hostResources
}

func (e *defaultExecutor) ExecDelegate(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	cc, err := cfg.GetConsulClient()
	if err != nil {
		return err
	}
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

	instances, err := tasks.GetInstances(cc.KV(), taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}
	e.allocatedResources, err = e.getAllocatedResourcesFromHostCapabilities(cc.KV(), deploymentID, nodeName)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve allocated resources from host capabilities for node %q and deploymentID %q", nodeName, deploymentID)
	}
	switch strings.ToLower(delegateOperation) {
	case "install":
		for _, instance := range instances {
			deployments.SetInstanceState(cc.KV(), deploymentID, nodeName, instance, tosca.NodeStateCreating)
		}
		err = e.hostsPoolCreate(ctx, cc, cfg, taskID, deploymentID, nodeName)
		if err != nil {
			return err
		}
		for _, instance := range instances {
			deployments.SetInstanceState(cc.KV(), deploymentID, nodeName, instance, tosca.NodeStateStarted)
		}
		return nil
	case "uninstall":
		for _, instance := range instances {
			deployments.SetInstanceState(cc.KV(), deploymentID, nodeName, instance, tosca.NodeStateDeleting)
		}
		err = e.hostsPoolDelete(ctx, cc, cfg, taskID, deploymentID, nodeName)
		if err != nil {
			return err
		}
		for _, instance := range instances {
			deployments.SetInstanceState(cc.KV(), deploymentID, nodeName, instance, tosca.NodeStateDeleted)
		}
		return nil
	}
	return errors.Errorf("operation %q not supported", delegateOperation)
}

func (e *defaultExecutor) hostsPoolCreate(originalCtx context.Context, cc *api.Client, cfg config.Configuration, taskID, deploymentID, nodeName string) error {
	hpManager := NewManager(cc)

	_, jsonProp, err := deployments.GetNodeProperty(cc.KV(), deploymentID, nodeName, "filters")
	if err != nil {
		return err
	}
	var filtersString []string
	if jsonProp != "" {
		err = json.Unmarshal([]byte(jsonProp), &filtersString)
		if err != nil {
			return errors.Wrapf(err, `failed to parse property "filter" for node %q as json %q`, nodeName, jsonProp)
		}
	}
	filters, err := createFiltersFromComputeCapabilities(cc.KV(), deploymentID, nodeName)
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

	instances, err := tasks.GetInstances(cc.KV(), taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}
	for _, instance := range instances {
		logOptFields, _ := events.FromContext(originalCtx)
		logOptFields[events.InstanceID] = instance
		ctx := events.NewContext(originalCtx, logOptFields)

		allocation, err := NewAllocation(nodeName, instance, deploymentID)
		if err != nil {
			return err
		}
		hostname, warnings, err := hpManager.Allocate(allocation, filters...)
		for _, warn := range warnings {
			events.WithContextOptionalFields(ctx).
				NewLogEntry(events.WARN, deploymentID).Registerf(`%v`, warn)
		}
		if err != nil {
			return err
		}
		err = deployments.SetInstanceAttribute(deploymentID, nodeName, instance, "hostname", hostname)
		if err != nil {
			return err
		}
		host, err := hpManager.GetHost(hostname)
		if err != nil {
			return err
		}
		err = deployments.SetInstanceCapabilityAttribute(deploymentID, nodeName, instance, "endpoint", "ip_address", host.Connection.Host)
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
		err = deployments.SetInstanceCapabilityAttributeComplex(deploymentID, nodeName, instance, "endpoint", "credentials", credentials)
		if err != nil {
			return err
		}

		privateAddress, ok := host.Labels["private_address"]
		if !ok {
			privateAddress = host.Connection.Host
			events.WithContextOptionalFields(ctx).
				NewLogEntry(events.WARN, deploymentID).Registerf(`no "private_address" label for host %q, we will use the address from the connection section`, hostname)
		}
		err = deployments.SetInstanceAttribute(deploymentID, nodeName, instance, "ip_address", privateAddress)
		if err != nil {
			return err
		}
		err = deployments.SetInstanceAttribute(deploymentID, nodeName, instance, "private_address", privateAddress)
		if err != nil {
			return err
		}

		if publicAddress, ok := host.Labels["public_address"]; ok {
			err = deployments.SetInstanceAttribute(deploymentID, nodeName, instance, "public_address", publicAddress)
			if err != nil {
				return err
			}
		}
		if host.Connection.Port != 0 {
			err = deployments.SetInstanceCapabilityAttribute(deploymentID, nodeName, instance, "endpoint", "port", strconv.FormatUint(host.Connection.Port, 10))
			if err != nil {
				return err
			}
		}

		for label, value := range host.Labels {
			err = setAttributeFromLabel(deploymentID, nodeName, instance, label, value, "networks", "network_name")
			if err != nil {
				return err
			}
			err = setAttributeFromLabel(deploymentID, nodeName, instance, label, value, "networks", "network_id")
			if err != nil {
				return err
			}
			// This is bad as we split value even if we are not sure that it matches
			err = setAttributeFromLabel(deploymentID, nodeName, instance, label, strings.Split(value, ","), "networks", "addresses")
			if err != nil {
				return err
			}
		}

		labels, err := updateHostResourcesLabels(&host, e.allocatedResources, true)
		if err != nil {
			return errors.Wrapf(err, "failed updating labels for hostname:%q", host.Name)
		}
		if err := saveLabels(host.Name, labels, hpManager); err != nil {
			return errors.Wrapf(err, "failed saving labels for hostname:%q", host.Name)
		}
	}

	return nil
}

func (e *defaultExecutor) getAllocatedResourcesFromHostCapabilities(kv *api.KV, deploymentID, nodeName string) (*hostResources, error) {
	res := &hostResources{}
	found, p, err := deployments.GetCapabilityProperty(kv, deploymentID, nodeName, "host", "num_cpus")
	if err != nil {
		return nil, err
	}
	if found && p != "" {
		c, err := strconv.Atoi(p)
		if err != nil {
			return nil, err
		}
		res.cpus = int64(c)
	}

	found, p, err = deployments.GetCapabilityProperty(kv, deploymentID, nodeName, "host", "mem_size")
	if err != nil {
		return nil, err
	}
	if found && p != "" {
		c, err := humanize.ParseBytes(p)
		if err != nil {
			return nil, err
		}
		res.memSize = int64(c)
	}

	found, p, err = deployments.GetCapabilityProperty(kv, deploymentID, nodeName, "host", "disk_size")
	if err != nil {
		return nil, err
	}
	if found && p != "" {
		c, err := humanize.ParseBytes(p)
		if err != nil {
			return nil, err
		}
		res.diskSize = int64(c)
	}
	return res, nil
}

func appendCapabilityFilter(kv *api.KV, deploymentID, nodeName, capName, propName, op string, filters []labelsutil.Filter) ([]labelsutil.Filter, error) {
	found, p, err := deployments.GetCapabilityProperty(kv, deploymentID, nodeName, capName, propName)
	if err != nil {
		return filters, err
	}
	if found && p != "" {
		f, err := labelsutil.CreateFilter(capName + "." + propName + " " + op + " " + p)
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

func (e *defaultExecutor) hostsPoolDelete(originalCtx context.Context, cc *api.Client, cfg config.Configuration, taskID, deploymentID, nodeName string) error {
	hpManager := NewManager(cc)
	instances, err := tasks.GetInstances(cc.KV(), taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}
	var errs error
	for _, instance := range instances {
		logOptFields, _ := events.FromContext(originalCtx)
		logOptFields[events.InstanceID] = instance
		ctx := events.NewContext(originalCtx, logOptFields)
		found, hostname, err := deployments.GetInstanceAttribute(cc.KV(), deploymentID, nodeName, instance, "hostname")
		if err != nil {
			errs = multierror.Append(errs, err)
		}
		if !found {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).Registerf("instance %q of node %q does not have a registered hostname. This may be due to an error at creation time. Should be checked.", instance, nodeName)
			continue
		}
		allocation, err := NewAllocation(nodeName, instance, deploymentID)
		if err != nil {
			return err
		}
		err = hpManager.Release(hostname, allocation)
		if err != nil {
			errs = multierror.Append(errs, err)
		}

		host, err := hpManager.GetHost(hostname)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
		if &host != nil {
			labels, err := updateHostResourcesLabels(&host, e.allocatedResources, false)
			if err != nil {
				errs = multierror.Append(errs, errors.Wrapf(err, "failed updating labels for hostname:%q", host.Name))
			}
			if err := saveLabels(host.Name, labels, hpManager); err != nil {
				errs = multierror.Append(errs, errors.Wrapf(err, "failed saving labels for hostname:%q", host.Name))
			}
		}

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

func updateHostResourcesLabels(host *Host, allocResources *hostResources, allocate bool) (map[string]string, error) {
	labels := make(map[string]string)

	// Host Resources Labels can only be updated when deployment resources requirement is described
	if allocResources.cpus != 0 {
		if cpusNbStr, ok := host.Labels["host.num_cpus"]; ok {
			cpus, err := strconv.Atoi(cpusNbStr)
			if err != nil {
				return nil, err
			}

			res := allocOrRelease(int64(cpus), int64(allocResources.cpus), allocate)
			if allocate && res < 0 {
				return nil, errors.Errorf("Illegal allocation : not enough cpus for host:%q", host.Name)
			}
			labels["host.num_cpus"] = strconv.Itoa(int(res))
		}
	}

	if allocResources.memSize != 0 {
		if memSizeStr, ok := host.Labels["host.mem_size"]; ok {
			memSize, err := humanize.ParseBytes(memSizeStr)
			if err != nil {
				return nil, err
			}

			res := allocOrRelease(int64(memSize), allocResources.memSize, allocate)
			if allocate && res < 0 {
				return nil, errors.Errorf("Illegal allocation : not enough memory for host:%q", host.Name)
			}
			labels["host.mem_size"] = formatBytes(res, isIECformat(memSizeStr))
		}
	}

	if allocResources.diskSize != 0 {
		if diskSizeStr, ok := host.Labels["host.disk_size"]; ok {
			diskSize, err := humanize.ParseBytes(diskSizeStr)
			if err != nil {
				return nil, err
			}

			res := allocOrRelease(int64(diskSize), allocResources.diskSize, allocate)
			if allocate && res < 0 {
				return nil, errors.Errorf("Illegal allocation : not enough disk space for host:%q", host.Name)
			}
			labels["host.disk_size"] = formatBytes(res, isIECformat(diskSizeStr))
		}
	}

	return labels, nil
}

func allocOrRelease(valA int64, valB int64, alloc bool) int64 {
	var res int64
	if alloc {
		res = valA - valB
	} else {
		res = valA + valB
	}
	return res
}

func saveLabels(hostname string, labels map[string]string, hpManager Manager) error {
	keys := make([]string, 0)
	for k := range labels {
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return nil
	}

	// Maybe updating labels is a better solution...
	if err := hpManager.RemoveLabels(hostname, keys); err != nil {
		return err
	}
	if err := hpManager.AddLabels(hostname, labels); err != nil {
		return err
	}
	return nil
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
