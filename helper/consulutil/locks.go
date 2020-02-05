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

package consulutil

import (
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"strings"
	"time"
)

// AutoDeleteLock is a composition of an consul Lock but its Unlock function also call the Destroy function
type AutoDeleteLock struct {
	*api.Lock
}

// Unlock calls in sequence Unlock from the original lock then destroy
func (adl *AutoDeleteLock) Unlock() error {
	err := adl.Lock.Unlock()
	if err != nil {
		return errors.Wrap(err, ConsulGenericErrMsg)
	}
	return errors.Wrap(adl.Lock.Destroy(), ConsulGenericErrMsg)
}

// AcquireLock returns an AutoDeleteLock on a specified key
// it skips "Missing check 'serfHealth' registration" error
// With specified timeout or default 10s
func AcquireLock(cc *api.Client, key string, timeout time.Duration) (*AutoDeleteLock, error) {
	lock, err := cc.LockKey(key)
	if err != nil {
		return nil, errors.Wrap(err, ConsulGenericErrMsg)
	}

	if timeout == 0 {
		timeout = 10 * time.Second
	}

	var lockCh <-chan struct{}
	timeAfter := time.After(timeout)
	for lockCh == nil {
		lockCh, err = lock.Lock(nil)
		if err != nil {
			if strings.Contains(err.Error(), "Missing check 'serfHealth' registration") {
				// Skip this error as it means that Consul has not correctly started abd register checks
				continue
			} else {
				return nil, err
			}
		}
		// error is not blocking until timeout
		select {
		case <-timeAfter:
			return nil, errors.Errorf("failed to acquire Consul lock for key:%q with timeout:%s due to last error:%+v", key, timeout.String(), err)
		default:
		}
	}
	return &AutoDeleteLock{Lock: lock}, nil

}
