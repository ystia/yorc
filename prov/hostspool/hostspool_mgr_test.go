package hostspool

import (
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

func testConsulManagerAdd(t *testing.T, cc *api.Client) {
	type args struct {
		hostname string
		conn     Connection
		tags     map[string]string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		checks  map[string]string
	}{
		{"TestMissingHostName", args{"", Connection{Password: "test"}, nil}, true, nil},
		{"TestMissingConnectionSecret", args{"host1", Connection{}, nil}, true, nil},
		{"TestConnectionDefaults", args{"host1", Connection{Password: "test"}, nil}, false, map[string]string{
			"connection/user":     "root",
			"connection/password": "test",
			"connection/host":     "host1",
			"connection/port":     "22"},
		},
		{"TestHostAlreadyExists", args{"host1", Connection{Password: "test2"}, nil}, true, nil},
		{"TestTagsSupport", args{"hostwithtags", Connection{
			Host:       "127.0.1.1",
			User:       "ubuntu",
			Port:       23,
			PrivateKey: "path/to/key",
		}, map[string]string{
			"tag1":         "val1",
			"tag/&special": "val&special",
		}}, false, map[string]string{
			"connection/user":        "ubuntu",
			"connection/password":    "",
			"connection/private_key": "path/to/key",
			"connection/host":        "127.0.1.1",
			"connection/port":        "23",
			"tags/tag1":              "val1",
			"tags/tag%2F&special":    "val&special",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := NewManager(cc)
			if err := cm.Add(tt.args.hostname, tt.args.conn, tt.args.tags); (err != nil) != tt.wantErr {
				t.Fatalf("consulManager.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
			for k, v := range tt.checks {
				kvp, _, err := cc.KV().Get(path.Join(consulutil.HostsPoolPrefix, tt.args.hostname, k), nil)
				if err != nil {
					t.Fatalf("consulManager.Add() consul comm error during result check (you should retry) %v", err)
				}
				if kvp == nil && v != "<nil>" {
					t.Fatalf("consulManager.Add() missing required key %q", k)
				}
				if string(kvp.Value) != v {
					t.Errorf("consulManager.Add() key check failed actual = %q, expected %q", string(kvp.Value), v)
				}
			}
		})
	}
}

func testConsulManagerRemove(t *testing.T, cc *api.Client) {
	cm := NewManager(cc)
	cm.Add("host1", Connection{PrivateKey: "key.pem"}, nil)
	cm.Add("host2", Connection{PrivateKey: "key.pem"}, nil)
	_, err := cc.KV().Put(&api.KVPair{Key: path.Join(consulutil.HostsPoolPrefix, "host2", "status"), Value: []byte(HostStatusAllocated.String())}, nil)
	require.NoError(t, err)
	type args struct {
		hostname string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"TestRemoveOK", args{"host1"}, false},
		{"TestRemoveFailedAllocated", args{"host2"}, true},
		{"TestRemoveFailedDoesNotExist", args{"host3"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if err := cm.Remove(tt.args.hostname); (err != nil) != tt.wantErr {
				t.Errorf("consulManager.Remove() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func testConsulManagerConcurrency(t *testing.T, cc *api.Client) {
	cm := &consulManager{cc}
	err := cm.Add("concurrent_host1", Connection{PrivateKey: "key.pem"}, nil)
	require.NoError(t, err)
	l, err := cc.LockKey(kvLockKey)
	require.NoError(t, err)
	_, err = l.Lock(nil)
	require.NoError(t, err)
	defer l.Unlock()

	err = cm.addWait("concurrent_host2", Connection{PrivateKey: "key.pem"}, nil, 1*time.Second)
	assert.Error(t, err)
	err = cm.removeWait("concurrent_host1", 1*time.Second)
	assert.Error(t, err)
}
