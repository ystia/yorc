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

package sshutil

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
	"gotest.tools/v3/assert"

	"github.com/ystia/yorc/v4/log"
)

func TestSSHAgent(t *testing.T) {
	log.SetDebug(true)

	// First generate a valid private key content
	priv, err := rsa.GenerateKey(rand.Reader, 1024)
	bArray := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(priv)})
	privateKeyContent := string(bArray)

	// Create new ssh-agent
	sshAg, err := NewSSHAgent(context.Background())
	require.Nil(t, err, "unexpected error while creating SSH-agent")
	defer func() {
		err = sshAg.Stop()
		require.Nil(t, err, "unexpected error while stopping SSH-agent")
	}()

	// Add key to ssh-agent
	err = sshAg.AddKey(privateKeyContent, 3600)
	require.Nil(t, err, "unexpected error while adding key to SSH-agent")

	keys, err := sshAg.agent.List()
	require.Nil(t, err)
	require.Len(t, keys, 1, "expected one key")

	rawKey, err := ssh.ParseRawPrivateKey([]byte(privateKeyContent))
	require.Nil(t, err)
	signer, err := ssh.NewSignerFromKey(rawKey)
	require.Nil(t, err)
	require.Equal(t, signer.PublicKey().Marshal(), keys[0].Blob)

	// Remove key to ssh-agent
	err = sshAg.RemoveKey(privateKeyContent)
	require.Nil(t, err, "unexpected error while removing key for SSH-agent")

	keys, err = sshAg.agent.List()
	require.Nil(t, err)
	require.Len(t, keys, 0, "no key expected")
}

// BER SSH key is not handled by crypto/ssh
// https://github.com/golang/go/issues/14145
func TestReadPrivateKey(t *testing.T) {
	_, err := ReadPrivateKey("./testdata/ber_test.pem")
	require.NotNil(t, err)
}

func TestSSHClient_RunCommand(t *testing.T) {
	// generate a testing private key
	private, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Fatal("Failed to generate private key: ", err)
	}

	privateSigner, err := ssh.NewSignerFromKey(private)
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}

	var trackAttempts int
	type fields struct {
		clientConfig *ssh.ClientConfig
		RetryBackoff time.Duration
		MaxRetries   uint64
	}
	type testServerConfig struct {
		enableAuth bool
		ech        execCommandHandler

		pkc func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error)
	}
	type args struct {
		cmd string
	}
	tests := []struct {
		name             string
		fields           fields
		serverConfig     testServerConfig
		args             args
		want             string
		wantErr          bool
		expectedAttempts int
	}{

		{"SimpleAllOK", fields{
			clientConfig: &ssh.ClientConfig{
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			},
			MaxRetries: 10,
		}, testServerConfig{
			ech: func(s string) (string, uint32) {
				trackAttempts++
				return s, 0
			},
		}, args{"echo toto"}, "echo toto", false, 1},

		{"SimpleWithExecErrorNotRetried", fields{
			clientConfig: &ssh.ClientConfig{
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			},
			MaxRetries: 3,
		}, testServerConfig{
			ech: func(s string) (string, uint32) {
				trackAttempts++
				return "Error!", 1
			},
		}, args{"echo toto"}, "Error!", true, 1},

		{"SimpleWithLoginError", fields{
			clientConfig: &ssh.ClientConfig{
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				Auth: []ssh.AuthMethod{
					ssh.PublicKeys(privateSigner),
				},
			},
			MaxRetries: 2,
		}, testServerConfig{
			enableAuth: true,
			pkc: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
				trackAttempts++
				return nil, errors.New("Unauthorized")
			},
		}, args{"echo toto"}, "", true, 3},

		{"SimpleLoginErrorRetriedThenOK", fields{
			clientConfig: &ssh.ClientConfig{
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				Auth: []ssh.AuthMethod{
					ssh.PublicKeys(privateSigner),
				},
			},
			MaxRetries: 10,
		}, testServerConfig{
			enableAuth: true,
			pkc: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
				trackAttempts++
				if trackAttempts == 1 {
					return nil, errors.New("Unauthorized")
				}
				return &ssh.Permissions{}, nil
			},
		}, args{"echo toto"}, "echo toto", false, 2},

		{"Timeout", fields{clientConfig: &ssh.ClientConfig{
			Timeout:         1 * time.Nanosecond,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}}, testServerConfig{}, args{"echo toto"}, "", true, 0},

		{"TimeoutNotReached", fields{clientConfig: &ssh.ClientConfig{
			Timeout:         2 * time.Second,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}}, testServerConfig{}, args{"echo toto"}, "echo toto", false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trackAttempts = 0
			ctx, cf := context.WithCancel(context.Background())
			defer cf()
			addr := newServer(ctx, func(s *serverConfig) {
				s.NoClientAuth = !tt.serverConfig.enableAuth
				if tt.serverConfig.ech != nil {
					s.execCommandHandler = tt.serverConfig.ech
				}
				if tt.serverConfig.pkc != nil {
					s.PublicKeyCallback = tt.serverConfig.pkc
				}
			})
			hostPort := strings.Split(addr.String(), ":")
			port, err := strconv.Atoi(hostPort[1])
			assert.NilError(t, err)
			client := &SSHClient{
				Config:       tt.fields.clientConfig,
				Host:         hostPort[0],
				Port:         port,
				MaxRetries:   tt.fields.MaxRetries,
				RetryBackoff: tt.fields.RetryBackoff,
			}
			got, err := client.RunCommand(tt.args.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("SSHClient.RunCommand() error = %+v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SSHClient.RunCommand() = %v, want %v", got, tt.want)
			}
			assert.Equal(t, trackAttempts, tt.expectedAttempts)
		})
	}
}

func TestSSHSessionWrapper_RunCommand(t *testing.T) {
	// generate a testing private key
	private, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Fatal("Failed to generate private key: ", err)
	}

	privateSigner, err := ssh.NewSignerFromKey(private)
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}

	var trackAttempts int
	type fields struct {
		clientConfig *ssh.ClientConfig
		RetryBackoff time.Duration
		MaxRetries   uint64
	}
	type testServerConfig struct {
		enableAuth bool
		ech        execCommandHandler

		pkc func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error)
	}
	type args struct {
		cmd string
	}
	tests := []struct {
		name             string
		fields           fields
		serverConfig     testServerConfig
		args             args
		want             string
		wantConnErr      bool
		wantErr          bool
		expectedAttempts int
	}{

		{"SimpleAllOK", fields{
			clientConfig: &ssh.ClientConfig{
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			},
			MaxRetries: 10,
		}, testServerConfig{
			ech: func(s string) (string, uint32) {
				trackAttempts++
				return s, 0
			},
		}, args{"echo toto"}, "echo toto", false, false, 1},

		{"SimpleWithExecErrorNotRetried", fields{
			clientConfig: &ssh.ClientConfig{
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			},
			MaxRetries: 3,
		}, testServerConfig{
			ech: func(s string) (string, uint32) {
				trackAttempts++
				return "Error!", 1
			},
		}, args{"echo toto"}, "Error!", false, true, 1},

		{"SimpleWithLoginError", fields{
			clientConfig: &ssh.ClientConfig{
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				Auth: []ssh.AuthMethod{
					ssh.PublicKeys(privateSigner),
				},
			},
			MaxRetries: 2,
		}, testServerConfig{
			enableAuth: true,
			pkc: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
				trackAttempts++
				return nil, errors.New("Unauthorized")
			},
		}, args{"echo toto"}, "", true, true, 3},

		{"SimpleLoginErrorRetriedThenOK", fields{
			clientConfig: &ssh.ClientConfig{
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				Auth: []ssh.AuthMethod{
					ssh.PublicKeys(privateSigner),
				},
			},
			MaxRetries: 10,
		}, testServerConfig{
			enableAuth: true,
			pkc: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
				trackAttempts++
				if trackAttempts == 1 {
					return nil, errors.New("Unauthorized")
				}
				return &ssh.Permissions{}, nil
			},
		}, args{"echo toto"}, "echo toto", false, false, 2},

		{"Timeout", fields{clientConfig: &ssh.ClientConfig{
			Timeout:         1 * time.Nanosecond,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}}, testServerConfig{}, args{"echo toto"}, "", true, false, 0},

		{"TimeoutNotReached", fields{clientConfig: &ssh.ClientConfig{
			Timeout:         2 * time.Second,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}}, testServerConfig{}, args{"echo toto"}, "echo toto", false, false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trackAttempts = 0
			ctx, cf := context.WithCancel(context.Background())
			defer cf()
			addr := newServer(ctx, func(s *serverConfig) {
				s.NoClientAuth = !tt.serverConfig.enableAuth
				if tt.serverConfig.ech != nil {
					s.execCommandHandler = tt.serverConfig.ech
				}
				if tt.serverConfig.pkc != nil {
					s.PublicKeyCallback = tt.serverConfig.pkc
				}
			})
			hostPort := strings.Split(addr.String(), ":")
			port, err := strconv.Atoi(hostPort[1])
			assert.NilError(t, err)
			client := &SSHClient{
				Config:       tt.fields.clientConfig,
				Host:         hostPort[0],
				Port:         port,
				MaxRetries:   tt.fields.MaxRetries,
				RetryBackoff: tt.fields.RetryBackoff,
			}

			session, err := client.GetSessionWrapper()
			if (err != nil) != tt.wantConnErr {
				t.Errorf("SSHSessionWrapper.RunCommand() error = %+v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				assert.Equal(t, trackAttempts, tt.expectedAttempts)
				return
			}

			err = session.RunCommand(context.Background(), tt.args.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("SSHSessionWrapper.RunCommand() error = %+v, wantErr %v", err, tt.wantErr)
				return
			}
			b, err := ioutil.ReadAll(session.Stdout)
			assert.NilError(t, err)
			got := string(b)

			if got != tt.want {
				t.Errorf("SSHSessionWrapperRunCommand() = %v, want %v", got, tt.want)
			}

			assert.Equal(t, trackAttempts, tt.expectedAttempts)
		})
	}
}
