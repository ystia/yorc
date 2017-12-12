package slurm

import (
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
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

func TestGetAttribute(t *testing.T) {
	t.Parallel()
	s := &MockSSHClient{
		MockRunCommand: func(cmd string) (string, error) {
			return "CUDA_VISIBLE_DEVICES=NoDevFiles", nil
		},
	}
	value, err := getAttribute(s, "cuda_visible_devices", "1234", "myNodeName")
	require.Nil(t, err)
	require.Equal(t, "NoDevFiles", value)
}

func TestGetAttributeWithUnknownKey(t *testing.T) {
	t.Parallel()
	s := &MockSSHClient{}
	value, err := getAttribute(s, "unknown_key", "1234", "myNodeName")
	require.Equal(t, "", value)
	require.Error(t, err, "unknown key error expected")
}

func TestGetAttributeWithFailure(t *testing.T) {
	t.Parallel()
	s := &MockSSHClient{
		MockRunCommand: func(cmd string) (string, error) {
			return "", errors.New("expected failure")
		},
	}
	value, err := getAttribute(s, "unknown_key", "1234", "myNodeName")
	require.Equal(t, "", value)
	require.Error(t, err, "expected failure expected")
}

func TestGetAttributeWithMalformedStdout(t *testing.T) {
	t.Parallel()
	s := &MockSSHClient{
		MockRunCommand: func(cmd string) (string, error) {
			return "MALFORMED_VALUE", nil
		},
	}
	value, err := getAttribute(s, "unknown_key", "1234", "myNodeName")
	require.Equal(t, "", value)
	require.Error(t, err, "expected property/value is malformed")
}
