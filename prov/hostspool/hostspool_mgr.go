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
	"fmt"
	"path"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"

	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/helper/labelsutil"
	"github.com/ystia/yorc/v3/helper/sshutil"
)

const (
	// CheckpointError is an error of checkpoint between the current Hosts Pool
	// and an apply change request
	CheckpointError = "Checkpoint for Hosts Pool error"
	// maxWaitTimeSeconds is the max time to wait for a lock on write operations
	maxWaitTimeSeconds = 120
	// maxNbTransactionOps is the maximum number of operations within a transaction
	// supported by Consul (limit hard-coded in Consul implementation)
	maxNbTransactionOps = 64
)

// A Manager is in charge of creating/updating/deleting hosts from the pool
type Manager interface {
	Add(hostname string, connection Connection, labels map[string]string) error
	Apply(pool []Host, checkpoint *uint64) error
	Remove(hostname string) error
	UpdateResourcesLabels(hostname string, diff map[string]string, operation func(a int64, b int64) int64, update func(orig map[string]string, diff map[string]string, operation func(a int64, b int64) int64) (map[string]string, error)) error
	AddLabels(hostname string, labels map[string]string) error
	RemoveLabels(hostname string, labels []string) error
	UpdateConnection(hostname string, connection Connection) error
	List(filters ...labelsutil.Filter) ([]string, []labelsutil.Warning, uint64, error)
	GetHost(hostname string) (Host, error)
	Allocate(allocation *Allocation, filters ...labelsutil.Filter) (string, []labelsutil.Warning, error)
	Release(hostname string, allocation *Allocation) error
}

// SSHClientFactory is a that could be called to customize the client used to check the connection.
//
// Currently this is used for testing purpose to mock the ssh connection.
type SSHClientFactory func(config *ssh.ClientConfig, conn Connection) sshutil.Client

// NewManager creates a Manager backed to Consul
func NewManager(cc *api.Client) Manager {
	return NewManagerWithSSHFactory(cc, func(config *ssh.ClientConfig, conn Connection) sshutil.Client {
		return &sshutil.SSHClient{
			Config: config,
			Host:   conn.Host,
			Port:   int(conn.Port),
		}
	})
}

// NewManagerWithSSHFactory creates a Manager with a given ssh factory
//
// Currently this is used for testing purpose to mock the ssh connection.
func NewManagerWithSSHFactory(cc *api.Client, sshClientFactory SSHClientFactory) Manager {
	return &consulManager{cc: cc, getSSHClient: sshClientFactory}
}

// Lock key is not under HostsPoolPrefix so that taking the lock and releasing
// without any change to the Hosts Pool will not update the last index of the
// Hosts Pool list
const kvLockKey = consulutil.YorcManagementPrefix + "/hosts_pool/lock"

type consulManager struct {
	cc           *api.Client
	getSSHClient SSHClientFactory
}

func (cm *consulManager) Add(hostname string, conn Connection, labels map[string]string) error {
	return cm.addWait(hostname, conn, labels, maxWaitTimeSeconds*time.Second)
}
func (cm *consulManager) addWait(hostname string, conn Connection, labels map[string]string, maxWaitTime time.Duration) error {
	ops, err := cm.getAddOperations(hostname, conn, labels, HostStatusFree, "", nil)
	if err != nil {
		return err
	}
	_, cleanupFn, err := cm.lockKey(hostname, "creation", maxWaitTime)
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
			if e.OpIndex == 0 {
				return errors.WithStack(hostAlreadyExistError{})
			}
			errs = append(errs, e.What)
		}
		return errors.Errorf("Failed to register host %q: %s", hostname, strings.Join(errs, ", "))
	}

	err = cm.checkConnection(hostname)
	if err != nil {
		cm.setHostStatusWithMessage(hostname, HostStatusError, "can't connect to host")
	}
	return err
}

func (cm *consulManager) getAddOperations(
	hostname string,
	conn Connection,
	labels map[string]string,
	status HostStatus,
	message string,
	allocations []Allocation) (api.KVTxnOps, error) {

	if hostname == "" {
		return nil, errors.WithStack(badRequestError{`"hostname" missing`})
	}

	if conn.Password == "" && conn.PrivateKey == "" {
		return nil, errors.WithStack(badRequestError{`at least "password" or "private_key" is required for a host pool connection`})
	}

	user := conn.User
	if user == "" {
		user = "root"
	}
	port := conn.Port
	if port == 0 {
		port = 22
	}
	host := conn.Host
	if host == "" {
		host = hostname
	}

	hostKVPrefix := path.Join(consulutil.HostsPoolPrefix, hostname)
	addOps := api.KVTxnOps{
		&api.KVTxnOp{
			Verb: api.KVCheckNotExists,
			Key:  path.Join(hostKVPrefix, "status"),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "status"),
			Value: []byte(status.String()),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "connection", "host"),
			Value: []byte(host),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "connection", "user"),
			Value: []byte(user),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "connection", "password"),
			Value: []byte(conn.Password),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "connection", "private_key"),
			Value: []byte(conn.PrivateKey),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "connection", "port"),
			Value: []byte(strconv.FormatUint(port, 10)),
		},
	}

	if message != "" {

		addOps = append(addOps, &api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "message"),
			Value: []byte(message),
		})
	}

	var allocsOps api.KVTxnOps
	var err error
	if allocsOps, err = getAddAllocationsOperation(hostname, allocations); err != nil {
		return nil, err
	} else if len(allocsOps) > 0 {
		addOps = append(addOps, allocsOps...)
	}

	labelOps, err := cm.getAddUpdatedLabelsOperations(hostname, labels)
	if err != nil {
		return nil, err
	}
	addOps = append(addOps, labelOps...)
	return addOps, nil
}

func (cm *consulManager) Remove(hostname string) error {
	return cm.removeWait(hostname, maxWaitTimeSeconds*time.Second)
}
func (cm *consulManager) removeWait(hostname string, maxWaitTime time.Duration) error {

	ops, err := cm.getRemoveOperations(hostname, true)
	if err != nil {
		return err
	}

	lockCh, cleanupFn, err := cm.lockKey(hostname, "deletion", maxWaitTime)
	if err != nil {
		return err
	}
	defer cleanupFn()

	select {
	case <-lockCh:
		return errors.Errorf("admin lock lost on hosts pool for host %q deletion", hostname)
	default:
	}

	ok, response, _, err := cm.cc.KV().Txn(ops, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to delete host %q", hostname)
	}
	if !ok {
		// Check the response
		errs := make([]string, 0)
		for _, e := range response.Errors {
			errs = append(errs, e.What)
		}
		return errors.Errorf("Failed to delete host %q: %s", hostname, strings.Join(errs, ", "))
	}

	return nil
}

func (cm *consulManager) getRemoveOperations(hostname string, checkStatus bool) (api.KVTxnOps, error) {
	if hostname == "" {
		return nil, errors.WithStack(badRequestError{`"hostname" missing`})
	}

	hostKey := path.Join(consulutil.HostsPoolPrefix, hostname)
	// Need to remove the host key subtree, but not the tree of hosts having a
	// hostname containing as a prefix the name of the host to delete
	hostKeyTreePrexix := hostKey + "/"

	if checkStatus {
		status, err := cm.GetHostStatus(hostname)
		if err != nil {
			return nil, err
		}
		switch status {
		case HostStatusFree, HostStatusError:
			// Ok go ahead
		default:
			return nil, errors.WithStack(badRequestError{fmt.Sprintf("can't delete host %q with status %q", hostname, status.String())})
		}
	}

	rmOps := api.KVTxnOps{
		&api.KVTxnOp{
			Verb: api.KVDeleteTree,
			Key:  hostKeyTreePrexix,
		},
		&api.KVTxnOp{
			Verb: api.KVDelete,
			Key:  hostKey,
		},
	}

	return rmOps, nil
}

func (cm *consulManager) lockKey(hostname, opType string, lockWaitTime time.Duration) (lockCh <-chan struct{}, cleanupFn func(), err error) {
	var sessionName string
	if hostname != "" {
		sessionName = fmt.Sprintf("%q %s", hostname, opType)
	} else {
		sessionName = opType
	}
	lock, err := cm.cc.LockOpts(&api.LockOptions{
		Key:            kvLockKey,
		Value:          []byte(fmt.Sprintf("locked for %s", sessionName)),
		MonitorRetries: 2,
		LockWaitTime:   lockWaitTime,
		// Not setting LockTryOnce to true to workaround this Consul issue:
		// https://github.com/hashicorp/consul/issues/4003
		// LockTryOnce: true,
		SessionName: sessionName,
		SessionTTL:  lockWaitTime.String(),
		SessionOpts: &api.SessionEntry{
			Behavior: api.SessionBehaviorDelete,
		},
	})
	if err != nil {
		err = errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		return
	}

	// To workaround Consul issue https://github.com/hashicorp/consul/issues/4003
	// LockTryOnce is false (default value) which means lock.Lock() will be
	// blocking.
	// Now to avoid being blocked forever attempting to get the lock, arming a
	// timer and closing a stopChannel if this timer expires to go out of the
	// call to lock.Lock(stopChannel) below
	stopChannel := make(chan struct{})
	timerWaitLock := time.NewTimer(lockWaitTime)
	go func() {
		<-timerWaitLock.C
		// Timer expired, closing stop channel to stop the blocking lock below
		if lockCh == nil {
			close(stopChannel)
		}
	}()
	lockCh, err = lock.Lock(stopChannel)
	timerWaitLock.Stop()

	if err != nil {
		err = errors.Wrapf(err, "failed to acquire admin lock on hosts pool for %s", sessionName)
		return
	}
	if lockCh == nil {
		err = errors.Errorf("failed to acquire admin lock on Hosts Pool for %s", sessionName)
		return
	}

	select {
	case <-lockCh:
		err = errors.Errorf("admin lock lost on hosts pool for %s", sessionName)
		return
	default:
	}

	cleanupFn = func() {
		lock.Unlock()
		lock.Destroy()
	}
	return
}

func (cm *consulManager) List(filters ...labelsutil.Filter) ([]string, []labelsutil.Warning, uint64, error) {
	hosts, metadata, err := cm.cc.KV().Keys(consulutil.HostsPoolPrefix+"/", "/", nil)
	if err != nil {
		return nil, nil, 0, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	warnings := make([]labelsutil.Warning, 0)
	results := hosts[:0]
	for _, host := range hosts {
		host = path.Base(host)
		labels, err := cm.GetHostLabels(host)
		if err != nil {
			return nil, nil, 0, err
		}
		ok, warn := labelsutil.MatchesAll(labels, filters...)
		if warn != nil {
			warnings = append(warnings, errors.Wrapf(warn, "host: %q", host))
		} else if ok {
			results = append(results, host)
		}
	}
	return results, warnings, metadata.LastIndex, nil
}

func (cm *consulManager) backupHostStatus(hostname string) error {
	status, err := cm.GetHostStatus(hostname)
	if err != nil {
		return err
	}
	message, err := cm.GetHostMessage(hostname)
	if err != nil {
		return err
	}
	hostPath := path.Join(consulutil.HostsPoolPrefix, hostname)
	_, errGrp, store := consulutil.WithContext(context.Background())
	store.StoreConsulKeyAsString(path.Join(hostPath, ".statusBackup"), status.String())
	store.StoreConsulKeyAsString(path.Join(hostPath, ".messageBackup"), message)
	return errGrp.Wait()
}
func (cm *consulManager) restoreHostStatus(hostname string) error {
	hostPath := path.Join(consulutil.HostsPoolPrefix, hostname)
	kvp, _, err := cm.cc.KV().Get(path.Join(hostPath, ".statusBackup"), nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return errors.Errorf("missing backup status for host %q", hostname)
	}
	status, err := ParseHostStatus(string(kvp.Value))
	if err != nil {
		return errors.Wrapf(err, "invalid backup status for host %q", hostname)
	}
	err = cm.setHostStatus(hostname, status)
	if err != nil {
		return err
	}
	_, err = cm.cc.KV().Delete(path.Join(hostPath, ".statusBackup"), nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	kvp, _, err = cm.cc.KV().Get(path.Join(hostPath, ".messageBackup"), nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	var msg string
	if kvp != nil {
		msg = string(kvp.Value)
	}
	err = cm.setHostMessage(hostname, msg)
	if err != nil {
		return err
	}
	_, err = cm.cc.KV().Delete(path.Join(hostPath, ".messageBackup"), nil)
	return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
}

func (cm *consulManager) setHostStatus(hostname string, status HostStatus) error {
	return cm.setHostStatusWithMessage(hostname, status, "")
}

func (cm *consulManager) setHostStatusWithMessage(hostname string, status HostStatus, message string) error {
	_, err := cm.GetHostStatus(hostname)
	if err != nil {
		return err
	}
	err = consulutil.StoreConsulKeyAsString(path.Join(consulutil.HostsPoolPrefix, hostname, "status"), status.String())
	if err != nil {
		return err
	}
	return cm.setHostMessage(hostname, message)
}

func (cm *consulManager) GetHostStatus(hostname string) (HostStatus, error) {
	return cm.getStatus(hostname, false)
}

func (cm *consulManager) getStatus(hostname string, backup bool) (HostStatus, error) {
	if hostname == "" {
		return HostStatus(0), errors.WithStack(badRequestError{`"hostname" missing`})
	}
	keyname := "status"
	if backup {
		keyname = ".statusBackup"
	}

	kvp, _, err := cm.cc.KV().Get(path.Join(consulutil.HostsPoolPrefix, hostname, keyname), nil)
	if err != nil {
		return HostStatus(0), errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return HostStatus(0), errors.WithStack(hostNotFoundError{})
	}
	status, err := ParseHostStatus(string(kvp.Value))
	if err != nil {
		return HostStatus(0), errors.Wrapf(err, "failed to retrieve %s for host %q", keyname, hostname)
	}

	return status, nil
}

func (cm *consulManager) GetHostMessage(hostname string) (string, error) {
	return cm.getMessage(hostname, false)
}

func (cm *consulManager) getMessage(hostname string, backup bool) (string, error) {
	if hostname == "" {
		return "", errors.WithStack(badRequestError{`"hostname" missing`})
	}

	// check if host exists
	_, err := cm.GetHostStatus(hostname)
	if err != nil {
		return "", err
	}

	keyname := "message"
	if backup {
		keyname = ".messageBackup"
	}

	kvp, _, err := cm.cc.KV().Get(path.Join(consulutil.HostsPoolPrefix, hostname, keyname), nil)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return "", nil
	}
	return string(kvp.Value), nil
}

func (cm *consulManager) setHostMessage(hostname, message string) error {
	if hostname == "" {
		return errors.WithStack(badRequestError{`"hostname" missing`})
	}
	// check if host exists
	_, err := cm.GetHostStatus(hostname)
	if err != nil {
		return err
	}
	return consulutil.StoreConsulKeyAsString(path.Join(consulutil.HostsPoolPrefix, hostname, "message"), message)
}

func (cm *consulManager) GetHost(hostname string) (Host, error) {
	host := Host{Name: hostname}
	if hostname == "" {
		return host, errors.WithStack(badRequestError{`"hostname" missing`})
	}
	var err error
	host.Status, err = cm.GetHostStatus(hostname)
	if err != nil {
		return host, err
	}
	host.Message, err = cm.GetHostMessage(hostname)
	if err != nil {
		return host, err
	}

	host.Connection, err = cm.GetHostConnection(hostname)
	if err != nil {
		return host, err
	}
	host.Allocations, err = cm.GetAllocations(hostname)
	if err != nil {
		return host, err
	}

	host.Labels, err = cm.GetHostLabels(hostname)
	return host, err
}

func getSSHConfig(conn Connection) (*ssh.ClientConfig, error) {
	conf := &ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		User:            conn.User,
	}

	if conn.PrivateKey != "" {
		keyAuth, err := sshutil.ReadPrivateKey(conn.PrivateKey)
		if err != nil {
			return nil, err
		}
		conf.Auth = append(conf.Auth, keyAuth)
	}

	if conn.Password != "" {
		conf.Auth = append(conf.Auth, ssh.Password(conn.Password))
	}
	return conf, nil
}

// Apply a Hosts Pool configuration.
// If checkpoint is not nil, it should point to a value returned by a previous
// call to the List() function described above. A checkpoint verification will
// be done to ensure that the Hosts Pool was not changed between the call to
// List() and the current call to Apply(). Once the Hosts Pool configuration
// has been applied, checkpoint will point to the new Hosts Pool checkpoint
// value.
// If checkpoint is nil, the Hosts Pool configuration will be applied without
// checkpoint verification.
func (cm *consulManager) Apply(pool []Host, checkpoint *uint64) error {
	return cm.applyWait(pool, checkpoint, maxWaitTimeSeconds*time.Second)
}

func (cm *consulManager) applyWait(
	pool []Host,
	checkpoint *uint64,
	maxWaitTime time.Duration) error {

	// First, checking the pool definition to verify there is no host with an
	// empty name or a duplicate name, or wrong connection definition, and
	// provide an error message referencing indexes in the definition to help
	// the user identify which definition is erroneous
	hostIndexDefinition := make(map[string]int)
	for i, host := range pool {
		if host.Name == "" {
			return errors.WithStack(badRequestError{
				fmt.Sprintf("A non-empty Name should be provided for Host number %d, defined with connection %q",
					i+1, host.Connection)})
		}

		// Check if the name has already been used. It must me unique in the Hosts Pool
		if index, ok := hostIndexDefinition[host.Name]; ok {
			return errors.WithStack(badRequestError{
				fmt.Sprintf("Name value %q must be unique but is re-used in host number %d when first used in host number %d",
					host.Name, i+1, index+1)})
		}
		hostIndexDefinition[host.Name] = i
	}

	// Take the lock to have a consistent view while computing needed
	// configuration changes
	lockCh, cleanupFn, err := cm.lockKey("", "apply", maxWaitTime)
	if err != nil {
		return err
	}
	defer cleanupFn()

	// Get all hosts currently registered to find which ones will have to be
	// unregistered or updated.
	// Attempting to unregister a host that is still allocated is illegal
	registeredHosts, _, runtimeCheckpoint, err := cm.List()
	if err != nil {
		return errors.Wrapf(err, "Failed to get list of registered hosts")
	}

	// Verify checkpoint, no change done if the checkpoint in argument is
	// lower than the current checkpoint, as it means that another Hosts Pool
	// change happened since
	if checkpoint != nil &&
		((*checkpoint == 0 && len(registeredHosts) > 0) ||
			(*checkpoint > 0 && *checkpoint < runtimeCheckpoint)) {
		return errors.WithStack(badRequestError{
			fmt.Sprintf("%s: value provided %d lower than expected checkpoint %d",
				CheckpointError, *checkpoint, runtimeCheckpoint)})
	}

	hostsToUnregisterCheckAllocatedStatus := make(map[string]bool)
	for _, registeredHost := range registeredHosts {
		hostsToUnregisterCheckAllocatedStatus[registeredHost] = true
	}

	// Compare  new hosts pool definition to the runtime to compute changes
	var hostChanged []string
	var addOps api.KVTxnOps
	for _, host := range pool {

		found := hostsToUnregisterCheckAllocatedStatus[host.Name]
		if found {

			// Host already in pool, check if an update is needed
			oldHost, _ := cm.GetHost(host.Name)
			if oldHost.Connection == host.Connection &&
				reflect.DeepEqual(oldHost.Labels, host.Labels) {

				// No config change, no update needed, ignoring this host
				delete(hostsToUnregisterCheckAllocatedStatus, host.Name)
				continue
			}

			// A config change is request for this already known host.
			hostChanged = append(hostChanged, host.Name)

			// This host will be unregistered then registered again.
			// No need to check the status of this host at unregistration time,
			// it will be recreated with the same status
			hostsToUnregisterCheckAllocatedStatus[host.Name] = false

			status, err := cm.GetHostStatus(host.Name)
			if err != nil {
				return err
			}
			message, err := cm.GetHostMessage(host.Name)
			if err != nil {
				return err
			}

			allocations, err := cm.GetAllocations(host.Name)
			if err != nil {
				return err
			}

			// Backup status and message if defined are restored at re-creation,
			// the connection check will be performed afterwards
			if status == HostStatusError {
				backupStatus, err := cm.getStatus(host.Name, true)
				if err == nil {
					status = backupStatus
					message, _ = cm.getMessage(host.Name, true)
				}
			}
			ops, err := cm.getAddOperations(host.Name, host.Connection, host.Labels,
				status, message, allocations)
			if err != nil {
				return err
			}
			addOps = append(addOps, ops...)
		} else {
			// Host is new, creating it
			hostChanged = append(hostChanged, host.Name)
			ops, err := cm.getAddOperations(host.Name, host.Connection, host.Labels,
				HostStatusFree, "", nil)
			if err != nil {
				return err
			}
			addOps = append(addOps, ops...)
		}
	}

	// Now manage hosts to delete
	var ops api.KVTxnOps
	for host, checkStatus := range hostsToUnregisterCheckAllocatedStatus {
		removeOps, err := cm.getRemoveOperations(host, checkStatus)
		if err != nil {
			return err
		}
		ops = append(ops, removeOps...)
	}

	ops = append(ops, addOps...)

	// Execute operations in a transaction

	select {
	case <-lockCh:
		return errors.Errorf("admin lock lost on hosts pool for apply operation")
	default:
	}

	// Need to split the transaction if there are more than the max number of
	// operations in a transaction supported by Consul
	opsLength := len(ops)
	for begin := 0; begin < opsLength; begin += maxNbTransactionOps {
		end := begin + maxNbTransactionOps
		if end > opsLength {
			end = opsLength
		}

		ok, response, _, err := cm.cc.KV().Txn(ops[begin:end], nil)
		if err != nil {
			return errors.Wrap(err, "Failed to apply new Hosts Pool configuration")
		}

		if !ok {
			// Check the response
			var errs []string
			for _, e := range response.Errors {
				errs = append(errs, e.What)
			}
			err = errors.Errorf("Failed to apply new Hosts Pool configuration: %s", strings.Join(errs, ", "))
		}
	}

	// Update the connection status for each updated/created host
	var waitGroup sync.WaitGroup
	for _, name := range hostChanged {
		waitGroup.Add(1)
		go cm.updateConnectionStatus(name, &waitGroup)
	}
	waitGroup.Wait()

	// Updating the checkpoint value
	// Not using querymeta.LastIndex from KV().Txn() as it doesn't work the same
	// way as in KV().Keys used in cm.List().
	if checkpoint != nil {
		_, _, newCheckpoint, errCkpt := cm.List()
		if errCkpt != nil {
			// If the apply didn't fail, return this error, else the apply error
			// takes precedence
			if err == nil {
				err = errors.Wrapf(errCkpt, "Failed to get list of registered hosts")
			}
		} else {
			*checkpoint = newCheckpoint
		}
	}

	return err
}
