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
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/helper/labelsutil"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

func (cm *consulManager) Allocate(allocation *Allocation, filters ...labelsutil.Filter) (string, []labelsutil.Warning, error) {
	return cm.allocateWait(maxWaitTimeSeconds*time.Second, allocation, filters...)
}
func (cm *consulManager) allocateWait(maxWaitTime time.Duration, allocation *Allocation, filters ...labelsutil.Filter) (string, []labelsutil.Warning, error) {
	// Build allocationID
	if err := allocation.buildID(); err != nil {
		return "", nil, err
	}

	lockCh, cleanupFn, err := cm.lockKey("", "allocation", maxWaitTime)
	if err != nil {
		return "", nil, err
	}
	defer cleanupFn()

	hosts, warnings, _, err := cm.List(filters...)
	if err != nil {
		return "", warnings, err
	}
	// Filters only free or allocated hosts in case of shareable allocation
	var lastErr error
	freeHosts := hosts[:0]
	for _, h := range hosts {
		select {
		case <-lockCh:
			return "", warnings, errors.New("admin lock lost on hosts pool during host allocation")
		default:
		}
		err := cm.checkConnection(h)
		if err != nil {
			lastErr = err
			continue
		}
		hs, err := cm.GetHostStatus(h)
		if err != nil {
			lastErr = err
		} else {
			if hs == HostStatusFree {
				freeHosts = append(freeHosts, h)
			} else if hs == HostStatusAllocated && allocation.Shareable {
				allocations, err := cm.GetAllocations(h)
				if err != nil {
					lastErr = err
					continue
				}
				// Check the host allocation is not unshareable
				if len(allocations) == 1 && !allocations[0].Shareable {
					continue
				}
				freeHosts = append(freeHosts, h)
			}
		}
	}

	if len(freeHosts) == 0 {
		if lastErr != nil {
			return "", warnings, lastErr
		}
		return "", warnings, errors.WithStack(noMatchingHostFoundError{})
	}
	// Get the first host that match
	hostname := freeHosts[0]
	select {
	case <-lockCh:
		return "", warnings, errors.New("admin lock lost on hosts pool during host allocation")
	default:
	}

	if err := cm.addAllocation(hostname, allocation); err != nil {
		return "", warnings, errors.Wrapf(err, "failed to add allocation for hostname:%q", hostname)
	}

	return hostname, warnings, cm.setHostStatus(hostname, HostStatusAllocated)
}
func (cm *consulManager) Release(hostname string, allocation *Allocation) error {
	return cm.releaseWait(hostname, allocation, maxWaitTimeSeconds*time.Second)
}

func (cm *consulManager) releaseWait(hostname string, allocation *Allocation, maxWaitTime time.Duration) error {
	// Build allocationID
	if err := allocation.buildID(); err != nil {
		return err
	}
	_, cleanupFn, err := cm.lockKey(hostname, "release", maxWaitTime)
	if err != nil {
		return err
	}
	defer cleanupFn()

	if err := cm.removeAllocation(hostname, allocation); err != nil {
		return errors.Wrapf(err, "failed to remove allocation with ID:%q and hostname:%q", allocation.ID, hostname)
	}

	host, err := cm.GetHost(hostname)
	if err != nil {
		return err
	}
	// Set the host status to free only for host with no allocations
	if len(host.Allocations) == 0 {
		if err = cm.setHostStatus(hostname, HostStatusFree); err != nil {
			return err
		}
	}
	err = cm.checkConnection(hostname)
	if err != nil {
		cm.backupHostStatus(hostname)
		cm.setHostStatusWithMessage(hostname, HostStatusError, "failed to connect to host")
	}
	return nil
}

func getAddAllocationsOperation(hostname string, allocations []Allocation) (api.KVTxnOps, error) {
	allocsOps := api.KVTxnOps{}
	hostKVPrefix := path.Join(consulutil.HostsPoolPrefix, hostname)
	if allocations != nil {
		for _, alloc := range allocations {
			allocKVPrefix := path.Join(hostKVPrefix, "allocations", alloc.ID)
			allocOps := api.KVTxnOps{
				&api.KVTxnOp{
					Verb:  api.KVSet,
					Key:   path.Join(allocKVPrefix),
					Value: []byte(alloc.ID),
				},
				&api.KVTxnOp{
					Verb:  api.KVSet,
					Key:   path.Join(allocKVPrefix, "node_name"),
					Value: []byte(alloc.NodeName),
				},
				&api.KVTxnOp{
					Verb:  api.KVSet,
					Key:   path.Join(allocKVPrefix, "instance"),
					Value: []byte(alloc.Instance),
				},
				&api.KVTxnOp{
					Verb:  api.KVSet,
					Key:   path.Join(allocKVPrefix, "deployment_id"),
					Value: []byte(alloc.DeploymentID),
				},
				&api.KVTxnOp{
					Verb:  api.KVSet,
					Key:   path.Join(allocKVPrefix, "shareable"),
					Value: []byte(strconv.FormatBool(alloc.Shareable)),
				},
			}

			for k, v := range alloc.Resources {
				k = url.PathEscape(k)
				if k == "" {
					return nil, errors.WithStack(badRequestError{"empty labels are not allowed"})
				}
				allocOps = append(allocOps, &api.KVTxnOp{
					Verb:  api.KVSet,
					Key:   path.Join(allocKVPrefix, "resources", k),
					Value: []byte(v),
				})
			}

			allocsOps = append(allocsOps, allocOps...)
		}
	}
	return allocsOps, nil
}

func (cm *consulManager) addAllocation(hostname string, allocation *Allocation) error {
	var allocOps api.KVTxnOps
	var err error
	if allocOps, err = getAddAllocationsOperation(hostname, []Allocation{*allocation}); err != nil {
		return errors.Wrapf(err, "failed to add allocation to host:%q", hostname)
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
		return errors.Errorf("Failed to add allocation on host %q: %s", hostname, strings.Join(errs, ", "))
	}
	return nil
}

func (cm *consulManager) removeAllocation(hostname string, allocation *Allocation) error {
	_, err := cm.cc.KV().DeleteTree(path.Join(consulutil.HostsPoolPrefix, hostname, "allocations", allocation.ID), nil)
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

func (cm *consulManager) GetAllocations(hostname string) ([]Allocation, error) {
	allocations := make([]Allocation, 0)
	if hostname == "" {
		return nil, errors.WithStack(badRequestError{`"hostname" missing`})
	}
	keys, _, err := cm.cc.KV().Keys(path.Join(consulutil.HostsPoolPrefix, hostname, "allocations")+"/", "/", nil)
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
		kvp, _, err := cm.cc.KV().Get(path.Join(key, "node_name"), nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp != nil && len(kvp.Value) > 0 {
			alloc.NodeName = string(kvp.Value)
		}

		kvp, _, err = cm.cc.KV().Get(path.Join(key, "instance"), nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp != nil && len(kvp.Value) > 0 {
			alloc.Instance = string(kvp.Value)
		}

		kvp, _, err = cm.cc.KV().Get(path.Join(key, "deployment_id"), nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp != nil && len(kvp.Value) > 0 {
			alloc.DeploymentID = string(kvp.Value)
		}

		kvp, _, err = cm.cc.KV().Get(path.Join(key, "shareable"), nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp != nil && len(kvp.Value) > 0 {
			alloc.Shareable, err = strconv.ParseBool(string(kvp.Value))
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse boolean from value:%q", string(kvp.Value))
			}
		}

		kvps, _, err := cm.cc.KV().List(path.Join(key, "resources"), nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		resources := make(map[string]string, len(kvps))
		for _, kvp := range kvps {
			resources[path.Base(kvp.Key)] = string(kvp.Value)
		}
		alloc.Resources = resources

		allocations = append(allocations, alloc)
	}
	return allocations, nil
}
