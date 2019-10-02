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
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
)

func (cm *consulManager) AddLabels(location, hostname string, labels map[string]string) error {
	return cm.addLabelsWait(location, hostname, labels, maxWaitTimeSeconds*time.Second)
}

func (cm *consulManager) RemoveLabels(location, hostname string, labels []string) error {
	return cm.removeLabelsWait(location, hostname, labels, maxWaitTimeSeconds*time.Second)
}

func (cm *consulManager) addLabelsWait(location, hostname string, labels map[string]string, maxWaitTime time.Duration) error {
	if hostname == "" {
		return errors.WithStack(badRequestError{`"hostname" missing`})
	}
	if labels == nil || len(labels) == 0 {
		return nil
	}

	_, cleanupFn, err := cm.lockKey(location, hostname, "labels addition", maxWaitTime)
	if err != nil {
		return err
	}
	defer cleanupFn()

	// Checks host existence
	// We don't care about host status for updating labels
	_, err = cm.GetHostStatus(location, hostname)
	if err != nil {
		return err
	}

	return cm.addLabels(location, hostname, labels)
}

func (cm *consulManager) addLabels(location, hostname string, labels map[string]string) error {
	ops, err := cm.getAddUpdatedLabelsOperations(location, hostname, labels)
	if err != nil {
		return err
	}
	ok, response, _, err := cm.cc.KV().Txn(ops, nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !ok {
		// Check the response
		errs := make([]string, 0)
		for _, e := range response.Errors {
			errs = append(errs, e.What)
		}
		return errors.Errorf("Failed to add labels to host %q: %s", hostname, strings.Join(errs, ", "))
	}
	return nil
}

func (cm *consulManager) getAddUpdatedLabelsOperations(location, hostname string, labels map[string]string) (api.KVTxnOps, error) {
	// Get labels operations
	ops, err := cm.getAddLabelsOperations(location, hostname, labels)
	if err != nil {
		return nil, err
	}

	// Get updated labels operations
	upLabelsOps, err := cm.getUpdateResourcesLabelsOperationsOnLabelsChange(location, hostname, labels)
	if err != nil {
		return nil, err
	}
	ops = append(ops, upLabelsOps...)

	return ops, nil
}

func (cm *consulManager) removeLabelsWait(location, hostname string, labels []string, maxWaitTime time.Duration) error {
	if location == "" {
		return errors.WithStack(badRequestError{`"location" missing`})
	}
	if hostname == "" {
		return errors.WithStack(badRequestError{`"hostname" missing`})
	}
	if labels == nil || len(labels) == 0 {
		return nil
	}

	hostKVPrefix := path.Join(consulutil.HostsPoolPrefix, location, hostname)
	ops := make(api.KVTxnOps, 0)

	for _, v := range labels {
		v = url.PathEscape(v)
		if v == "" {
			return errors.WithStack(badRequestError{"empty labels are not allowed"})
		}
		ops = append(ops, &api.KVTxnOp{
			Verb: api.KVDelete,
			Key:  path.Join(hostKVPrefix, "labels", v),
		})
	}

	_, cleanupFn, err := cm.lockKey(location, hostname, "labels remove", maxWaitTime)
	if err != nil {
		return err
	}
	defer cleanupFn()

	// Checks host existence
	_, err = cm.GetHostStatus(location, hostname)
	if err != nil {
		return err
	}

	// We don't care about host status for updating labels

	ok, response, _, err := cm.cc.KV().Txn(ops, nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !ok {
		// Check the response
		errs := make([]string, 0)
		for _, e := range response.Errors {
			errs = append(errs, e.What)
		}
		return errors.Errorf("Failed to delete labels on host %q and location:%q: %s", hostname, location, strings.Join(errs, ", "))
	}

	return nil
}

func (cm *consulManager) UpdateResourcesLabels(location, hostname string, diff map[string]string, operation func(a int64, b int64) int64, update func(orig map[string]string, diff map[string]string, operation func(a int64, b int64) int64) (map[string]string, error)) error {
	return cm.updateResourcesLabelsWait(location, hostname, diff, operation, update, maxWaitTimeSeconds*time.Second)
}

// Labels must be read and write in the same transaction to avoid concurrency issues
func (cm *consulManager) updateResourcesLabelsWait(location, hostname string, diff map[string]string, operation func(a int64, b int64) int64, update func(orig map[string]string, diff map[string]string, operation func(a int64, b int64) int64) (map[string]string, error), maxWaitTime time.Duration) error {
	if location == "" {
		return errors.WithStack(badRequestError{`"location" missing`})
	}
	if hostname == "" {
		return errors.WithStack(badRequestError{`"hostname" missing`})
	}

	lockCh, cleanupFn, err := cm.lockKey(location, hostname, "updateLabels", maxWaitTime)
	if err != nil {
		return err
	}
	defer cleanupFn()

	select {
	case <-lockCh:
		return errors.Errorf("admin lock lost on hosts pool for updating labels with host %q", hostname)
	default:
	}

	labels, err := cm.GetHostLabels(location, hostname)

	upLabels, err := update(labels, diff, operation)
	if err != nil {
		return err
	}

	if upLabels == nil || len(upLabels) == 0 {
		return nil
	}

	log.Debugf("Updating labels:%+v", upLabels)
	ops, err := cm.getAddLabelsOperations(location, hostname, upLabels)
	if err != nil {
		return err
	}

	ok, response, _, err := cm.cc.KV().Txn(ops, nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !ok {
		// Check the response
		errs := make([]string, 0)
		for _, e := range response.Errors {
			errs = append(errs, e.What)
		}
		return errors.Errorf("Failed to add labels to host %q: %s", hostname, strings.Join(errs, ", "))
	}
	return nil
}

func (cm *consulManager) GetHostLabels(location, hostname string) (map[string]string, error) {
	if location == "" {
		return nil, errors.WithStack(badRequestError{`"location" missing`})
	}
	if hostname == "" {
		return nil, errors.WithStack(badRequestError{`"hostname" missing`})
	}
	// check if host exists
	_, err := cm.GetHostStatus(location, hostname)
	if err != nil {
		return nil, err
	}
	// Appending a final "/" here is not necessary as there is no other keys starting with "labels" prefix
	kvps, _, err := cm.cc.KV().List(path.Join(consulutil.HostsPoolPrefix, location, hostname, "labels"), nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	labels := make(map[string]string, len(kvps))
	for _, kvp := range kvps {
		labels[path.Base(kvp.Key)] = string(kvp.Value)
	}
	return labels, nil
}

func (cm *consulManager) getUpdateResourcesLabelsOperationsOnLabelsChange(location, hostname string, newLabels map[string]string) (api.KVTxnOps, error) {
	allocs, err := cm.GetAllocations(location, hostname)
	if err != nil {
		return nil, err
	}

	// Apply allocations resources on new labels
	upLabels := newLabels
	for _, alloc := range allocs {
		upLabels, err = cm.calculateLabels(alloc.Resources, upLabels, subtract, updateResourcesLabels)
		if err != nil {
			return nil, err
		}
	}
	return cm.getAddLabelsOperations(location, hostname, upLabels)
}

func (cm *consulManager) getUpdateResourcesLabelsOperations(location, hostname string, diff map[string]string, new map[string]string, operation func(a int64, b int64) int64, update func(orig map[string]string, diff map[string]string, operation func(a int64, b int64) int64) (map[string]string, error)) (api.KVTxnOps, error) {
	upLabels, err := cm.calculateLabels(diff, new, operation, update)
	if err != nil {
		return nil, err
	}
	if upLabels == nil || len(upLabels) == 0 {
		return nil, nil
	}
	return cm.getAddLabelsOperations(location, hostname, upLabels)
}

func (cm *consulManager) calculateLabels(diff map[string]string, new map[string]string, operation func(a int64, b int64) int64, update func(orig map[string]string, diff map[string]string, operation func(a int64, b int64) int64) (map[string]string, error)) (map[string]string, error) {
	upLabels, err := update(new, diff, operation)
	if err != nil {
		return nil, err
	}

	if upLabels == nil || len(upLabels) == 0 {
		return nil, nil
	}

	return upLabels, nil
}

func (cm *consulManager) getAddLabelsOperations(location, hostname string, labels map[string]string) (api.KVTxnOps, error) {
	hostKVPrefix := path.Join(consulutil.HostsPoolPrefix, location, hostname)
	ops := make(api.KVTxnOps, 0)
	for k, v := range labels {
		k = url.PathEscape(k)
		if k == "" {
			return nil, errors.WithStack(badRequestError{"empty labels are not allowed"})
		}
		ops = append(ops, &api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "labels", k),
			Value: []byte(v),
		})
	}
	return ops, nil
}
