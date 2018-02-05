package hostspool

import (
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

func cleanupHostsPool(t *testing.T, cc *api.Client) {
	t.Helper()
	_, err := cc.KV().DeleteTree(consulutil.HostsPoolPrefix, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func testConsulManagerAddTags(t *testing.T, cc *api.Client) {
	cleanupHostsPool(t, cc)
	cm := NewManager(cc)
	err := cm.Add("host_tags_update", Connection{PrivateKey: "key.pem"}, map[string]string{
		"tag1":         "val1",
		"tag/&special": "val/&special",
	})
	require.NoError(t, err)
	err = cm.Add("host_no_tags_update", Connection{PrivateKey: "key.pem"}, map[string]string{
		"tag1": "val1",
	})
	require.NoError(t, err)
	type args struct {
		hostname string
		tags     map[string]string
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		checks     map[string]string
		errorCheck func(error) bool
	}{
		{"TestMissingHostName", args{"", nil}, true, nil, IsBadRequestError},
		{"TestHostNameDoesNotExist", args{"does_not_exist", map[string]string{"t1": "v1"}}, true, nil, IsHostNotFoundError},
		{"TestEmptyTags", args{"host_tags_update", map[string]string{
			"tag1":         "update1",
			"tag/&special": "update/&special",
			"":             "added",
		}}, true, map[string]string{
			"tags/tag1":           "val1",
			"tags/tag%2F&special": "val/&special",
		}, nil},
		{"TestTagsUpdates", args{"host_tags_update", map[string]string{
			"tag1":         "update1",
			"tag/&special": "update/&special",
			"addition":     "added",
		}}, false, map[string]string{
			"tags/tag1":           "update1",
			"tags/tag%2F&special": "update/&special",
			"tags/addition":       "added",
		}, nil},
		{"TestTagsNoUpdatesNil", args{"host_no_tags_update", nil}, false, map[string]string{
			"tags/tag1": "val1",
		}, nil},
		{"TestTagsNoUpdatesEmpty", args{"host_no_tags_update", map[string]string{}}, false, map[string]string{
			"tags/tag1": "val1",
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if err = cm.AddTags(tt.args.hostname, tt.args.tags); (err != nil) != tt.wantErr {
				t.Fatalf("consulManager.AddTags() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errorCheck != nil {
				assert.True(t, tt.errorCheck(err), "consulManager.AddTags() unexpected error %T", err)
			}
			for k, v := range tt.checks {
				kvp, _, err := cc.KV().Get(path.Join(consulutil.HostsPoolPrefix, tt.args.hostname, k), nil)
				if err != nil {
					t.Fatalf("consulManager.AddTags() consul comm error during result check (you should retry) %v", err)
				}
				if kvp == nil && v != "<nil>" {
					t.Fatalf("consulManager.AddTags() missing required key %q", k)
				}
				if kvp != nil && v == "<nil>" {
					t.Fatalf("consulManager.AddTags() expecting key %q to not exists", k)
				}
				if kvp != nil && string(kvp.Value) != v {
					t.Errorf("consulManager.AddTags() key check failed actual = %q, expected %q", string(kvp.Value), v)
				}
			}
		})
	}
	tags, _, err := cc.KV().Keys(path.Join(consulutil.HostsPoolPrefix, "host_no_tags_update", "tags")+"/", "/", nil)
	require.NoError(t, err)
	assert.Len(t, tags, 1, `Expecting only one tag for host "host_no_tags_update", something updated those tags`)
}
func testConsulManagerRemoveTags(t *testing.T, cc *api.Client) {
	cleanupHostsPool(t, cc)
	cm := NewManager(cc)
	err := cm.Add("host_tags_remove", Connection{PrivateKey: "key.pem"}, map[string]string{
		"tag1":         "val1",
		"tag/&special": "val/&special",
		"tagSurvivor":  "still here!",
	})
	require.NoError(t, err)
	err = cm.Add("host_no_tags_remove", Connection{PrivateKey: "key.pem"}, map[string]string{
		"tag1": "val1",
	})
	require.NoError(t, err)
	type args struct {
		hostname string
		tags     []string
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		checks     map[string]string
		errorCheck func(error) bool
	}{
		{"TestMissingHostName", args{"", nil}, true, nil, IsBadRequestError},
		{"TestHostNameDoesNotExist", args{"does_not_exist", []string{"tag1"}}, true, nil, IsHostNotFoundError},
		{"TestEmptyTag", args{"host_tags_remove", []string{""}}, true, nil, IsBadRequestError},
		{"TestTagsUpdates", args{"host_tags_remove", []string{"tag1", "tag/&special"}}, false, map[string]string{
			"tag1":             "<nil>",
			"tag/&special":     "<nil>",
			"tags/tagSurvivor": "still here!",
		}, nil},
		{"TestTagsNoUpdatesNil", args{"host_no_tags_remove", nil}, false, map[string]string{
			"tags/tag1": "val1",
		}, nil},
		{"TestTagsNoUpdatesEmpty", args{"host_no_tags_remove", []string{}}, false, map[string]string{
			"tags/tag1": "val1",
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if err = cm.RemoveTags(tt.args.hostname, tt.args.tags); (err != nil) != tt.wantErr {
				t.Fatalf("consulManager.RemoveTags() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errorCheck != nil {
				assert.True(t, tt.errorCheck(err), "consulManager.AddTags() unexpected error %T", err)
			}
			for k, v := range tt.checks {
				kvp, _, err := cc.KV().Get(path.Join(consulutil.HostsPoolPrefix, tt.args.hostname, k), nil)
				if err != nil {
					t.Fatalf("consulManager.RemoveTags() consul comm error during result check (you should retry) %v", err)
				}
				if kvp == nil && v != "<nil>" {
					t.Fatalf("consulManager.RemoveTags() missing required key %q", k)
				}
				if kvp != nil && v == "<nil>" {
					t.Fatalf("consulManager.RemoveTags() expecting key %q to not exists", k)
				}
				if kvp != nil && string(kvp.Value) != v {
					t.Errorf("consulManager.RemoveTags() key check failed actual = %q, expected %q", string(kvp.Value), v)
				}
			}
		})
	}
	tags, _, err := cc.KV().Keys(path.Join(consulutil.HostsPoolPrefix, "host_no_tags_remove", "tags")+"/", "/", nil)
	require.NoError(t, err)
	assert.Len(t, tags, 1, `Expecting only one tag for host "host_no_tags_remove", something updated those tags`)
}
func testConsulManagerAdd(t *testing.T, cc *api.Client) {
	cleanupHostsPool(t, cc)
	type args struct {
		hostname string
		conn     Connection
		tags     map[string]string
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		checks     map[string]string
		errorCheck func(error) bool
	}{
		{"TestMissingHostName", args{"", Connection{Password: "test"}, nil}, true, nil, IsBadRequestError},
		{"TestMissingConnectionSecret", args{"host1", Connection{}, nil}, true, nil, IsBadRequestError},
		{"TestEmptyTag", args{"host1", Connection{Password: "test"}, map[string]string{"tag1": "v1", "": "val2"}}, true, nil, IsBadRequestError},
		{"TestConnectionDefaults", args{"host1", Connection{Password: "test"}, nil}, false, map[string]string{
			"connection/user":     "root",
			"connection/password": "test",
			"connection/host":     "host1",
			"connection/port":     "22"},
			nil},
		{"TestHostAlreadyExists", args{"host1", Connection{Password: "test2"}, nil}, true, nil, IsHostAlreadyExistError},
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
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := NewManager(cc)
			var err error
			if err = cm.Add(tt.args.hostname, tt.args.conn, tt.args.tags); (err != nil) != tt.wantErr {
				t.Fatalf("consulManager.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errorCheck != nil {
				assert.True(t, tt.errorCheck(err), "consulManager.AddTags() unexpected error %T", err)
			}
			for k, v := range tt.checks {
				kvp, _, err := cc.KV().Get(path.Join(consulutil.HostsPoolPrefix, tt.args.hostname, k), nil)
				if err != nil {
					t.Fatalf("consulManager.Add() consul comm error during result check (you should retry) %v", err)
				}
				if kvp == nil && v != "<nil>" {
					t.Fatalf("consulManager.Add() missing required key %q", k)
				}
				if kvp != nil && v == "<nil>" {
					t.Fatalf("consulManager.Add() expecting key %q to not exists", k)
				}
				if kvp != nil && string(kvp.Value) != v {
					t.Errorf("consulManager.Add() key check failed actual = %q, expected %q", string(kvp.Value), v)
				}
			}
		})
	}
}

func testConsulManagerUpdateConn(t *testing.T, cc *api.Client) {
	cleanupHostsPool(t, cc)
	cm := NewManager(cc)
	originalConn := Connection{User: "u1", Password: "test", Host: "h1", Port: 24, PrivateKey: "key.pem"}
	cm.Add("hostUpdateConn1", originalConn, nil)
	type args struct {
		hostname string
		conn     Connection
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		checks     map[string]string
		errorCheck func(error) bool
	}{
		{"TestMissingHostName", args{"", Connection{}}, true, nil, IsBadRequestError},
		{"TestHostNotFound", args{"does_not_exist", Connection{}}, true, nil, IsHostNotFoundError},
		{"TestEmptyConnNoModification", args{"hostUpdateConn1", Connection{}}, false, map[string]string{
			"connection/user":        originalConn.User,
			"connection/password":    originalConn.Password,
			"connection/port":        fmt.Sprint(originalConn.Port),
			"connection/private_key": originalConn.PrivateKey,
			"connection/host":        originalConn.Host,
		}, nil},
		{"TestConnectionDefaults", args{"hostUpdateConn1", Connection{
			User:       "root",
			Password:   "test",
			Host:       "host1",
			PrivateKey: "new_key.pem",
			Port:       22,
		}}, false, map[string]string{
			"connection/user":        "root",
			"connection/password":    "test",
			"connection/host":        "host1",
			"connection/port":        "22",
			"connection/private_key": "new_key.pem"},
			nil},
		{"TestConnectionCantRemoveBothPrivateKeyAndPassword", args{"hostUpdateConn1", Connection{
			Password:   "-",
			PrivateKey: "-",
		}}, true, nil, IsBadRequestError},
		{"TestConnectionRemovePass", args{"hostUpdateConn1", Connection{
			Password: "-",
		}}, false, map[string]string{
			"connection/password":    "",
			"connection/private_key": "new_key.pem"},
			nil},
		{"TestConnectionCantRemovePKifNoPass", args{"hostUpdateConn1", Connection{
			PrivateKey: "-",
		}}, true, nil, IsBadRequestError},
		{"TestConnectionSwithPKandPass", args{"hostUpdateConn1", Connection{
			PrivateKey: "-",
			Password:   "mypass",
		}}, false, map[string]string{
			"connection/password":    "mypass",
			"connection/private_key": ""},
			nil},
		{"TestConnectionCantRemovePassIfNoPK", args{"hostUpdateConn1", Connection{
			Password: "-",
		}}, true, nil, IsBadRequestError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if err = cm.UpdateConnection(tt.args.hostname, tt.args.conn); (err != nil) != tt.wantErr {
				t.Fatalf("consulManager.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errorCheck != nil {
				assert.True(t, tt.errorCheck(err), "consulManager.AddTags() unexpected error %T", err)
			}
			for k, v := range tt.checks {
				kvp, _, err := cc.KV().Get(path.Join(consulutil.HostsPoolPrefix, tt.args.hostname, k), nil)
				if err != nil {
					t.Fatalf("consulManager.Add() consul comm error during result check (you should retry) %v", err)
				}
				if kvp == nil && v != "<nil>" {
					t.Fatalf("consulManager.Add() missing required key %q", k)
				}
				if kvp != nil && v == "<nil>" {
					t.Fatalf("consulManager.Add() expecting key %q to not exists", k)
				}
				if kvp != nil && string(kvp.Value) != v {
					t.Errorf("consulManager.Add() key check failed actual = %q, expected %q", string(kvp.Value), v)
				}
			}
		})
	}
}

func testConsulManagerRemove(t *testing.T, cc *api.Client) {
	cleanupHostsPool(t, cc)
	cm := NewManager(cc)
	cm.Add("host1", Connection{PrivateKey: "key.pem"}, nil)
	cm.Add("host2", Connection{PrivateKey: "key.pem"}, nil)
	_, err := cc.KV().Put(&api.KVPair{Key: path.Join(consulutil.HostsPoolPrefix, "host2", "status"), Value: []byte(HostStatusAllocated.String())}, nil)
	require.NoError(t, err)
	type args struct {
		hostname string
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		errorCheck func(error) bool
	}{
		{"TestRemoveOK", args{"host1"}, false, nil},
		{"TestRemoveFailedAllocated", args{"host2"}, true, IsBadRequestError},
		{"TestRemoveFailedDoesNotExist", args{"host3"}, true, IsHostNotFoundError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if err := cm.Remove(tt.args.hostname); (err != nil) != tt.wantErr {
				t.Errorf("consulManager.Remove() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errorCheck != nil {
				assert.True(t, tt.errorCheck(err), "consulManager.AddTags() unexpected error %T", err)
			}
		})
	}
}

func testConsulManagerList(t *testing.T, cc *api.Client) {
	cleanupHostsPool(t, cc)
	cm := &consulManager{cc}
	err := cm.Add("list_host1", Connection{PrivateKey: "key.pem"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = cm.Add("list_host2", Connection{PrivateKey: "key.pem"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = cm.Add("list_host3", Connection{PrivateKey: "key.pem"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = cm.Add("list_host4", Connection{PrivateKey: "key.pem"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	hosts, err := cm.List()
	require.NoError(t, err)
	assert.Len(t, hosts, 4)
	assert.Contains(t, hosts, "list_host1")
	assert.Contains(t, hosts, "list_host2")
	assert.Contains(t, hosts, "list_host3")
	assert.Contains(t, hosts, "list_host4")

}

func testConsulManagerGetHost(t *testing.T, cc *api.Client) {
	cleanupHostsPool(t, cc)
	cm := &consulManager{cc}
	tagList := map[string]string{
		"tag1": "v1",
		"tag2": "v2",
		"tag3": "v3",
	}
	connection := Connection{
		User:       "rootget",
		Password:   "testget",
		Host:       "host1get",
		Port:       26,
		PrivateKey: "keyget.pem",
	}
	err := cm.Add("get_host1", connection, tagList)
	require.NoError(t, err)

	host, err := cm.GetHost("get_host1")
	require.NoError(t, err)
	assert.Equal(t, connection, host.Connection)
	assert.Equal(t, tagList, host.Tags)
	assert.Equal(t, "get_host1", host.Name)
	assert.Equal(t, HostStatusFree, host.Status)

}

func testConsulManagerConcurrency(t *testing.T, cc *api.Client) {
	cleanupHostsPool(t, cc)
	cm := &consulManager{cc}
	err := cm.Add("concurrent_host1", Connection{PrivateKey: "key.pem"}, nil)
	require.NoError(t, err)
	l, err := cc.LockKey(kvLockKey)
	require.NoError(t, err)
	_, err = l.Lock(nil)
	require.NoError(t, err)
	defer l.Unlock()

	err = cm.addWait("concurrent_host2", Connection{PrivateKey: "key.pem"}, nil, 500*time.Millisecond)
	assert.Error(t, err, "Expecting concurrency lock for addWait()")
	err = cm.removeWait("concurrent_host1", 500*time.Millisecond)
	assert.Error(t, err, "Expecting concurrency lock for removeWait()")
	err = cm.addTagsWait("concurrent_host1", map[string]string{"t1": "v1"}, 500*time.Millisecond)
	assert.Error(t, err, "Expecting concurrency lock for addTagsWait()")
	err = cm.removeTagsWait("concurrent_host1", []string{"t1"}, 500*time.Millisecond)
	assert.Error(t, err, "Expecting concurrency lock for removeTagsWait()")
	err = cm.updateConnWait("concurrent_host1", Connection{}, 500*time.Millisecond)
	assert.Error(t, err, "Expecting concurrency lock for removeTagsWait()")
}
