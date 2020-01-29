// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/labelsutil"
	"github.com/ystia/yorc/v4/tosca"
)

func setInstancesStateWithContextualLogs(ctx context.Context, op operationParameters, instances []string, state tosca.NodeState) {

	for _, instance := range instances {
		deployments.SetInstanceStateWithContextualLogs(events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.InstanceID: instance}), op.deploymentID, op.nodeName, instance, state)
	}
}

func setInstanceAttributesValue(ctx context.Context, op operationParameters, instance, value string, attributes []string) error {
	for _, attr := range attributes {
		err := deployments.SetInstanceAttribute(ctx, op.deploymentID, op.nodeName, instance,
			attr, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func setInstanceAttributesFromLabels(ctx context.Context, op operationParameters, instance string, labels map[string]string) error {
	for label, value := range labels {
		err := setAttributeFromLabel(ctx, op.deploymentID, op.nodeName, instance,
			label, value, tosca.ComputeNodeNetworksAttributeName, tosca.NetworkNameProperty)
		if err != nil {
			return err
		}
		err = setAttributeFromLabel(ctx, op.deploymentID, op.nodeName, instance,
			label, value, tosca.ComputeNodeNetworksAttributeName, tosca.NetworkIDProperty)
		if err != nil {
			return err
		}
		// This is bad as we split value even if we are not sure that it matches
		err = setAttributeFromLabel(ctx, op.deploymentID, op.nodeName, instance,
			label, strings.Split(value, ","), tosca.ComputeNodeNetworksAttributeName,
			tosca.NetworkAddressesProperty)
		if err != nil {
			return err
		}
	}
	return nil
}

func appendCapabilityFilter(ctx context.Context, deploymentID, nodeName, capName, propName, op string, filters []labelsutil.Filter) ([]labelsutil.Filter, error) {
	p, err := deployments.GetCapabilityPropertyValue(ctx, deploymentID, nodeName, capName, propName)
	if err != nil {
		return filters, err
	}

	hasProp, propDataType, err := deployments.GetCapabilityPropertyType(ctx, deploymentID, nodeName, capName, propName)
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

func createFiltersFromComputeCapabilities(ctx context.Context, deploymentID, nodeName string) ([]labelsutil.Filter, error) {
	var err error
	filters := make([]labelsutil.Filter, 0)

	filtersParams := []struct{
		capabilityName string
		propertyName string
		operator string
	}{
		{"host", "num_cpus", ">="},
		{"host", "cpu_frequency", ">="},
		{"host", "disk_size", ">="},
		{"host", "mem_size", ">="},
		{"os", "architecture", "="},
		{"os", "type", "="},
		{"os", "distribution", "="},
		{"os", "version", "="},
	}

	for _, filterParam := range filtersParams {
		filters, err = appendCapabilityFilter(ctx, deploymentID, nodeName, filterParam.capabilityName, filterParam.propertyName, filterParam.operator, filters)
		if err != nil {
			return nil, err
		}
	}
	return filters, nil
}

func setAttributeFromLabel(ctx context.Context, deploymentID, nodeName, instance, label string, value interface{}, prefix, suffix string) error {
	if strings.HasPrefix(label, prefix+".") && strings.HasSuffix(label, "."+suffix) {
		attrName := strings.Replace(strings.Replace(label, prefix+".", prefix+"/", -1), "."+suffix, "/"+suffix, -1)
		err := deployments.SetInstanceAttributeComplex(ctx, deploymentID, nodeName, instance, attrName, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateResourcesLabels(origin map[string]string, diff map[string]string, operation func(a int64, b int64) int64) (map[string]string, error) {
	labels := make(map[string]string)

	// Host Resources Labels can only be updated when deployment resources requirement is described

	intResourcesLabels := []struct{
		name string
	}{
		{"host.num_cpus"},
	}

	for _, resource := range intResourcesLabels {
		if resourceDiffStr, ok := diff[resource.name]; ok {
			if resourceOriginStr, ok := origin[resource.name]; ok {
				resourceOrigin, err := strconv.Atoi(resourceOriginStr)
				if err != nil {
					return nil, err
				}
				resourceDiff, err := strconv.Atoi(resourceDiffStr)
				if err != nil {
					return nil, err
				}

				res := operation(int64(resourceOrigin), int64(resourceDiff))
				labels[resource.name] = strconv.Itoa(int(res))
			}
		}
	}

	sizeResourcesLabels := []struct{
		name string
	}{
		{"host.mem_size"},
		{"host.disk_size"},
	}

	for _, resource := range sizeResourcesLabels {
		if resourceDiffStr, ok := diff[resource.name]; ok {
			if resourceOriginStr, ok := origin[resource.name]; ok {
				resourceOrigin, err := humanize.ParseBytes(resourceOriginStr)
				if err != nil {
					return nil, err
				}
				resourceDiff, err := humanize.ParseBytes(resourceDiffStr)
				if err != nil {
					return nil, err
				}

				res := operation(int64(resourceOrigin), int64(resourceDiff))
				labels[resource.name] = formatBytes(res, isIECformat(resourceOriginStr))
			}
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
