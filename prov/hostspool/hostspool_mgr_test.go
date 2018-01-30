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

func testConsulManagerAddTags(t *testing.T, cc *api.Client) {
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
		name    string
		args    args
		wantErr bool
		checks  map[string]string
	}{
		{"TestMissingHostName", args{"", nil}, true, nil},
		{"TestHostNameDoesNotExist", args{"does_not_exist", map[string]string{"t1": "v1"}}, true, nil},
		{"TestEmptyTags", args{"host_tags_update", map[string]string{
			"tag1":         "update1",
			"tag/&special": "update/&special",
			"":             "added",
		}}, true, map[string]string{
			"tags/tag1":           "val1",
			"tags/tag%2F&special": "val/&special",
		}},
		{"TestTagsUpdates", args{"host_tags_update", map[string]string{
			"tag1":         "update1",
			"tag/&special": "update/&special",
			"addition":     "added",
		}}, false, map[string]string{
			"tags/tag1":           "update1",
			"tags/tag%2F&special": "update/&special",
			"tags/addition":       "added",
		}},
		{"TestTagsNoUpdatesNil", args{"host_no_tags_update", nil}, false, map[string]string{
			"tags/tag1": "val1",
		}},
		{"TestTagsNoUpdatesEmpty", args{"host_no_tags_update", map[string]string{}}, false, map[string]string{
			"tags/tag1": "val1",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := cm.AddTags(tt.args.hostname, tt.args.tags); (err != nil) != tt.wantErr {
				t.Fatalf("consulManager.AddTags() error = %v, wantErr %v", err, tt.wantErr)
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
		name    string
		args    args
		wantErr bool
		checks  map[string]string
	}{
		{"TestMissingHostName", args{"", nil}, true, nil},
		{"TestHostNameDoesNotExist", args{"does_not_exist", []string{"tag1"}}, true, nil},
		{"TestEmptyTag", args{"host_tags_remove", []string{""}}, true, nil},
		{"TestTagsUpdates", args{"host_tags_remove", []string{"tag1", "tag/&special"}}, false, map[string]string{
			"tag1":             "<nil>",
			"tag/&special":     "<nil>",
			"tags/tagSurvivor": "still here!",
		}},
		{"TestTagsNoUpdatesNil", args{"host_no_tags_remove", nil}, false, map[string]string{
			"tags/tag1": "val1",
		}},
		{"TestTagsNoUpdatesEmpty", args{"host_no_tags_remove", []string{}}, false, map[string]string{
			"tags/tag1": "val1",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := cm.RemoveTags(tt.args.hostname, tt.args.tags); (err != nil) != tt.wantErr {
				t.Fatalf("consulManager.RemoveTags() error = %v, wantErr %v", err, tt.wantErr)
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
		{"TestEmptyTag", args{"host1", Connection{Password: "test"}, map[string]string{"tag1": "v1", "": "val2"}}, true, nil},
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

	err = cm.addWait("concurrent_host2", Connection{PrivateKey: "key.pem"}, nil, 500*time.Millisecond)
	assert.Error(t, err, "Expecting concurrency lock for addWait()")
	err = cm.removeWait("concurrent_host1", 500*time.Millisecond)
	assert.Error(t, err, "Expecting concurrency lock for removeWait()")
	err = cm.addTagsWait("concurrent_host1", map[string]string{"t1": "v1"}, 500*time.Millisecond)
	assert.Error(t, err, "Expecting concurrency lock for addTagsWait()")
	err = cm.removeTagsWait("concurrent_host1", []string{"t1"}, 500*time.Millisecond)
	assert.Error(t, err, "Expecting concurrency lock for removeTagsWait()")
}
