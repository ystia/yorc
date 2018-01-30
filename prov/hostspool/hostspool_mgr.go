package hostspool

//go:generate go-enum -f=hostspool_mgr.go --lower

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

// HostStatus x ENUM(
// Free,
// Allocated
// )
type HostStatus int

// TODO support winrm for windows hosts

// A Connection holds info used to connect to an host using SSH
type Connection struct {
	// The User that we should use for the connection. Defaults to root.
	User string `json:"user,omitempty"`
	// The Password that we should use for the connection. One of Password or PrivateKey is required. PrivateKey takes the precedence.
	Password string `json:"password,omitempty"`
	// The SSH Private Key that we should use for the connection. One of Password or PrivateKey is required. PrivateKey takes the precedence.
	PrivateKey string `json:"private_key,omitempty"`
	// The address of the Host to connect to. Defaults to the hostname specified during the registration.
	Host string `json:"host,omitempty"`
	// The Port to connect to. Defaults to 22 if set to 0.
	Port uint64 `json:"port,omitempty"`
}

// A Manager is in charge of creating/updating/deleting hosts from the pool
type Manager interface {
	Add(hostname string, connection Connection, tags map[string]string) error
	Remove(hostname string) error
	AddTags(hostname string, tags map[string]string) error
	RemoveTags(hostname string, tags []string) error
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
		return errors.New(`"hostname" missing`)
	}

	if conn.Password == "" && conn.PrivateKey == "" {
		return errors.New(`at least "password" or "private_key" is required for an host pool connection`)
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
			return errors.New("empty tags are not allowed")
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
				return errors.Errorf("an host with the same name already exists in the pool: %s", e.What)
			}
			errs = append(errs, e.What)
		}
		return errors.Errorf("Failed to register host %q: %s", hostname, strings.Join(errs, ", "))
	}

	return nil
}

func (cm *consulManager) Remove(hostname string) error {
	return cm.removeWait(hostname, 45*time.Second)
}
func (cm *consulManager) removeWait(hostname string, maxWaitTime time.Duration) error {
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
		return errors.Errorf("host %q does not exist", hostname)
	}
	status, err := ParseHostStatus(string(kvp.Value))
	if err != nil {
		return errors.Wrapf(err, "invalid status for host %q", hostname)
	}
	if status != HostStatusFree {
		return errors.Errorf("can't delete host %q with status %q", hostname, status.String())
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
		return errors.New(`"hostname" missing`)
	}
	if tags == nil || len(tags) == 0 {
		return nil
	}

	hostKVPrefix := path.Join(consulutil.HostsPoolPrefix, hostname)
	ops := make(api.KVTxnOps, 0)

	for k, v := range tags {
		k = url.PathEscape(k)
		if k == "" {
			return errors.New("empty tags are not allowed")
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
		return errors.Errorf("host %q does not exist", hostname)
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
		return errors.Errorf("Failed to register host %q: %s", hostname, strings.Join(errs, ", "))
	}

	return nil
}

func (cm *consulManager) RemoveTags(hostname string, tags []string) error {
	return cm.removeTagsWait(hostname, tags, 45*time.Second)
}
func (cm *consulManager) removeTagsWait(hostname string, tags []string, maxWaitTime time.Duration) error {
	if hostname == "" {
		return errors.New(`"hostname" missing`)
	}
	if tags == nil || len(tags) == 0 {
		return nil
	}

	hostKVPrefix := path.Join(consulutil.HostsPoolPrefix, hostname)
	ops := make(api.KVTxnOps, 0)

	for _, v := range tags {
		v = url.PathEscape(v)
		if v == "" {
			return errors.New("empty tags are not allowed")
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
		return errors.Errorf("host %q does not exist", hostname)
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
			if e.OpIndex == 0 {
				return errors.Errorf("an host with the same name already exists in the pool: %s", e.What)
			}
			errs = append(errs, e.What)
		}
		return errors.Errorf("Failed to register host %q: %s", hostname, strings.Join(errs, ", "))
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
