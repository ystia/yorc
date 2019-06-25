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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v3/config"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
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
			"slurm": {
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
	ret, err := retrieveJobID(str)
	require.Nil(t, err, "unexpected error")
	require.Equal(t, "4567", ret, "unexpected JobID parsing")
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

func TestParseJob(t *testing.T) {
	t.Parallel()
	data, err := os.Open("testdata/scontrol.txt")
	require.Nil(t, err, "unexpected error while opening test file")
	info, err := parseJobInfo(data)
	require.Nil(t, err, "unexpected error while parsing job info")
	require.Equal(t, "hpda19", info["NodeList"], "unexpected value for \"NodeList\" key")
	require.Equal(t, "/home_nfs/john/file.out", info["StdOut"], "unexpected value for \"StdOut\" key")
	require.Equal(t, "None", info["Reason"], "unexpected value for \"Reason\" key")
	require.Equal(t, "/home_nfs/john/file.err", info["StdErr"], "unexpected value for \"StdErr\" key")
	require.Equal(t, "RUNNING", info["JobState"], "unexpected value for \"JobState\" key")
	require.Equal(t, "test-salloc-Environment", info["JobName"], "unexpected value for \"JobName\" key")
	require.Equal(t, "2-19:42:53", info["RunTime"], "unexpected value for \"RunTime\" key")
}

func TestGetJobInfoWithInvalidJob(t *testing.T) {
	t.Parallel()
	s := &MockSSHClient{
		MockRunCommand: func(cmd string) (string, error) {
			return "slurm_load_jobs error: Invalid job id specified", errors.New("")
		},
	}
	info, err := getJobInfo(s, "1234")
	require.Nil(t, info, "info should be nil")

	require.Equal(t, true, isNoJobFoundError(err), "expected no job found error")
}

func TestGetJobInfo(t *testing.T) {
	t.Parallel()
	s := &MockSSHClient{
		MockRunCommand: func(cmd string) (string, error) {
			content, err := ioutil.ReadFile("testdata/scontrol.txt")
			require.Nil(t, err, "Unexpected error reading scontrol")
			return string(content), nil
		},
	}

	data, err := os.Open("testdata/scontrol.txt")
	require.Nil(t, err, "unexpected error while opening test file")
	expected, err := parseJobInfo(data)
	require.Nil(t, err, "Unexpected error parsing job info")
	info, err := getJobInfo(s, "1234")
	require.Nil(t, err, "Unexpected error retrieving job info")
	require.NotNil(t, info, "info should not be nil")

	require.Equal(t, expected, info, "unexpected job info")
}

func TestGetJobInfoWithEmptyResponse(t *testing.T) {
	t.Parallel()
	s := &MockSSHClient{
		MockRunCommand: func(cmd string) (string, error) {
			return "", nil
		},
	}
	info, err := getJobInfo(s, "1234")
	require.Nil(t, info, "info should be nil")

	require.Equal(t, true, isNoJobFoundError(err), "expected no job found error")
}

func TestGetJobInfoWithError(t *testing.T) {
	t.Parallel()
	s := &MockSSHClient{
		MockRunCommand: func(cmd string) (string, error) {
			return "oups, it's bad", errors.New("this is an error !")
		},
	}
	info, err := getJobInfo(s, "1234")
	require.Nil(t, info, "info should be nil")

	require.Equal(t, "oups, it's bad: this is an error !", err.Error(), "expected error")
}

func TestToSlurmMemFormat(t *testing.T) {
	t.Parallel()
	type args struct {
		memStr string
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"TestMemInGB", args{"250 GB"}, "244140625K", false},
		{"TestMemInMiB", args{"2010 MiB"}, "2058240K", false},
		{"TestMemInMiB", args{"800 MiB"}, "819200K", false},
		{"TestMemInGiBWithDecimal", args{"0.5 GiB"}, "524288K", false},
		{"TestMemInGBWithDecimal", args{"0.5GB"}, "488281K", false},
		{"TestBadFormat", args{"0.5 Bad"}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memSlurm, err := toSlurmMemFormat(tt.args.memStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("toSlurmMemFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, memSlurm, "unexpected slurm mem format")
		})
	}

}
