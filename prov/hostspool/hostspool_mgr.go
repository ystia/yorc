package hostspool

import (
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"

	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/helper/labelsutil"
	"novaforge.bull.com/starlings-janus/janus/helper/sshutil"
)

// A Manager is in charge of creating/updating/deleting hosts from the pool
type Manager interface {
	Add(hostname string, connection Connection, labels map[string]string) error
	Remove(hostname string) error
	AddLabels(hostname string, labels map[string]string) error
	RemoveLabels(hostname string, labels []string) error
	UpdateConnection(hostname string, connection Connection) error
	List(filters ...labelsutil.Filter) ([]string, []labelsutil.Warning, error)
	GetHost(hostname string) (Host, error)
	Allocate(message string, filters ...labelsutil.Filter) (string, []labelsutil.Warning, error)
	Release(hostname string) error
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

const kvLockKey = consulutil.HostsPoolPrefix + "/.mgrLock"

type consulManager struct {
	cc           *api.Client
	getSSHClient SSHClientFactory
}

func (cm *consulManager) Add(hostname string, conn Connection, labels map[string]string) error {
	return cm.addWait(hostname, conn, labels, 45*time.Second)
}
func (cm *consulManager) addWait(hostname string, conn Connection, labels map[string]string, maxWaitTime time.Duration) error {
	if hostname == "" {
		return errors.WithStack(badRequestError{`"hostname" missing`})
	}

	if conn.Password == "" && conn.PrivateKey == "" {
		return errors.WithStack(badRequestError{`at least "password" or "private_key" is required for a host pool connection`})
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
	ops := api.KVTxnOps{
		&api.KVTxnOp{
			Verb: api.KVCheckNotExists,
			Key:  path.Join(hostKVPrefix, "status"),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "status"),
			Value: []byte(HostStatusFree.String()),
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

	for k, v := range labels {
		k = url.PathEscape(k)
		if k == "" {
			return errors.WithStack(badRequestError{"empty labels are not allowed"})
		}
		ops = append(ops, &api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "labels", k),
			Value: []byte(v),
		})
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

func (cm *consulManager) UpdateConnection(hostname string, conn Connection) error {
	return cm.updateConnWait(hostname, conn, 45*time.Second)
}
func (cm *consulManager) updateConnWait(hostname string, conn Connection, maxWaitTime time.Duration) error {
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

func (cm *consulManager) Remove(hostname string) error {
	return cm.removeWait(hostname, 45*time.Second)
}
func (cm *consulManager) removeWait(hostname string, maxWaitTime time.Duration) error {
	if hostname == "" {
		return errors.WithStack(badRequestError{`"hostname" missing`})
	}

	lockCh, cleanupFn, err := cm.lockKey(hostname, "deletion", maxWaitTime)
	if err != nil {
		return err
	}
	defer cleanupFn()

	hostKVPrefix := path.Join(consulutil.HostsPoolPrefix, hostname)

	kv := cm.cc.KV()

	status, err := cm.GetHostStatus(hostname)
	if err != nil {
		return err
	}
	switch status {
	case HostStatusFree, HostStatusError:
		// Ok go ahead
	default:
		return errors.WithStack(badRequestError{fmt.Sprintf("can't delete host %q with status %q", hostname, status.String())})
	}

	select {
	case <-lockCh:
		return errors.Errorf("admin lock lost on hosts pool for host %q deletion", hostname)
	default:
	}

	_, err = kv.DeleteTree(hostKVPrefix, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to delete host %q", hostname)
	}

	return nil
}

func (cm *consulManager) AddLabels(hostname string, labels map[string]string) error {
	return cm.addLabelsWait(hostname, labels, 45*time.Second)
}
func (cm *consulManager) addLabelsWait(hostname string, labels map[string]string, maxWaitTime time.Duration) error {
	if hostname == "" {
		return errors.WithStack(badRequestError{`"hostname" missing`})
	}
	if labels == nil || len(labels) == 0 {
		return nil
	}

	hostKVPrefix := path.Join(consulutil.HostsPoolPrefix, hostname)
	ops := make(api.KVTxnOps, 0)

	for k, v := range labels {
		k = url.PathEscape(k)
		if k == "" {
			return errors.WithStack(badRequestError{"empty labels are not allowed"})
		}
		ops = append(ops, &api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "labels", k),
			Value: []byte(v),
		})
	}

	_, cleanupFn, err := cm.lockKey(hostname, "labels addition", maxWaitTime)
	if err != nil {
		return err
	}
	defer cleanupFn()

	// Checks host existence
	_, err = cm.GetHostStatus(hostname)
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
		return errors.Errorf("Failed to add labels to host %q: %s", hostname, strings.Join(errs, ", "))
	}

	return nil
}

func (cm *consulManager) RemoveLabels(hostname string, labels []string) error {
	return cm.removeLabelsWait(hostname, labels, 45*time.Second)
}
func (cm *consulManager) removeLabelsWait(hostname string, labels []string, maxWaitTime time.Duration) error {
	if hostname == "" {
		return errors.WithStack(badRequestError{`"hostname" missing`})
	}
	if labels == nil || len(labels) == 0 {
		return nil
	}

	hostKVPrefix := path.Join(consulutil.HostsPoolPrefix, hostname)
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

	_, cleanupFn, err := cm.lockKey(hostname, "labels remove", maxWaitTime)
	if err != nil {
		return err
	}
	defer cleanupFn()

	// Checks host existence
	_, err = cm.GetHostStatus(hostname)
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
		return errors.Errorf("Failed to delete labels on host %q: %s", hostname, strings.Join(errs, ", "))
	}

	return nil
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
		LockTryOnce:    true,
		SessionName:    sessionName,
		SessionTTL:     lockWaitTime.String(),
		SessionOpts: &api.SessionEntry{
			Behavior: api.SessionBehaviorDelete,
		},
	})
	if err != nil {
		err = errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		return
	}

	lockCh, err = lock.Lock(nil)
	if err != nil {
		err = errors.Wrapf(err, "failed to acquire admin lock on hosts pool for host %q deletion", hostname)
		return
	}
	if lockCh == nil {
		err = errors.Errorf("failed to acquire admin lock on hosts pool for host %q deletion", hostname)
		return
	}

	select {
	case <-lockCh:
		err = errors.Errorf("admin lock lost on hosts pool for host %q deletion", hostname)
		return
	default:
	}

	cleanupFn = func() {
		lock.Unlock()
		lock.Destroy()
	}
	return
}

func (cm *consulManager) List(filters ...labelsutil.Filter) ([]string, []labelsutil.Warning, error) {
	hosts, _, err := cm.cc.KV().Keys(consulutil.HostsPoolPrefix+"/", "/", nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	warnings := make([]labelsutil.Warning, 0)
	results := hosts[:0]
	for _, host := range hosts {
		if host == kvLockKey {
			continue
		}
		host = path.Base(host)
		labels, err := cm.GetHostLabels(host)
		if err != nil {
			return nil, nil, err
		}
		ok, warn := labelsutil.MatchesAll(labels, filters...)
		if warn != nil {
			warnings = append(warnings, errors.Wrapf(warn, "host: %q", host))
		} else if ok {
			results = append(results, host)
		}
	}
	return results, warnings, nil
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
	_, err = cm.cc.KV().Put(&api.KVPair{Key: path.Join(hostPath, ".statusBackup"), Value: []byte(status.String())}, nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	_, err = cm.cc.KV().Put(&api.KVPair{Key: path.Join(hostPath, ".messageBackup"), Value: []byte(message)}, nil)
	return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
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
	_, err = cm.cc.KV().Put(&api.KVPair{Key: path.Join(consulutil.HostsPoolPrefix, hostname, "status"), Value: []byte(status.String())}, nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return cm.setHostMessage(hostname, message)
}

func (cm *consulManager) GetHostStatus(hostname string) (HostStatus, error) {
	if hostname == "" {
		return HostStatus(0), errors.WithStack(badRequestError{`"hostname" missing`})
	}
	kvp, _, err := cm.cc.KV().Get(path.Join(consulutil.HostsPoolPrefix, hostname, "status"), nil)
	if err != nil {
		return HostStatus(0), errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return HostStatus(0), errors.WithStack(hostNotFoundError{})
	}
	status, err := ParseHostStatus(string(kvp.Value))
	if err != nil {
		return HostStatus(0), errors.Wrapf(err, "failed to retrieve status for host %q", hostname)
	}
	return status, nil
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
	}
	kvp, _, err = kv.Get(path.Join(connKVPrefix, "user"), nil)
	if err != nil {
		return conn, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil {
		conn.User = string(kvp.Value)
	}
	kvp, _, err = kv.Get(path.Join(connKVPrefix, "password"), nil)
	if err != nil {
		return conn, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil {
		conn.Password = string(kvp.Value)
	}
	kvp, _, err = kv.Get(path.Join(connKVPrefix, "private_key"), nil)
	if err != nil {
		return conn, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil {
		conn.PrivateKey = string(kvp.Value)
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

func (cm *consulManager) GetHostMessage(hostname string) (string, error) {
	if hostname == "" {
		return "", errors.WithStack(badRequestError{`"hostname" missing`})
	}
	// check if host exists
	_, err := cm.GetHostStatus(hostname)
	if err != nil {
		return "", err
	}
	kvp, _, err := cm.cc.KV().Get(path.Join(consulutil.HostsPoolPrefix, hostname, "message"), nil)
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

func (cm *consulManager) GetHostLabels(hostname string) (map[string]string, error) {
	if hostname == "" {
		return nil, errors.WithStack(badRequestError{`"hostname" missing`})
	}
	// check if host exists
	_, err := cm.GetHostStatus(hostname)
	if err != nil {
		return nil, err
	}
	kvps, _, err := cm.cc.KV().List(path.Join(consulutil.HostsPoolPrefix, hostname, "labels"), nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	labels := make(map[string]string, len(kvps))
	for _, kvp := range kvps {
		labels[path.Base(kvp.Key)] = string(kvp.Value)
	}
	return labels, nil
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

	host.Labels, err = cm.GetHostLabels(hostname)
	return host, err
}

func (cm *consulManager) Allocate(message string, filters ...labelsutil.Filter) (string, []labelsutil.Warning, error) {
	return cm.allocateWait(45*time.Second, message, filters...)
}
func (cm *consulManager) allocateWait(maxWaitTime time.Duration, message string, filters ...labelsutil.Filter) (string, []labelsutil.Warning, error) {
	lockCh, cleanupFn, err := cm.lockKey("", "allocation", maxWaitTime)
	if err != nil {
		return "", nil, err
	}
	defer cleanupFn()

	hosts, warnings, err := cm.List(filters...)
	if err != nil {
		return "", warnings, err
	}
	// Filters only free and connectable hosts but try to bypass errors if we can allocate an host
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
		} else if hs == HostStatusFree {
			freeHosts = append(freeHosts, h)
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
	return hostname, warnings, cm.setHostStatusWithMessage(hostname, HostStatusAllocated, message)
}
func (cm *consulManager) Release(hostname string) error {
	return cm.releaseWait(hostname, 45*time.Second)
}
func (cm *consulManager) releaseWait(hostname string, maxWaitTime time.Duration) error {
	_, cleanupFn, err := cm.lockKey(hostname, "release", maxWaitTime)
	if err != nil {
		return err
	}
	defer cleanupFn()

	status, err := cm.GetHostStatus(hostname)
	if err != nil {
		return err
	}
	if status != HostStatusAllocated {
		return errors.WithStack(badRequestError{fmt.Sprintf("unexpected status %q when releasing host %q", status.String(), hostname)})
	}
	err = cm.setHostMessage(hostname, "")
	if err != nil {
		return err
	}
	return cm.setHostStatus(hostname, HostStatusFree)

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

func getSSHConfig(conn Connection) (*ssh.ClientConfig, error) {
	conf := &ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		User:            conn.User,
	}

	if conn.PrivateKey != "" {
		keyAuth, err := readPrivateKey(conn.PrivateKey)
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

func readPrivateKey(pk string) (ssh.AuthMethod, error) {
	var p []byte
	// check if pk is a path
	keyPath, err := homedir.Expand(pk)
	if err != nil {
		return nil, errors.Wrap(err, "failed to expand key path")
	}
	if _, err := os.Stat(keyPath); err == nil {
		p, err = ioutil.ReadFile(keyPath)
		if err != nil {
			p = []byte(pk)
		}
	} else {
		p = []byte(pk)
	}

	// We parse the private key on our own first so that we can
	// show a nicer error if the private key has a password.
	block, _ := pem.Decode(p)
	if block == nil {
		return nil, errors.Errorf("Failed to read key %q: no key found", pk)
	}
	if block.Headers["Proc-Type"] == "4,ENCRYPTED" {
		return nil, errors.Errorf(
			"Failed to read key %q: password protected keys are\n"+
				"not supported. Please decrypt the key prior to use.", pk)
	}

	signer, err := ssh.ParsePrivateKey(p)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to parse key file %q", pk)
	}

	return ssh.PublicKeys(signer), nil
}
