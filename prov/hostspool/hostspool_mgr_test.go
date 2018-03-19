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
	"fmt"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/consul/api"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/helper/labelsutil"
	"github.com/ystia/yorc/helper/sshutil"
)

var dummySSHkey = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAo3x7OqBJ3ftBxFPaEXrRdmJXo8xqNxNVm7i5tJSqGICzqO2P
ft1GW7PDb8EgZn8UM/xuQLBfJEStovMaWD8hiCN3ekEguAMrcZ2g+/7QOxwicpEc
054eH5CmIOyGpcED6KGVLdTWSGa/MfLW6HHAyjdeeX4NPA0tPm4kswT6ehOc6otX
adjCh0H83Nx+qqSvAIAIkeofw/AEd8Zbmfz9+iibYHwwaaPmVuNqj93FvcpK3Bma
8Hlo4KoeCmI8XD1ixpeJq83wJf5wEXTaw1zajQToUegWITofP8v9mbbCuS+cwZO2
YvVYIzr2/oHaSDD54O+yQ3H7bbR/1EQNWmYtAwIDAQABAoIBABxrqnyBmvXFFSTN
Mu6w/DLpW7T0904FxW8hyN7UrVE/Jnxqd/SlAPM2J/aIi1pmIxv6eSwzvQZwDgNy
4ZSPvQOOrtmI8ugqXOYOcgr8vDRaar6h7XH4XeI84jR9CddM26IYXPevtWS2v+wt
/CBCjjJZN8pFGIXlAIWG3khkyCpqcUj0yZZoLadA8aaxZmp+h4bs638awRRVbvzz
8sqaWaVcX6H2MLtFb8IjxLPWqVx7e1UWuUbpmIAnMJtmEhT3vEAyWSAuOvwUXz9S
wxK3eewJNPzehOK5mAqNhSLGSo3UExVYFxFCXbZE+7WH8KqeDoZp6lp/gRKwXTGG
t18ZT9kCgYEA0ddxBshuzk+grEGoHEBs67pSjezR2nHTq1nwzowDB4PQwBGXZq7q
Q/LbQUbKHS9f5mVpPziGIxnSNoNemFpX9quReghZ5u70dm7GZzZaltZCjbKcma4l
Jms0bHDDtseHHQwjIY/pQKeaubt7HVEPc4+pGxovyMWSRg6ZJVvD0YcCgYEAx3Kw
Mx+FOyrr08s+hBPG9pXS02xOwZmZOpI62elv5t/oEaIJVL2TNcfU3a7Vv8848LVJ
9F1xuV4j8rRGop0IYW8R/EHs/Oq+2c1rR6JAyLOzTHEnFyCKACzAYSKUsCEOPqYc
gxZ4Qz11Xf0s17yrnyieI3oEzBPBP4Ei5eL7F6UCgYB1nvxk3+50SF/4jijsBRTI
oTzq/sa2Wj1ae+Sl8gc0rCdTsciarwrzIWrS0Rozd72aiFeRL17IyA1zrvlUDrfl
tU+rBolWD7UJuZgOfIIUsG7HvElZPyrluQu+iQq7JmZO2uHKSz9klU3+M9+TlD9D
+E/CuE/2iwAtsrsXHLPLewKBgC7uOLG+3/29KsKqV2qCsNWDCZnAKYP6nYifsgNm
n3MnCpdjlmh/Ny13eQo0wo0guJhDQESk3Eau9Sx96QUIiFlM5mGCLb6Rihj78htn
/XB8gFsjYPxbJr3FyfrRRUVwccaiFaFu3xuLUZutICkfdw67YwKcCpbuqxFDVK/d
ShIVAoGBAMe5pMcaQhjwGMcsNySH0cTF+0xWbFqoiMZJYtqe6Bi3C9ZM4wCfQDVI
WSPb8a8DvmtjA0/IyWLtRhiKDPo0ultXa3DHINOC2+sZ5qJpsycNaqV4ObcmCJCc
h5pSY3nqKgmTiTW5EGnhLxUnEmS0MMvVT59ldx2pZhzgDyxYWO09
-----END RSA PRIVATE KEY-----`

type mockSSHClient struct{}

func (m *mockSSHClient) RunCommand(string) (string, error) {
	return "ok", nil
}

var mockSSHClientFactory = func(config *ssh.ClientConfig, conn Connection) sshutil.Client {
	return &mockSSHClient{}
}

func cleanupHostsPool(t *testing.T, cc *api.Client) {
	t.Helper()
	_, err := cc.KV().DeleteTree(consulutil.HostsPoolPrefix, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func testConsulManagerAddLabels(t *testing.T, cc *api.Client) {
	cleanupHostsPool(t, cc)
	cm := NewManagerWithSSHFactory(cc, mockSSHClientFactory)
	err := cm.Add("host_labels_update", Connection{PrivateKey: dummySSHkey}, map[string]string{
		"label1":         "val1",
		"label/&special": "val/&special",
	})
	require.NoError(t, err)
	err = cm.Add("host_no_labels_update", Connection{PrivateKey: dummySSHkey}, map[string]string{
		"label1": "val1",
	})
	require.NoError(t, err)
	type args struct {
		hostname string
		labels   map[string]string
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
		{"TestEmptyLabels", args{"host_labels_update", map[string]string{
			"label1":         "update1",
			"label/&special": "update/&special",
			"":               "added",
		}}, true, map[string]string{
			"labels/label1":           "val1",
			"labels/label%2F&special": "val/&special",
		}, nil},
		{"TestLabelsUpdates", args{"host_labels_update", map[string]string{
			"label1":         "update1",
			"label/&special": "update/&special",
			"addition":       "added",
		}}, false, map[string]string{
			"labels/label1":           "update1",
			"labels/label%2F&special": "update/&special",
			"labels/addition":         "added",
		}, nil},
		{"TestLabelsNoUpdatesNil", args{"host_no_labels_update", nil}, false, map[string]string{
			"labels/label1": "val1",
		}, nil},
		{"TestLabelsNoUpdatesEmpty", args{"host_no_labels_update", map[string]string{}}, false, map[string]string{
			"labels/label1": "val1",
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if err = cm.AddLabels(tt.args.hostname, tt.args.labels); (err != nil) != tt.wantErr {
				t.Fatalf("consulManager.AddLabels() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errorCheck != nil {
				assert.True(t, tt.errorCheck(err), "consulManager.AddLabels() unexpected error %T", err)
			}
			for k, v := range tt.checks {
				kvp, _, err := cc.KV().Get(path.Join(consulutil.HostsPoolPrefix, tt.args.hostname, k), nil)
				if err != nil {
					t.Fatalf("consulManager.AddLabels() consul comm error during result check (you should retry) %v", err)
				}
				if kvp == nil && v != "<nil>" {
					t.Fatalf("consulManager.AddLabels() missing required key %q", k)
				}
				if kvp != nil && v == "<nil>" {
					t.Fatalf("consulManager.AddLabels() expecting key %q to not exists", k)
				}
				if kvp != nil && string(kvp.Value) != v {
					t.Errorf("consulManager.AddLabels() key check failed actual = %q, expected %q", string(kvp.Value), v)
				}
			}
		})
	}
	labels, _, err := cc.KV().Keys(path.Join(consulutil.HostsPoolPrefix, "host_no_labels_update", "labels")+"/", "/", nil)
	require.NoError(t, err)
	assert.Len(t, labels, 1, `Expecting only one label for host "host_no_labels_update", something updated those labels`)
}
func testConsulManagerRemoveLabels(t *testing.T, cc *api.Client) {
	cleanupHostsPool(t, cc)
	cm := NewManagerWithSSHFactory(cc, mockSSHClientFactory)
	err := cm.Add("host_labels_remove", Connection{PrivateKey: dummySSHkey}, map[string]string{
		"label1":         "val1",
		"label/&special": "val/&special",
		"labelSurvivor":  "still here!",
	})
	require.NoError(t, err)
	err = cm.Add("host_no_labels_remove", Connection{PrivateKey: dummySSHkey}, map[string]string{
		"label1": "val1",
	})
	require.NoError(t, err)
	type args struct {
		hostname string
		labels   []string
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		checks     map[string]string
		errorCheck func(error) bool
	}{
		{"TestMissingHostName", args{"", nil}, true, nil, IsBadRequestError},
		{"TestHostNameDoesNotExist", args{"does_not_exist", []string{"label1"}}, true, nil, IsHostNotFoundError},
		{"TestEmptyLabel", args{"host_labels_remove", []string{""}}, true, nil, IsBadRequestError},
		{"TestLabelsUpdates", args{"host_labels_remove", []string{"label1", "label/&special"}}, false, map[string]string{
			"label1":               "<nil>",
			"label/&special":       "<nil>",
			"labels/labelSurvivor": "still here!",
		}, nil},
		{"TestLabelsNoUpdatesNil", args{"host_no_labels_remove", nil}, false, map[string]string{
			"labels/label1": "val1",
		}, nil},
		{"TestLabelsNoUpdatesEmpty", args{"host_no_labels_remove", []string{}}, false, map[string]string{
			"labels/label1": "val1",
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if err = cm.RemoveLabels(tt.args.hostname, tt.args.labels); (err != nil) != tt.wantErr {
				t.Fatalf("consulManager.RemoveLabels() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errorCheck != nil {
				assert.True(t, tt.errorCheck(err), "consulManager.AddLabels() unexpected error %T", err)
			}
			for k, v := range tt.checks {
				kvp, _, err := cc.KV().Get(path.Join(consulutil.HostsPoolPrefix, tt.args.hostname, k), nil)
				if err != nil {
					t.Fatalf("consulManager.RemoveLabels() consul comm error during result check (you should retry) %v", err)
				}
				if kvp == nil && v != "<nil>" {
					t.Fatalf("consulManager.RemoveLabels() missing required key %q", k)
				}
				if kvp != nil && v == "<nil>" {
					t.Fatalf("consulManager.RemoveLabels() expecting key %q to not exists", k)
				}
				if kvp != nil && string(kvp.Value) != v {
					t.Errorf("consulManager.RemoveLabels() key check failed actual = %q, expected %q", string(kvp.Value), v)
				}
			}
		})
	}
	labels, _, err := cc.KV().Keys(path.Join(consulutil.HostsPoolPrefix, "host_no_labels_remove", "labels")+"/", "/", nil)
	require.NoError(t, err)
	assert.Len(t, labels, 1, `Expecting only one label for host "host_no_labels_remove", something updated those labels`)
}
func testConsulManagerAdd(t *testing.T, cc *api.Client) {
	cleanupHostsPool(t, cc)
	type args struct {
		hostname string
		conn     Connection
		labels   map[string]string
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
		{"TestEmptyLabel", args{"host1", Connection{Password: "test"}, map[string]string{"label1": "v1", "": "val2"}}, true, nil, IsBadRequestError},
		{"TestConnectionDefaults", args{"host1", Connection{Password: "test"}, nil}, false, map[string]string{
			"connection/user":     "root",
			"connection/password": "test",
			"connection/host":     "host1",
			"connection/port":     "22"},
			nil},
		{"TestHostAlreadyExists", args{"host1", Connection{Password: "test2"}, nil}, true, nil, IsHostAlreadyExistError},
		{"TestLabelsSupport", args{"hostwithlabels", Connection{
			Host:       "127.0.1.1",
			User:       "ubuntu",
			Port:       23,
			PrivateKey: "testdata/new_key.pem",
		}, map[string]string{
			"label1":         "val1",
			"label/&special": "val&special",
		}}, false, map[string]string{
			"connection/user":         "ubuntu",
			"connection/password":     "",
			"connection/private_key":  "testdata/new_key.pem",
			"connection/host":         "127.0.1.1",
			"connection/port":         "23",
			"labels/label1":           "val1",
			"labels/label%2F&special": "val&special",
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := NewManagerWithSSHFactory(cc, mockSSHClientFactory)
			var err error
			if err = cm.Add(tt.args.hostname, tt.args.conn, tt.args.labels); (err != nil) != tt.wantErr {
				t.Fatalf("consulManager.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errorCheck != nil {
				assert.True(t, tt.errorCheck(err), "consulManager.AddLabels() unexpected error %T", err)
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
	cm := NewManagerWithSSHFactory(cc, mockSSHClientFactory)
	originalConn := Connection{User: "u1", Password: "test", Host: "h1", Port: 24, PrivateKey: dummySSHkey}
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
			PrivateKey: "testdata/new_key.pem",
			Port:       22,
		}}, false, map[string]string{
			"connection/user":        "root",
			"connection/password":    "test",
			"connection/host":        "host1",
			"connection/port":        "22",
			"connection/private_key": "testdata/new_key.pem"},
			nil},
		{"TestConnectionCantRemoveBothPrivateKeyAndPassword", args{"hostUpdateConn1", Connection{
			Password:   "-",
			PrivateKey: "-",
		}}, true, nil, IsBadRequestError},
		{"TestConnectionRemovePass", args{"hostUpdateConn1", Connection{
			Password: "-",
		}}, false, map[string]string{
			"connection/password":    "",
			"connection/private_key": "testdata/new_key.pem"},
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
				assert.True(t, tt.errorCheck(err), "consulManager.AddLabels() unexpected error %T", err)
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
	cm := NewManagerWithSSHFactory(cc, mockSSHClientFactory)
	cm.Add("host1", Connection{PrivateKey: dummySSHkey}, nil)
	cm.Add("host2", Connection{PrivateKey: dummySSHkey}, nil)
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
				assert.True(t, tt.errorCheck(err), "consulManager.AddLabels() unexpected error %T", err)
			}
		})
	}
}

func testConsulManagerList(t *testing.T, cc *api.Client) {
	cleanupHostsPool(t, cc)
	cm := &consulManager{cc, mockSSHClientFactory}
	err := cm.Add("list_host1", Connection{PrivateKey: dummySSHkey}, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = cm.Add("list_host2", Connection{PrivateKey: dummySSHkey}, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = cm.Add("list_host3", Connection{PrivateKey: dummySSHkey}, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = cm.Add("list_host4", Connection{PrivateKey: dummySSHkey}, nil)
	if err != nil {
		t.Fatal(err)
	}

	hosts, warnings, version, err := cm.List()
	require.NoError(t, err)
	assert.Len(t, warnings, 0)
	assert.Len(t, hosts, 4)
	assert.Contains(t, hosts, "list_host1")
	assert.Contains(t, hosts, "list_host2")
	assert.Contains(t, hosts, "list_host3")
	assert.Contains(t, hosts, "list_host4")

	// Check version increase
	err = cm.Add("list_host5", Connection{PrivateKey: dummySSHkey}, nil)
	if err != nil {
		t.Fatal(err)
	}
	hosts, warnings, version2, err := cm.List()
	require.NoError(t, err)
	assert.Len(t, warnings, 0)
	assert.Len(t, hosts, 5)
	assert.NotEqual(t, version, version2, "Expected a version change after update")

}

func testConsulManagerGetHost(t *testing.T, cc *api.Client) {
	cleanupHostsPool(t, cc)
	cm := &consulManager{cc, mockSSHClientFactory}
	labelList := map[string]string{
		"label1": "v1",
		"label2": "v2",
		"label3": "v3",
	}
	connection := Connection{
		User:       "rootget",
		Password:   "testget",
		Host:       "host1get",
		Port:       26,
		PrivateKey: dummySSHkey,
	}
	err := cm.Add("get_host1", connection, labelList)
	require.NoError(t, err)

	host, err := cm.GetHost("get_host1")
	require.NoError(t, err)
	assert.Equal(t, connection, host.Connection)
	assert.Equal(t, labelList, host.Labels)
	assert.Equal(t, "get_host1", host.Name)
	assert.Equal(t, HostStatusFree, host.Status)

}

func testConsulManagerConcurrency(t *testing.T, cc *api.Client) {
	cleanupHostsPool(t, cc)
	cm := &consulManager{cc, mockSSHClientFactory}
	err := cm.Add("concurrent_host1", Connection{PrivateKey: dummySSHkey}, nil)
	require.NoError(t, err)
	l, err := cc.LockKey(kvLockKey)
	require.NoError(t, err)
	_, err = l.Lock(nil)
	require.NoError(t, err)
	defer l.Unlock()

	err = cm.addWait("concurrent_host2", Connection{PrivateKey: dummySSHkey}, nil, 500*time.Millisecond)
	assert.Error(t, err, "Expecting concurrency lock for addWait()")
	err = cm.removeWait("concurrent_host1", 500*time.Millisecond)
	assert.Error(t, err, "Expecting concurrency lock for removeWait()")
	err = cm.addLabelsWait("concurrent_host1", map[string]string{"t1": "v1"}, 500*time.Millisecond)
	assert.Error(t, err, "Expecting concurrency lock for addLabelsWait()")
	err = cm.removeLabelsWait("concurrent_host1", []string{"t1"}, 500*time.Millisecond)
	assert.Error(t, err, "Expecting concurrency lock for removeLabelsWait()")
	err = cm.updateConnWait("concurrent_host1", Connection{}, 500*time.Millisecond)
	assert.Error(t, err, "Expecting concurrency lock for removeLabelsWait()")
	_, _, err = cm.allocateWait(500*time.Millisecond, "test message")
	assert.Error(t, err, "Expecting concurrency lock for allocateWait()")
	err = cm.releaseWait("concurrent_host1", 500*time.Millisecond)
	assert.Error(t, err, "Expecting concurrency lock for releaseWait()")
}

func testConsulManagerApply(t *testing.T, cc *api.Client) {
	cleanupHostsPool(t, cc)
	cm := &consulManager{cc, mockSSHClientFactory}

	var hostpool []Host

	for i := 0; i < 3; i++ {
		suffix := strconv.Itoa(i)
		hostpool = append(hostpool, Host{
			Name: "host" + suffix,
			Connection: Connection{
				User:     "testuser" + suffix,
				Password: "testpwd" + suffix,
				Host:     "testhost" + suffix,
				Port:     uint64(i + 1),
			},
			Labels: map[string]string{
				"label1": "value1" + suffix,
				"label2": "value2" + suffix,
				"label3": "value3" + suffix,
			},
		})
	}

	// Apply this definition
	var version uint64
	err := cm.Apply(hostpool, &version)
	require.NoError(t, err, "Unexpected failure applying host pool definition")
	// Check the pool now
	hosts, warnings, version, err := cm.List()
	require.NoError(t, err, "Unexpected error getting list of hosts in pool")
	assert.Len(t, warnings, 0)
	assert.Len(t, hosts, 3)
	assert.NotEqual(t, 0, version)
	for i := 0; i < 3; i++ {
		suffix := strconv.Itoa(i)
		hostname := "host" + suffix
		assert.Contains(t, hosts, hostname)
		host, err := cm.GetHost(hostname)
		require.NoError(t, err, "Could not get host %s", hostname)
		assert.Equal(t, hostpool[i].Connection, host.Connection,
			"Unexpected connection value for host %s", hostname)
		assert.Equal(t, hostpool[i].Labels, host.Labels,
			"Unexpected labels for host %s", hostname)
	}

	// Allocate host1
	allocationMsg := "For tests purposes"
	filterLabel := "label2"
	filter, err := labelsutil.CreateFilter(
		fmt.Sprintf("%s=%s", filterLabel, hostpool[1].Labels[filterLabel]))
	require.NoError(t, err, "Unexpected error creating a filter")
	allocatedName, warnings, err := cm.Allocate(allocationMsg, filter)
	assert.Equal(t, hostpool[1].Name, allocatedName,
		"Unexpected host allocated")
	allocatedHost, err := cm.GetHost(allocatedName)
	require.NoError(t, err, "Unexpected error getting allocated host")
	assert.Equal(t, allocationMsg, allocatedHost.Message,
		"Unexpected allocation message for allocated host")

	// Change the pool definition by :
	// - changing connection settings of host0
	// - changing labels of host1
	// - removing host2
	// - adding host3 and host4
	hostpool[0].Connection = Connection{
		User:       "newUser",
		PrivateKey: dummySSHkey,
		Host:       "testhost0.example.com",
		Port:       123,
	}

	hostpool[1].Labels = map[string]string{
		"newlabel1": "newvalue1",
		"newlabel2": "newvalue2",
		"label2":    hostpool[1].Labels[filterLabel],
		"label3":    "value32",
	}

	// Removing host2
	removedHostname := hostpool[len(hostpool)-1].Name
	hostpool = hostpool[:len(hostpool)-1]

	// Adding new hosts
	for i := 3; i < 5; i++ {
		suffix := strconv.Itoa(i)
		hostpool = append(hostpool, Host{
			Name: "host" + suffix,
			Connection: Connection{
				User:     "testuser" + suffix,
				Password: "testpwd" + suffix,
				Host:     "testhost" + suffix,
				Port:     uint64(i + 1),
			},
			Labels: map[string]string{
				"label1": "value1" + suffix,
				"label2": "value2" + suffix,
				"label3": "value3" + suffix,
			},
		})
	}

	// Apply this new definition
	err = cm.Apply(hostpool, &version)
	require.NoError(t, err,
		"Unexpected failure applying new host pool definition")

	// Check the pool now
	hosts, warnings, version2, err := cm.List()
	require.NoError(t, err, "Unexpected error getting list of hosts in pool")
	assert.Len(t, warnings, 0)
	assert.Len(t, hosts, 4)
	assert.NotEqual(t, version, version2, "Expected a version change")
	assert.NotContains(t, hosts, removedHostname)
	for i := 0; i < 4; i++ {
		var suffix string
		if i < 2 {
			suffix = strconv.Itoa(i)
		} else {
			suffix = strconv.Itoa(i + 1)
		}

		hostname := "host" + suffix
		assert.Contains(t, hosts, hostname)
		host, err := cm.GetHost(hostname)
		require.NoError(t, err, "Could not get host %s", hostname)
		assert.Equal(t, hostpool[i].Connection, host.Connection,
			"Unexpected connection value for host %s", hostname)
		assert.Equal(t, hostpool[i].Labels, host.Labels,
			"Unexpected labels for host %s", hostname)
	}

	// Check the allocated status of host1 didn't change
	allocatedHost, err = cm.GetHost(allocatedName)
	require.NoError(t, err, "Unexpected error getting allocated host")
	assert.Equal(t, allocationMsg, allocatedHost.Message)
	assert.Equal(t, allocationMsg, allocatedHost.Message,
		"Unexpected alloc message for allocated host after Pool redefinition")
	assert.Equal(t, HostStatusAllocated, allocatedHost.Status,
		"Unexpected status for an allocated host after Pool redefinition")

	// Error case: entry with no name
	oldName := hostpool[0].Name
	hostpool[0].Name = "newName"
	hostpool = append(hostpool, hostpool[0])
	hostpool[len(hostpool)-1].Name = ""
	err = cm.Apply(hostpool, &version2)
	assert.Error(t, err, "Expected an error adding a host with no name")

	// Check the new definition wasn't applied after this error
	hosts, warnings, version3, err := cm.List()
	require.NoError(t, err, "Unexpected error getting list of hosts in pool")
	assert.Len(t, warnings, 0)
	assert.Len(t, hosts, 4)
	assert.Equal(t, version2, version3, "Expected no version change after name error")
	assert.Contains(t, hosts, oldName,
		"Hosts Pool unexpectedly changed after an apply error using empty name")
	assert.NotContains(t, hosts, "newName")

	// Error case: duplicate names
	hostpool[len(hostpool)-1].Name = hostpool[0].Name
	err = cm.Apply(hostpool, &version3)
	assert.Error(t, err,
		"Expected an error applying a hosts pool with duplicate names")

	// Check the new definition wasn't applied after this error
	hosts1, warnings, version4, err := cm.List()
	require.NoError(t, err, "Unexpected error getting list of hosts in pool")
	assert.Len(t, warnings, 0)
	assert.Len(t, hosts1, 4)
	assert.Contains(t, hosts, oldName,
		"Hosts Pool unexpectedly changed after an apply error using duplicates")
	assert.Equal(t, version3, version4, "Expected no version change after duplicate error")

	// Error case : attempt to delete an allocated host
	hostpool = hostpool[:1]
	err = cm.Apply(hostpool, &version4)
	assert.Error(t, err, "Expected an error deleting an allocated host")
	hosts2, warnings, version, err := cm.List()
	require.NoError(t, err, "Unexpected error getting list of hosts in pool")
	assert.Len(t, warnings, 0)
	assert.Len(t, hosts, 4)
	assert.Equal(t, hosts1, hosts2, "Expected no change in hosts pool after deletion error")
}
