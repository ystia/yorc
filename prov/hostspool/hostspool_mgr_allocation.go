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
	"encoding/json"
	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/log"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/helper/labelsutil"
)

const (
	binPackingPlacement     = "yorc.policies.hostspool.BinPackingPlacement"
	weightBalancedPlacement = "yorc.policies.hostspool.WeightBalancedPlacement"
	placementPolicy         = "yorc.policies.hostspool.Placement"
)

type hostCandidate struct {
	name        string
	allocations int
}

func (cm *consulManager) Allocate(locationName string, allocation *Allocation, filters ...labelsutil.Filter) (string, []labelsutil.Warning, error) {
	return cm.allocateWait(locationName, maxWaitTimeSeconds*time.Second, allocation, filters...)
}
func (cm *consulManager) allocateWait(locationName string, maxWaitTime time.Duration, allocation *Allocation, filters ...labelsutil.Filter) (string, []labelsutil.Warning, error) {
	// Build allocationID
	if err := allocation.buildID(); err != nil {
		return "", nil, err
	}

	lockCh, cleanupFn, err := cm.lockKey(locationName, "", "allocation", maxWaitTime)
	if err != nil {
		return "", nil, err
	}
	defer cleanupFn()

	hosts, warnings, _, err := cm.List(locationName, filters...)
	if err != nil {
		return "", warnings, err
	}
	// define host candidates in only free or allocated hosts in case of shareable allocation
	candidates := make([]hostCandidate, 0)
	var lastErr error
	for _, h := range hosts {
		select {
		case <-lockCh:
			return "", warnings, errors.New("admin lock lost on hosts pool during host allocation")
		default:
		}
		err := cm.checkConnection(locationName, h)
		if err != nil {
			lastErr = err
			continue
		}
		hs, err := cm.GetHostStatus(locationName, h)
		if err != nil {
			lastErr = err
		} else {
			if hs == HostStatusFree {
				candidates = append(candidates, hostCandidate{
					name:        h,
					allocations: 0,
				})
			} else if hs == HostStatusAllocated && allocation.Shareable {
				allocations, err := cm.GetAllocations(locationName, h)
				if err != nil {
					lastErr = err
					continue
				}
				// Check the host allocation is not shareable
				if len(allocations) == 1 && !allocations[0].Shareable {
					continue
				}
				candidates = append(candidates, hostCandidate{
					name:        h,
					allocations: len(allocations),
				})
			}
		}
	}

	if len(candidates) == 0 {
		if lastErr != nil {
			return "", warnings, lastErr
		}
		return "", warnings, errors.WithStack(noMatchingHostFoundError{})
	}

	// Apply the policy placement
	hostname := cm.electHostFromCandidates(locationName, allocation, candidates)
	select {
	case <-lockCh:
		return "", warnings, errors.New("admin lock lost on hosts pool during host allocation")
	default:
	}

	if err := cm.addAllocation(locationName, hostname, allocation); err != nil {
		return "", warnings, errors.Wrapf(err, "failed to add allocation for hostname:%q", hostname)
	}

	return hostname, warnings, cm.setHostStatus(locationName, hostname, HostStatusAllocated)
}
func (cm *consulManager) electHostFromCandidates(locationName string, allocation *Allocation, candidates []hostCandidate) string {
	switch allocation.PlacementPolicy {
	case weightBalancedPlacement:
		log.Printf("Applying weight-balanced placement policy for location:%s, deployment:%s, node name:%s, instance:%s", locationName, allocation.DeploymentID, allocation.NodeName, allocation.Instance)
		return weightBalanced(candidates)
	case binPackingPlacement:
		log.Printf("Applying bin packing placement policy for location:%s, deployment:%s, node name:%s, instance:%s", locationName, allocation.DeploymentID, allocation.NodeName, allocation.Instance)
		return binPacking(candidates)
	default: // default is bin packing placement
		log.Printf("Applying default bin packing placement policy for location:%s, deployment:%s, node name:%s, instance:%s", locationName, allocation.DeploymentID, allocation.NodeName, allocation.Instance)
		return binPacking(candidates)
	}
}

func weightBalanced(candidates []hostCandidate) string {
	hostname := candidates[0].name
	minAllocations := candidates[0].allocations
	for _, candidate := range candidates {
		if candidate.allocations < minAllocations {
			minAllocations = candidate.allocations
			hostname = candidate.name
		}
	}
	return hostname
}

func binPacking(candidates []hostCandidate) string {
	hostname := candidates[0].name
	maxAllocations := candidates[0].allocations
	for _, candidate := range candidates {
		if candidate.allocations > maxAllocations {
			maxAllocations = candidate.allocations
			hostname = candidate.name
		}
	}
	return hostname
}

func (cm *consulManager) Release(locationName, hostname, deploymentID, nodeName, instance string) (*Allocation, error) {
	return cm.releaseWait(locationName, hostname, deploymentID, nodeName, instance, maxWaitTimeSeconds*time.Second)
}

func (cm *consulManager) releaseWait(locationName, hostname, deploymentID, nodeName, instance string, maxWaitTime time.Duration) (*Allocation, error) {
	_, cleanupFn, err := cm.lockKey(locationName, hostname, "release", maxWaitTime)
	if err != nil {
		return nil, err
	}
	defer cleanupFn()

	// Need to retrieve complete information about allocation for resources updates
	allocation, err := cm.GetAllocation(locationName, hostname, buildAllocationID(deploymentID, nodeName, instance))
	if err != nil {
		return nil, err
	}
	if err := cm.removeAllocation(locationName, hostname, allocation); err != nil {
		return nil, errors.Wrapf(err, "failed to remove allocation with ID:%q, hostname:%q, location: %q", allocation.ID, hostname, locationName)
	}

	host, err := cm.GetHost(locationName, hostname)
	if err != nil {
		return nil, err
	}
	// Set the host status to free only for host with no allocations
	if len(host.Allocations) == 0 {
		if err = cm.setHostStatus(locationName, hostname, HostStatusFree); err != nil {
			return nil, err
		}
	}
	err = cm.checkConnection(locationName, hostname)
	if err != nil {
		cm.backupHostStatus(locationName, hostname)
		cm.setHostStatusWithMessage(locationName, hostname, HostStatusError, "failed to connect to host")
	}
	return allocation, nil
}

func getKVTxnOp(verb api.KVOp, key string, value []byte) *api.KVTxnOp {
	return &api.KVTxnOp{
		Verb:  verb,
		Key:   key,
		Value: value,
	}
}

func getAddAllocationsOperation(locationName, hostname string, allocations []Allocation) (api.KVTxnOps, error) {
	allocsOps := api.KVTxnOps{}
	hostKVPrefix := path.Join(consulutil.HostsPoolPrefix, locationName, hostname)
	if allocations != nil {
		for _, alloc := range allocations {
			allocKVPrefix := path.Join(hostKVPrefix, "allocations", alloc.ID)
			allocOps := api.KVTxnOps{
				getKVTxnOp(api.KVSet, path.Join(allocKVPrefix), []byte(alloc.ID)),
				getKVTxnOp(api.KVSet, path.Join(allocKVPrefix, "node_name"), []byte(alloc.NodeName)),
				getKVTxnOp(api.KVSet, path.Join(allocKVPrefix, "instance"), []byte(alloc.Instance)),
				getKVTxnOp(api.KVSet, path.Join(allocKVPrefix, "deployment_id"), []byte(alloc.DeploymentID)),
				getKVTxnOp(api.KVSet, path.Join(allocKVPrefix, "shareable"), []byte(strconv.FormatBool(alloc.Shareable))),
				getKVTxnOp(api.KVSet, path.Join(allocKVPrefix, "placement_policy"), []byte(alloc.PlacementPolicy)),
			}

			for k, v := range alloc.Resources {
				k = url.PathEscape(k)
				if k == "" {
					return nil, errors.WithStack(badRequestError{"empty labels are not allowed"})
				}
				allocOps = append(allocOps, getKVTxnOp(api.KVSet, path.Join(allocKVPrefix, "resources", k), []byte(v)))
			}

			for _, gResource := range alloc.GenericResources {
				// save generic resource in JSON
				data, err := json.Marshal(gResource)
				if err != nil {
					return nil, err
				}
				allocOps = append(allocOps, getKVTxnOp(api.KVSet, path.Join(allocKVPrefix, "generic_resources", gResource.Name), data))
			}
			allocsOps = append(allocsOps, allocOps...)
		}
	}
	return allocsOps, nil
}

func (cm *consulManager) addAllocation(locationName, hostname string, allocation *Allocation) error {
	var allocOps api.KVTxnOps
	var err error

	if err = cm.allocateGenericResources(locationName, hostname, allocation); err != nil {
		return err
	}

	if allocOps, err = getAddAllocationsOperation(locationName, hostname, []Allocation{*allocation}); err != nil {
		return errors.Wrapf(err, "failed to add allocation to host:%q, location: %q", hostname, locationName)
	}

	ok, response, _, err := cm.cc.KV().Txn(allocOps, nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !ok {
		// Check the response
		errs := make([]string, 0)
		for _, e := range response.Errors {
			errs = append(errs, e.What)
		}
		return errors.Errorf("Failed to add allocation on host %q, location %q: %s", hostname, locationName, strings.Join(errs, ", "))
	}
	return nil
}

func (cm *consulManager) allocateGenericResources(locationName, hostname string, allocation *Allocation) error {
	if allocation.GenericResources == nil {
		return nil
	}

	// Retrieve host generic resources labels
	labels, err := cm.GetHostLabels(locationName, hostname)
	if err != nil {
		return err
	}
	for i := range allocation.GenericResources {
		gResource := allocation.GenericResources[i]
		hostGenResourceStr, ok := labels[gResource.Label]
		if !ok {
			return errors.Errorf("failed to retrieve generic resource:%q for location:%q, host:%s", gResource.Name, locationName, hostname)
		}
		hostGenResource := strings.Split(hostGenResourceStr, ",")

		value := make([]string, 0)
		if gResource.ids != nil {
			// Check list contains ids
			for _, id := range gResource.ids {
				if !collections.ContainsString(hostGenResource, id) {
					return errors.Errorf("missing expected id:%q for generic resource:%q, location:%q, host:%s", id, gResource.Name, locationName, hostname)
				}
			}
			value = gResource.ids
		} else {
			if len(hostGenResource) < gResource.nb {
				return errors.Errorf("missing %d expected resources for generic resource:%q, location:%q, host:%s", gResource.nb-len(hostGenResource), gResource.Name, locationName, hostname)
			}
			value = hostGenResource[:gResource.nb]
		}
		gResource.Value = strings.Join(value, ",")
	}
	return nil
}

func (cm *consulManager) removeAllocation(locationName, hostname string, allocation *Allocation) error {
	_, err := cm.cc.KV().DeleteTree(path.Join(consulutil.HostsPoolPrefix, locationName, hostname, "allocations", allocation.ID)+"/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	_, err = cm.cc.KV().Delete(path.Join(consulutil.HostsPoolPrefix, locationName, hostname, "allocations", allocation.ID), nil)
	return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
}

func exist(allocations []Allocation, ID string) bool {
	for _, alloc := range allocations {
		if alloc.ID == ID {
			return true
		}
	}
	return false
}

func (cm *consulManager) GetAllocations(locationName, hostname string) ([]Allocation, error) {
	allocations := make([]Allocation, 0)
	if locationName == "" {
		return nil, errors.WithStack(badRequestError{`"locationName" missing`})
	}
	if hostname == "" {
		return nil, errors.WithStack(badRequestError{`"hostname" missing`})
	}
	keys, _, err := cm.cc.KV().Keys(path.Join(consulutil.HostsPoolPrefix, locationName, hostname, "allocations")+"/", "/", nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	for _, key := range keys {
		id := path.Base(key)
		if exist(allocations, id) {
			continue
		}
		alloc := Allocation{}
		alloc.ID = id

		stringFields := []struct {
			name string
			attr *string
		}{
			{"node_name", &alloc.NodeName},
			{"instance", &alloc.Instance},
			{"deployment_id", &alloc.DeploymentID},
			{"placement_policy", &alloc.PlacementPolicy},
		}

		for _, field := range stringFields {
			kvp, _, err := cm.cc.KV().Get(path.Join(key, field.name), nil)
			if err != nil {
				return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
			}
			if kvp != nil && len(kvp.Value) > 0 {
				*field.attr = string(kvp.Value)
			}
		}

		kvp, _, err := cm.cc.KV().Get(path.Join(key, "shareable"), nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp != nil && len(kvp.Value) > 0 {
			alloc.Shareable, err = strconv.ParseBool(string(kvp.Value))
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse boolean from value:%q", string(kvp.Value))
			}
		}
		// Appending a final "/" here is not necessary as there is no other keys starting with "resources" prefix
		kvps, _, err := cm.cc.KV().List(path.Join(key, "resources"), nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		resources := make(map[string]string, len(kvps))
		for _, kvp := range kvps {
			resources[path.Base(kvp.Key)] = string(kvp.Value)
		}
		alloc.Resources = resources

		// Retrieve generic resources
		kvps, _, err = cm.cc.KV().List(path.Join(key, "generic_resources"), nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		gResources := make([]*GenericResource, 0)
		for _, kvp := range kvps {
			var gResource GenericResource
			err = json.Unmarshal(kvp.Value, &gResource)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to unmarshal generic resource for key:%q", kvp.Key)
			}
			gResources = append(gResources, &gResource)
		}
		alloc.GenericResources = gResources
		allocations = append(allocations, alloc)
	}
	return allocations, nil
}

func (cm *consulManager) GetAllocation(locationName, hostname, allocationID string) (*Allocation, error) {
	if locationName == "" {
		return nil, errors.WithStack(badRequestError{`"locationName" missing`})
	}
	if hostname == "" {
		return nil, errors.WithStack(badRequestError{`"hostname" missing`})
	}
	if allocationID == "" {
		return nil, errors.WithStack(badRequestError{`"allocationID" missing`})
	}

	allocs, err := cm.GetAllocations(locationName, hostname)
	if err != nil {
		return nil, err
	}

	for i := range allocs {
		if allocs[i].ID == allocationID {
			return &allocs[i], nil
		}
	}

	return nil, errors.Errorf("failed to retrieve allocation with ID:%q for location name:%q, hostname:%q", allocationID, locationName, hostname)
}

// CheckPlacementPolicy check if placement policy is supported
func (cm *consulManager) CheckPlacementPolicy(placementPolicy string) error {
	if placementPolicy == "" {
		return nil
	}

	switch placementPolicy {
	case weightBalancedPlacement, binPackingPlacement:
		return nil
	default:
		return errors.Errorf("placement policy:%q is not actually supported", placementPolicy)
	}
}
