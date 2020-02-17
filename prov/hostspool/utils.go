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
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/helper/collections"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/labelsutil"
	"github.com/ystia/yorc/v4/log"
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

	filtersParams := []struct {
		capabilityName string
		propertyName   string
		operator       string
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

// createGenericResourcesFilters create regex filters to determine if label contains the required ids or required number of elements
func createGenericResourcesFilters(ctx context.Context, instance string, genericResources []*GenericResource) ([]labelsutil.Filter, error) {
	filters := make([]labelsutil.Filter, 0)
	for _, genericResource := range genericResources {
		instance, err := strconv.Atoi(instance)
		if err != nil {
			return nil, errors.Wrapf(err, "unexpected non integer value for instance:%q", instance)
		}
		if genericResource.ids != nil && len(genericResource.ids) > instance {
			idFilters, err := appendGenericResourceFiltersOnIDs(genericResource.Name, genericResource.ids[instance])
			if err != nil {
				return filters, err
			}
			filters = append(filters, idFilters...)
		}

		if genericResource.nb != 0 {
			filter, err := appendGenericResourceFilterOnNumber(genericResource.Name, genericResource.nb)
			if err != nil {
				return filters, err
			}
			filters = append(filters, filter)
		}
	}

	return filters, nil
}

// appendGenericResourceFiltersOnIDs create regex filters to determine if label contains the required ids
func appendGenericResourceFiltersOnIDs(name string, ids []string) ([]labelsutil.Filter, error) {
	filters := make([]labelsutil.Filter, 0)
	for _, id := range ids {
		log.Debugf("Create filter for id:%q", id)
		// regex expressing id is contained on the comma-separated list
		f := fmt.Sprintf(`%s.%s ~= "^%s,|,%s,|,%s$|^%s$"`, genericResourceLabelPrefix, name, id, id, id, id)
		filter, err := labelsutil.CreateFilter(f)
		if err != nil {
			return filters, err
		}
		filters = append(filters, filter)
	}
	return filters, nil
}

// appendGenericResourceFilterOnNumber create a regex filter to determine if label contains the required number of elements
func appendGenericResourceFilterOnNumber(name string, nb int) (labelsutil.Filter, error) {
	var f string
	// host.resource.<name> contains at least <nb> elements for nb > 1
	if nb > 1 {
		f = fmt.Sprintf(`%s.%s ~= "^([^,]*,){%d}.*$"`, genericResourceLabelPrefix, name, nb-1)
	} else {
		f = fmt.Sprintf(`%s.%s ~= "^(.+)$"`, genericResourceLabelPrefix, name)
	}

	return labelsutil.CreateFilter(f)
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

func removeWhitespaces(str string) string {
	return strings.Join(strings.Fields(str), "")
}

func toSlice(str string) []string {
	str = removeWhitespaces(str)
	if str == "" {
		return make([]string, 0)
	}
	return strings.Split(str, ",")
}

func updateGenericResourcesLabels(origin map[string]string, diffGenericResources []*GenericResource, operation genericResourceOperationFunc) map[string]string {
	updatedLabels := make(map[string]string)
	for _, diffGenericResource := range diffGenericResources {
		if !diffGenericResource.NoConsumable {
			if genResourceStr, ok := origin[diffGenericResource.Label]; ok {
				genResource := toSlice(genResourceStr)
				genResourceElements := toSlice(diffGenericResource.Value)
				result := operation(genResource, genResourceElements)
				updatedLabels[diffGenericResource.Label] = strings.Join(result, ",")
			}
		}
	}
	return updatedLabels
}

func updateResourcesLabels(origin map[string]string, diff map[string]string, operation resourceOperationFunc) (map[string]string, error) {
	// Host Resources Labels can only be updated when deployment resources requirement is described
	updatedLabels := make(map[string]string)

	err := updateNumberResourcesLabels(origin, diff, operation, updatedLabels)
	if err != nil {
		return nil, err
	}

	err = updateSizeResourcesLabels(origin, diff, operation, updatedLabels)
	if err != nil {
		return nil, err
	}
	return updatedLabels, nil
}

func updateNumberResourcesLabels(origin map[string]string, diff map[string]string, operation resourceOperationFunc, updatedLabels map[string]string) error {
	cpuResourcesLabel := "host.num_cpus"
	if resourceDiffStr, ok := diff[cpuResourcesLabel]; ok {
		if resourceOriginStr, ok := origin[cpuResourcesLabel]; ok {
			resourceOrigin, err := strconv.Atoi(resourceOriginStr)
			if err != nil {
				return err
			}
			resourceDiff, err := strconv.Atoi(resourceDiffStr)
			if err != nil {
				return err
			}

			res := operation(int64(resourceOrigin), int64(resourceDiff))
			updatedLabels[cpuResourcesLabel] = strconv.Itoa(int(res))
		}
	}
	return nil
}

func updateSizeResourcesLabels(origin map[string]string, diff map[string]string, operation resourceOperationFunc, updatedLabels map[string]string) error {
	sizeResourcesLabels := []struct {
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
					return err
				}
				resourceDiff, err := humanize.ParseBytes(resourceDiffStr)
				if err != nil {
					return err
				}

				res := operation(int64(resourceOrigin), int64(resourceDiff))
				updatedLabels[resource.name] = formatBytes(res, isIECformat(resourceOriginStr))
			}
		}
	}
	return nil
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

func removeElements(source, elements []string) []string {
	for _, element := range elements {
		for i := 0; i < len(source); i++ {
			if strings.TrimSpace(source[i]) == strings.TrimSpace(element) {
				source = append(source[:i], source[i+1:]...)
				break
			}
		}
	}

	sort.Strings(source)
	return source
}

func addElements(source, elements []string) []string {
	for _, element := range elements {
		source = append(source, element)
	}

	source = collections.RemoveDuplicates(source)
	sort.Strings(source)
	return source
}

func toGenericResource(item interface{}) (*GenericResource, error) {
	type gres struct {
		Name   string   `json:"name" mapstructure:"name"`
		IDS    []string `json:"ids,omitempty" mapstructure:"ids"`
		Number string   `json:"number,omitempty" mapstructure:"number"`
	}

	g := new(gres)
	err := mapstructure.Decode(item, g)
	if err != nil {
		return nil, err
	}

	if g.Name == "" {
		return nil, errors.New("Missing generic resource mandatory name")
	}

	if g.IDS == nil && g.Number == "" || (g.IDS != nil && g.Number != "") {
		return nil, errors.Errorf("Either ids or number must be filled to define the resource need gor generic resource name:%q.", g.Name)
	}

	var nb int
	if g.Number != "" {
		nb, err = strconv.Atoi(g.Number)
		if err != nil {
			return nil, errors.Wrapf(err, "expected integer value for number property of generic resource with name:%q", g.Name)
		}
	}

	gResource := GenericResource{
		Name:  g.Name,
		Label: fmt.Sprintf("%s.%s", genericResourceLabelPrefix, g.Name),
		nb:    nb,
	}

	if g.IDS != nil {
		gResource.ids = make([][]string, 0)
		for _, item := range g.IDS {
			// check id
			if !isAllowedID(item) {
				return nil, errors.Errorf("generic resource ID:%q must only contains the following characters: a-zA-Z0-9_:/-", item)
			}
			gResource.ids = append(gResource.ids, toSlice(item))
		}

	}
	return &gResource, nil
}

func isAllowedID(str string) bool {
	re := regexp.MustCompile("^[a-zA-Z0-9_:/-]*$")
	return re.MatchString(strings.TrimSpace(str))
}
