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
	"bytes"
	"fmt"
	"github.com/bramvdbogaerde/go-scp"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/log"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/context"
	"io"
	"path"
)

// Client is interface allowing running command
type Client interface {
	RunCommand(string) (string, error)
}

// SSHSessionWrapper is a wrapper with a piped SSH session
type SSHSessionWrapper struct {
	Session *ssh.Session
	Stdout  io.Reader
	Stderr  io.Reader
}

// SSHClient is a client SSH
type SSHClient struct {
	Config *ssh.ClientConfig
	Host   string
	Port   int
}

// GetSessionWrapper allows to return a session wrapper in order to handle stdout/stderr for running long synchronous commands
func (client *SSHClient) GetSessionWrapper() (*SSHSessionWrapper, error) {
	var ps = &SSHSessionWrapper{}
	var err error
	ps.Session, err = client.newSession()
	if err != nil {
		return nil, errors.Wrap(err, "Unable to prepare SSH command")
	}

	log.Debug("[SSHSession] Add Stderr/Stdout pipelines")
	ps.Stdout, err = ps.Session.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "Unable to setup stdout for session")
	}

	ps.Stderr, err = ps.Session.StderrPipe()
	if err != nil {
		return nil, errors.Wrap(err, "Unable to setup stderr for session")
	}

	return ps, nil
}

// RunCommand allows to run a specified command
func (client *SSHClient) RunCommand(cmd string) (string, error) {
	session, err := client.newSession()
	if err != nil {
		return "", errors.Wrap(err, "Unable to setup stdout for session")
	}
	defer session.Close()
	var b bytes.Buffer
	session.Stderr = &b
	session.Stdout = &b

	log.Debugf("[SSHSession] %q", cmd)
	err = session.Run(cmd)
	return b.String(), err
}

func (client *SSHClient) newSession() (*ssh.Session, error) {
	connection, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", client.Host, client.Port), client.Config)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to open SSH connection")
	}

	session, err := connection.NewSession()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create session")
	}

	return session, nil
}

// RunCommand allows to run a specified command from a session wrapper in order to handle stdout/stderr during long synchronous commands
func (sw *SSHSessionWrapper) RunCommand(ctx context.Context, cmd string) error {
	chClosed := make(chan struct{})
	defer func() {
		sw.Session.Close()
		close(chClosed)
	}()
	log.Debugf("[SSHSession] running command: %q", cmd)
	go func() {
		select {
		case <-ctx.Done():
			log.Debug("[SSHSession] Cancellation has been sent: a sigkill signal is sent to remote process")
			sw.Session.Signal(ssh.SIGKILL)
			sw.Session.Close()
			return
		case <-chClosed:
			return
		}
	}()

	err := sw.Session.Run(cmd)
	return err
}

// CopyFile allows to copy a reader over SSH with defined remote path and specific permissions
func (client *SSHClient) CopyFile(source io.Reader, remotePath, permissions string) error {
	// Create a new SCP client
	scpHostPort := fmt.Sprintf("%s:%d", client.Host, client.Port)
	scpClient := scp.NewClient(scpHostPort, client.Config)

	// Connect to the remote server
	err := scpClient.Connect()
	if err != nil {
		return errors.Wrapf(err, "Couldn't establish a connection to the remote host:%q", scpHostPort)
	}
	defer scpClient.Session.Close()

	// Create the remote directory
	remoteDir := path.Dir(remotePath)
	mkdirCmd := fmt.Sprintf("mkdir -p %s", remoteDir)
	_, err = client.RunCommand(mkdirCmd)
	if err != nil {
		return errors.Wrapf(err, "Couldn't create the remote directory:%q", remoteDir)
	}

	// Finally, copy the reader over SSH
	log.Debugf("Copy source over SSH to remote path:%s", remotePath)
	scpClient.CopyFile(source, remotePath, permissions)
	return nil
}
