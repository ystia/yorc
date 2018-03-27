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
	"fmt"
	"strings"

	"github.com/ystia/yorc/helper/labelsutil"

	"github.com/hashicorp/go-multierror"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/tosca"
	"strconv"
)

type defaultExecutor struct{}

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
		hostname, warnings, err := hpManager.Allocate(fmt.Sprintf(`allocated for node instance "%s-%s" in deployment %q`, nodeName, instance, deploymentID), filters...)
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
	}

	return nil
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
		err = hpManager.Release(hostname)
		if err != nil {
			errs = multierror.Append(errs, err)
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
