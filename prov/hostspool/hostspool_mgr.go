package hostspool

import (
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

// A Manager is in charge of creating/updating/deleting hosts from the pool
type Manager interface {
	Add(hostname string, connection Connection, tags map[string]string) error
	Remove(hostname string) error
	AddTags(hostname string, tags map[string]string) error
	RemoveTags(hostname string, tags []string) error
	UpdateConnection(hostname string, connection Connection) error
	List() ([]string, error)
	GetHost(hostname string) (Host, error)
}

// NewManager creates a Manager backed to Consul
func NewManager(cc *api.Client) Manager {
	return &consulManager{cc: cc}
}

const kvLockKey = consulutil.HostsPoolPrefix + "/.mgrLock"

type consulManager struct {
	cc *api.Client
}

func (cm *consulManager) Add(hostname string, conn Connection, tags map[string]string) error {
	return cm.addWait(hostname, conn, tags, 45*time.Second)
}
func (cm *consulManager) addWait(hostname string, conn Connection, tags map[string]string, maxWaitTime time.Duration) error {
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

	for k, v := range tags {
		k = url.PathEscape(k)
		if k == "" {
			return errors.WithStack(badRequestError{"empty tags are not allowed"})
		}
		ops = append(ops, &api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "tags", k),
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

	return nil
}

func (cm *consulManager) UpdateConnection(hostname string, conn Connection) error {
	return cm.updateConnWait(hostname, conn, 45*time.Second)
}
func (cm *consulManager) updateConnWait(hostname string, conn Connection, maxWaitTime time.Duration) error {
	if hostname == "" {
		return errors.WithStack(badRequestError{`"hostname" missing`})
	}

	// check if host exists
	_, err := cm.GetHostStatus(hostname)
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
		ops = append(ops, &api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "connection", "private_key"),
			Value: []byte(conn.PrivateKey),
		})
	}
	if conn.Password != "" {
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

	kvp, _, err := kv.Get(path.Join(hostKVPrefix, "status"), nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return errors.WithStack(hostNotFoundError{})
	}
	status, err := ParseHostStatus(string(kvp.Value))
	if err != nil {
		return errors.Wrapf(err, "invalid status for host %q", hostname)
	}
	if status != HostStatusFree {
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

func (cm *consulManager) AddTags(hostname string, tags map[string]string) error {
	return cm.addTagsWait(hostname, tags, 45*time.Second)
}
func (cm *consulManager) addTagsWait(hostname string, tags map[string]string, maxWaitTime time.Duration) error {
	if hostname == "" {
		return errors.WithStack(badRequestError{`"hostname" missing`})
	}
	if tags == nil || len(tags) == 0 {
		return nil
	}

	hostKVPrefix := path.Join(consulutil.HostsPoolPrefix, hostname)
	ops := make(api.KVTxnOps, 0)

	for k, v := range tags {
		k = url.PathEscape(k)
		if k == "" {
			return errors.WithStack(badRequestError{"empty tags are not allowed"})
		}
		ops = append(ops, &api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(hostKVPrefix, "tags", k),
			Value: []byte(v),
		})
	}

	_, cleanupFn, err := cm.lockKey(hostname, "tags addition", maxWaitTime)
	if err != nil {
		return err
	}
	defer cleanupFn()

	kv := cm.cc.KV()

	// Checks host existence
	kvp, _, err := kv.Get(path.Join(hostKVPrefix, "status"), nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return errors.WithStack(hostNotFoundError{})
	}

	// We don't care about host status for updating tags

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
		return errors.Errorf("Failed to add tags to host %q: %s", hostname, strings.Join(errs, ", "))
	}

	return nil
}

func (cm *consulManager) RemoveTags(hostname string, tags []string) error {
	return cm.removeTagsWait(hostname, tags, 45*time.Second)
}
func (cm *consulManager) removeTagsWait(hostname string, tags []string, maxWaitTime time.Duration) error {
	if hostname == "" {
		return errors.WithStack(badRequestError{`"hostname" missing`})
	}
	if tags == nil || len(tags) == 0 {
		return nil
	}

	hostKVPrefix := path.Join(consulutil.HostsPoolPrefix, hostname)
	ops := make(api.KVTxnOps, 0)

	for _, v := range tags {
		v = url.PathEscape(v)
		if v == "" {
			return errors.WithStack(badRequestError{"empty tags are not allowed"})
		}
		ops = append(ops, &api.KVTxnOp{
			Verb: api.KVDelete,
			Key:  path.Join(hostKVPrefix, "tags", v),
		})
	}

	_, cleanupFn, err := cm.lockKey(hostname, "tags remove", maxWaitTime)
	if err != nil {
		return err
	}
	defer cleanupFn()

	kv := cm.cc.KV()

	// Checks host existence
	kvp, _, err := kv.Get(path.Join(hostKVPrefix, "status"), nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return errors.WithStack(hostNotFoundError{})
	}

	// We don't care about host status for updating tags

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
		return errors.Errorf("Failed to delete tags on host %q: %s", hostname, strings.Join(errs, ", "))
	}

	return nil
}

func (cm *consulManager) lockKey(hostname, opType string, lockWaitTime time.Duration) (lockCh <-chan struct{}, cleanupFn func(), err error) {
	lock, err := cm.cc.LockOpts(&api.LockOptions{
		Key:            kvLockKey,
		Value:          []byte(fmt.Sprintf("locked for %q %s", hostname, opType)),
		MonitorRetries: 2,
		LockWaitTime:   lockWaitTime,
		LockTryOnce:    true,
		SessionName:    fmt.Sprintf("%q %s", hostname, opType),
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

func (cm *consulManager) List() ([]string, error) {
	hosts, _, err := cm.cc.KV().Keys(consulutil.HostsPoolPrefix+"/", "/", nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for i := range hosts {
		hosts[i] = path.Base(hosts[i])
	}
	return hosts, nil
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

func (cm *consulManager) GetHostTags(hostname string) (map[string]string, error) {
	if hostname == "" {
		return nil, errors.WithStack(badRequestError{`"hostname" missing`})
	}
	// check if host exists
	_, err := cm.GetHostStatus(hostname)
	if err != nil {
		return nil, err
	}
	kvps, _, err := cm.cc.KV().List(path.Join(consulutil.HostsPoolPrefix, hostname, "tags"), nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	tags := make(map[string]string, len(kvps))
	for _, kvp := range kvps {
		tags[path.Base(kvp.Key)] = string(kvp.Value)
	}
	return tags, nil
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

	host.Connection, err = cm.GetHostConnection(hostname)
	if err != nil {
		return host, err
	}

	host.Tags, err = cm.GetHostTags(hostname)
	return host, err
}
