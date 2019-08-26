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
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/helper/consulutil"
)

func (cm *consulManager) UpdateConnection(hostname string, conn Connection) error {
	return cm.updateConnectionWait(hostname, conn, maxWaitTimeSeconds*time.Second)
}
func (cm *consulManager) updateConnectionWait(hostname string, conn Connection, maxWaitTime time.Duration) error {
	if hostname == "" {
		return errors.WithStack(badRequestError{`"hostname" missing`})
	}

	// check if host exists
	status, err := cm.GetHostStatus(hostname)
	if err != nil {
		return err
	}

	ops := make(api.KVTxnOps, 0)
	hostKVPrefix := path.Join(consulutil.HostsPoolPrefix, hostname)
	if conn.User != "" {
		ops = append(ops, &api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "connection", "user"),
			Value: []byte(conn.User),
		})
	}
	if conn.Port != 0 {
		ops = append(ops, &api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "connection", "port"),
			Value: []byte(strconv.FormatUint(conn.Port, 10)),
		})
	}
	if conn.Host != "" {
		ops = append(ops, &api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "connection", "host"),
			Value: []byte(conn.Host),
		})
	}
	if conn.PrivateKey != "" {
		if conn.PrivateKey == "-" {
			ok, err := cm.DoesHostHasConnectionPassword(hostname)
			if err != nil {
				return err
			}
			if !ok && conn.Password == "" || ok && conn.Password == "-" {
				return errors.WithStack(badRequestError{`at any time at least one of "password" or "private_key" is required`})
			}
			conn.PrivateKey = ""
		}
		ops = append(ops, &api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "connection", "private_key"),
			Value: []byte(conn.PrivateKey),
		})
	}
	if conn.Password != "" {
		if conn.Password == "-" {
			ok, err := cm.DoesHostHasConnectionPrivateKey(hostname)
			if err != nil {
				return err
			}
			if !ok && conn.PrivateKey == "" || ok && conn.PrivateKey == "-" {
				return errors.WithStack(badRequestError{`at any time at least one of "password" or "private_key" is required`})
			}
			conn.Password = ""
		}
		ops = append(ops, &api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "connection", "password"),
			Value: []byte(conn.Password),
		})
	}

	_, cleanupFn, err := cm.lockKey(hostname, "update", maxWaitTime)
	if err != nil {
		return err
	}
	defer cleanupFn()

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
		return errors.Errorf("Failed to update host %q connection: %s", hostname, strings.Join(errs, ", "))
	}

	err = cm.checkConnection(hostname)
	if err != nil {
		if status != HostStatusError {
			cm.backupHostStatus(hostname)
			cm.setHostStatusWithMessage(hostname, HostStatusError, "failed to connect to host")
		}
		return err
	}
	if status == HostStatusError {
		cm.restoreHostStatus(hostname)
	}
	return nil
}

func (cm *consulManager) DoesHostHasConnectionPrivateKey(hostname string) (bool, error) {
	c, err := cm.GetHostConnection(hostname)
	if err != nil {
		return false, err
	}
	return c.PrivateKey != "", nil
}

func (cm *consulManager) DoesHostHasConnectionPassword(hostname string) (bool, error) {
	c, err := cm.GetHostConnection(hostname)
	if err != nil {
		return false, err
	}
	return c.Password != "", nil
}

func (cm *consulManager) GetHostConnection(hostname string) (Connection, error) {
	conn := Connection{}
	if hostname == "" {
		return conn, errors.WithStack(badRequestError{`"hostname" missing`})
	}
	kv := cm.cc.KV()
	connKVPrefix := path.Join(consulutil.HostsPoolPrefix, hostname, "connection")

	kvp, _, err := kv.Get(path.Join(connKVPrefix, "host"), nil)
	if err != nil {
		return conn, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil {
		conn.Host = string(kvp.Value)
		conn.Host = config.DefaultConfigTemplateResolver.ResolveValueWithTemplates("Connection.Host", conn.Host).(string)
	}
	kvp, _, err = kv.Get(path.Join(connKVPrefix, "user"), nil)
	if err != nil {
		return conn, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil {
		conn.User = string(kvp.Value)
		conn.User = config.DefaultConfigTemplateResolver.ResolveValueWithTemplates("Connection.User", conn.User).(string)
	}
	kvp, _, err = kv.Get(path.Join(connKVPrefix, "password"), nil)
	if err != nil {
		return conn, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil {
		conn.Password = string(kvp.Value)
		conn.Password = config.DefaultConfigTemplateResolver.ResolveValueWithTemplates("Connection.Password", conn.Password).(string)
	}
	kvp, _, err = kv.Get(path.Join(connKVPrefix, "private_key"), nil)
	if err != nil {
		return conn, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil {
		conn.PrivateKey = string(kvp.Value)
		conn.PrivateKey = config.DefaultConfigTemplateResolver.ResolveValueWithTemplates("Connection.PrivateKey", conn.PrivateKey).(string)
	}
	kvp, _, err = kv.Get(path.Join(connKVPrefix, "port"), nil)
	if err != nil {
		return conn, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil {
		conn.Port, err = strconv.ParseUint(string(kvp.Value), 10, 64)
		if err != nil {
			return conn, errors.Wrapf(err, "failed to retrieve connection port for host %q", hostname)
		}
	}

	return conn, nil
}

// Check if we can log into an host given a connection
func (cm *consulManager) checkConnection(hostname string) error {

	conn, err := cm.GetHostConnection(hostname)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to host %q", hostname)
	}
	conf, err := getSSHConfig(conn)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to host %q", hostname)
	}

	client := cm.getSSHClient(conf, conn)
	_, err = client.RunCommand(`echo "Connected!"`)
	return errors.Wrapf(err, "failed to connect to host %q", hostname)
}

// Go routine checking a Host connection and updating the Host status
func (cm *consulManager) updateConnectionStatus(name string, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	status, err := cm.GetHostStatus(name)
	if err != nil {
		// No such host anymore
		return
	}

	err = cm.checkConnection(name)
	if err != nil {
		if status != HostStatusError {
			cm.backupHostStatus(name)
			cm.setHostStatusWithMessage(name, HostStatusError, "failed to connect to host")
		}
		return
	}
	// Connection is up now. If it was previously down, restoring the status as
	// it was before the failure (free, allocated)
	if status == HostStatusError {
		cm.restoreHostStatus(name)
	}
}
