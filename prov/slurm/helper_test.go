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

package slurm

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ystia/yorc/helper/sshutil"

	"os"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/config"
)

// MockSSHSession allows to mock an SSH session
type MockSSHClient struct {
	MockRunCommand func(string) (string, error)
}

// RunCommand to mock a command ran via SSH
func (s *MockSSHClient) RunCommand(cmd string) (string, error) {
	if s.MockRunCommand != nil {
		return s.MockRunCommand(cmd)
	}
	return "", nil
}

func TestGetAttributesWithCudaVisibleDeviceKey(t *testing.T) {
	t.Parallel()
	s := &MockSSHClient{
		MockRunCommand: func(cmd string) (string, error) {
			return "CUDA_VISIBLE_DEVICES=NoDevFiles", nil
		},
	}
	values, err := getAttributes(s, "cuda_visible_devices", "1234", "myNodeName")
	require.Nil(t, err)
	require.Len(t, values, 1, "values length not equal to 1")
	require.Equal(t, "NoDevFiles", values[0])
}

func TestGetAttributesWithNodePartitionKey(t *testing.T) {
	t.Parallel()
	s := &MockSSHClient{
		MockRunCommand: func(cmd string) (string, error) {
			return "node1,part1", nil
		},
	}
	values, err := getAttributes(s, "node_partition", "1234")
	require.Nil(t, err)
	require.Len(t, values, 2, "values length not equal to 2")
	require.Equal(t, "node1", values[0])
	require.Equal(t, "part1", values[1])
}

func TestGetAttributesWithNodePartitionKeyAndNotEnoughParameters(t *testing.T) {
	t.Parallel()
	s := &MockSSHClient{
		MockRunCommand: func(cmd string) (string, error) {
			return "node1,part1", nil
		},
	}
	_, err := getAttributes(s, "node_partition")
	require.Error(t, err, "expected not enough parameters error")
}

func TestGetAttributesWithNodePartitionKeyAndMalformedResponse(t *testing.T) {
	t.Parallel()
	s := &MockSSHClient{
		MockRunCommand: func(cmd string) (string, error) {
			return "a", nil
		},
	}
	_, err := getAttributes(s, "node_partition", "1234")
	require.Error(t, err, "expected unexpected stdout")
}

func TestGetAttributesWithUnknownKey(t *testing.T) {
	t.Parallel()
	s := &MockSSHClient{}
	_, err := getAttributes(s, "unknown_key", "1234")
	require.Error(t, err, "unknown key error expected")
}

func TestGetAttributesWithFailure(t *testing.T) {
	t.Parallel()
	s := &MockSSHClient{
		MockRunCommand: func(cmd string) (string, error) {
			return "", errors.New("expected failure")
		},
	}
	_, err := getAttributes(s, "unknown_key", "1234")
	require.Error(t, err, "expected failure expected")
}

func TestGetAttributesWithMalformedStdout(t *testing.T) {
	t.Parallel()
	s := &MockSSHClient{
		MockRunCommand: func(cmd string) (string, error) {
			return "MALFORMED_VALUE", nil
		},
	}
	_, err := getAttributes(s, "unknown_key", "1234")
	require.Error(t, err, "expected property/value is malformed")
}

// We test parsing the stderr line: ""
func TestParseSallocResponseWithEmpty(t *testing.T) {
	t.Parallel()
	str := ""
	chResult := make(chan allocationResponse)
	chErr := make(chan error)

	go parseSallocResponse(strings.NewReader(str), chResult, chErr)
	select {
	case <-chResult:
		require.Fail(t, "No response expected")
		return
	case err := <-chErr:
		require.Fail(t, "unexpected error", err.Error())
		return
	default:
		require.True(t, true)
	}
}

// We test parsing the stderr line: "salloc: Pending job allocation 1881"
func TestParseSallocResponseWithExpectedPending(t *testing.T) {
	t.Parallel()
	str := "salloc: Pending job allocation 1881\n"
	chResult := make(chan allocationResponse)
	chErr := make(chan error)

	var res allocationResponse

	go parseSallocResponse(strings.NewReader(str), chResult, chErr)
	select {
	case res = <-chResult:
		require.Equal(t, "1881", res.jobID)
		require.Equal(t, false, res.granted)
		return
	case err := <-chErr:
		require.Fail(t, "unexpected error", err.Error())
		return
	case <-time.After(1 * time.Second):
		require.Fail(t, "No response received")
	}
}

// salloc: Required node not available (down, drained or reserved)
// salloc: Pending job allocation 2220
// salloc: job 2220 queued and waiting for resources
func TestParseSallocResponseWithExpectedPendingInOtherThanFirstLine(t *testing.T) {
	t.Parallel()
	str := "salloc: Required node not available (down, drained or reserved)\nsalloc: Pending job allocation 2220\nsalloc: job 2220 queued and waiting for resources"
	chResult := make(chan allocationResponse)
	chErr := make(chan error)

	var res allocationResponse

	go parseSallocResponse(strings.NewReader(str), chResult, chErr)
	select {
	case res = <-chResult:
		require.Equal(t, "2220", res.jobID)
		require.Equal(t, false, res.granted)
		return
	case err := <-chErr:
		require.Fail(t, "unexpected error", err.Error())
		return
	case <-time.After(1 * time.Second):
		require.Fail(t, "No response received")
	}
}

// We test parsing the stdout line: "salloc: Granted job allocation 1881"
func TestParseSallocResponseWithExpectedGranted(t *testing.T) {
	t.Parallel()
	str := "salloc: Granted job allocation 1881\n"
	chResult := make(chan allocationResponse)
	chErr := make(chan error)

	var res allocationResponse

	go parseSallocResponse(strings.NewReader(str), chResult, chErr)
	select {
	case res = <-chResult:
		require.Equal(t, "1881", res.jobID)
		require.Equal(t, true, res.granted)
		return
	case err := <-chErr:
		require.Fail(t, "unexpected error", err.Error())
		return
	case <-time.After(1 * time.Second):
		require.Fail(t, "No response received")
	}
}

// We test parsing the stderr lines:
// "salloc: Job allocation 1882 has been revoked."
// "salloc: error: CPU count per node can not be satisfied"
// "salloc: error: Job submit/allocate failed: Requested node configuration is not available"
func TestParseSallocResponseWithExpectedRevokedAllocation(t *testing.T) {
	t.Parallel()
	str := "salloc: Job allocation 1882 has been revoked.\nsalloc: error: CPU count per node can not be satisfied\nsalloc: error: Job submit/allocate failed: Requested node configuration is not available"
	chResult := make(chan allocationResponse)
	chErr := make(chan error)

	go parseSallocResponse(strings.NewReader(str), chResult, chErr)
	select {
	case <-chResult:
		require.Fail(t, "No expected response")
		return
	case err := <-chErr:
		require.Error(t, err)
		return
	case <-time.After(1 * time.Second):
		require.Fail(t, "No response received")
	}
}

// Tests the definition of a private key in configuration
func TestPrivateKey(t *testing.T) {
	t.Parallel()
	// First generate a valid private key content
	priv, err := rsa.GenerateKey(rand.Reader, 1024)
	bArray := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(priv)})
	privateKeyContent := string(bArray)

	// Config to test
	cfg := config.Configuration{
		Infrastructures: map[string]config.DynamicMap{
			"slurm": config.DynamicMap{
				"user_name":   "jdoe",
				"url":         "127.0.0.1",
				"port":        22,
				"private_key": privateKeyContent}},
	}

	err = checkInfraUserConfig(cfg)
	assert.NoError(t, err, "Unexpected error parsing a configuration with private key")
	_, err = getSSHClient("", "", "", cfg)
	assert.NoError(t, err, "Unexpected error getting a ssh client using a configuration with private key")
	_, err = getSSHClient("jdoe", privateKeyContent, "", cfg)
	assert.NoError(t, err, "Unexpected error getting a ssh client using provided properties with private key")

	// Remove the private key.
	// As there is no password defined either, check an error is returned
	cfg.Infrastructures["slurm"].Set("private_key", "")
	err = checkInfraUserConfig(cfg)
	assert.Error(t, err, "Expected an error parsing a wrong configuration with no private key and no password defined")
	_, err = getSSHClient("", "", "", cfg)
	assert.Error(t, err, "Expected an error getting a ssh client using a configuration with no private key and no password defined")
	_, err = getSSHClient("jdoe", "", "", cfg)
	assert.Error(t, err, "Expected an error getting a ssh client using a provided user name property but no private key and no password provided")

	// Setting a wrong private key path
	// Check the attempt to use this key for the authentication method is failing
	cfg.Infrastructures["slurm"].Set("private_key", "invalid_path_to_key.pem")
	err = checkInfraUserConfig(cfg)
	assert.NoError(t, err, "Unexpected error parsing a configuration with private key")
	_, err = getSSHClient("", "", "", cfg)
	assert.Error(t, err, "Expected an error getting a ssh client using a configuration with bad private key and no password defined")
	_, err = getSSHClient("jdo", "invalid_path_to_key.pem", "", cfg)
	assert.Error(t, err, "Expected an error getting a ssh client using provided credentials with bad private key and no password defined")

	// Slurm Configuration with no private key but a password, the config should be valid
	cfg.Infrastructures["slurm"] = config.DynamicMap{
		"user_name": "jdoe",
		"url":       "127.0.0.1",
		"port":      22,
		"password":  "test",
	}

	err = checkInfraUserConfig(cfg)
	assert.NoError(t, err, "Unexpected error parsing a configuration with password")
	_, err = getSSHClient("", "", "", cfg)
	assert.NoError(t, err, "Unexpected error getting a ssh client using a configuration with password")
	_, err = getSSHClient("jdoe", "", "test", cfg)
	assert.NoError(t, err, "Unexpected error getting a ssh client using provided credentials with password")
}

func TestParseJobIDFromSbatchOut(t *testing.T) {
	t.Parallel()
	str := "Submitted batch job 4567"
	ret, err := parseJobIDFromBatchOutput(str)
	require.Nil(t, err, "unexpected error")
	require.Equal(t, "4567", ret, "unexpected JobID parsing")
}

func TestParseOutputConfigFromBatchScriptWithAll(t *testing.T) {
	t.Parallel()
	expected := []string{"c.out", "file", "b.out"}
	data, err := os.Open("testdata/submit.sh")
	require.Nil(t, err, "unexpected error while opening test file")
	outputParams, err := parseOutputConfigFromBatchScript(data, true)
	require.Nil(t, err, "unexpected error while parsing output params from test file")
	require.Equal(t, expected, outputParams)
}

func TestParseOutputConfigFromBatchScript(t *testing.T) {
	t.Parallel()
	expected := []string{"file", "b.out"}
	data, err := os.Open("testdata/submit.sh")
	require.Nil(t, err, "unexpected error while opening test file")
	outputParams, err := parseOutputConfigFromBatchScript(data, false)
	require.Nil(t, err, "unexpected error while parsing output params from test file")
	require.Equal(t, expected, outputParams)
}

func TestParseOutputConfigFromOpts(t *testing.T) {
	t.Parallel()
	expected := []string{"b.out"}
	data := []string{"--output=b.out"}
	outputParams := parseOutputConfigFromOpts(data)
	require.Equal(t, expected, outputParams)
}

func TestParseKeyValue(t *testing.T) {
	t.Parallel()
	type args struct {
		str string
	}
	type checks struct {
		is    bool
		key   string
		value string
	}

	tests := []struct {
		name string
		args args
		want checks
	}{
		{"TestKeyValueSimple", args{"aaa=bbb"}, checks{true, "aaa", "bbb"}},
		{"TestNoKeyValue", args{"azerty"}, checks{false, "", ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is, k, v := parseKeyValue(tt.args.str)
			assert.Equal(t, tt.want.is, is)
			assert.Equal(t, tt.want.key, k)
			assert.Equal(t, tt.want.value, v)
		})
	}

}

func TestGetJobInfo(t *testing.T) {
	t.Parallel()
	type args struct {
		sshCli  sshutil.Client
		jobID   string
		jobName string
	}

	tests := []struct {
		name    string
		args    args
		want    *jobInfoShort
		wantErr bool
		err     error
	}{
		{"TestWithJobID", args{&MockSSHClient{
			MockRunCommand: func(cmd string) (string, error) {
				return "my_test,123,RUNNING", nil
			}}, "123", ""}, &jobInfoShort{ID: "123", name: "my_test", state: "RUNNING"}, false, nil},
		{"TestWithJobName", args{&MockSSHClient{
			MockRunCommand: func(cmd string) (string, error) {
				return "my_test,123,RUNNING", nil
			}}, "", "my_test"}, &jobInfoShort{ID: "123", name: "my_test", state: "RUNNING"}, false, nil},
		{"TestWithoutParams", args{&MockSSHClient{
			MockRunCommand: func(cmd string) (string, error) {
				return "", nil
			}}, "", ""}, nil, true, nil},
		{"TestWithMalformedOutput", args{&MockSSHClient{
			MockRunCommand: func(cmd string) (string, error) {
				return "MALFORMED", nil
			}}, "123", ""}, nil, true, nil},
		{"TestWithJobNotFound", args{&MockSSHClient{
			MockRunCommand: func(cmd string) (string, error) {
				return "", nil
			}}, "123", ""}, nil, true, &noJobFound{msg: fmt.Sprintf("no information found for job with id:%q, name:%q", "123", "")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := getJobInfo(tt.args.sshCli, tt.args.jobID, tt.args.jobName)
			if (err != nil) != tt.wantErr {
				t.Errorf("TestGetJobInfo() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.err != nil && !reflect.DeepEqual(err, tt.err) {
				t.Errorf("TestGetJobInfo() error = %v, expected err:%v", err, tt.err)
			}
			if !reflect.DeepEqual(info, tt.want) {
				t.Fatalf("TestGetJobInfo() = %v, want %v", info, tt.want)
			}
		})
	}
}
