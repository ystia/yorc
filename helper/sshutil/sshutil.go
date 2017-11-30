package sshutil

import (
	"bytes"
	"golang.org/x/crypto/ssh"
	"novaforge.bull.com/starlings-janus/janus/log"
)

// Session is interface allowing running command
type Session interface {
	RunCommand(string) (string, error)
}

// SSHSession allows to run command via SSH
type SSHSession struct {
	User, Password, URL, Port string
	session                   *ssh.Session
}

// NewSSHSession allows to create a new SSH session
func NewSSHSession(User, Password, URL, Port string) *SSHSession {
	s := new(SSHSession)
	s.User = User
	s.Password = Password
	s.URL = URL
	s.Port = Port

	return s
}

func (s *SSHSession) initSession() *ssh.Session {
	config := &ssh.ClientConfig{
		User: s.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.Password),
		},
	}
	client, err := ssh.Dial("tcp", s.URL+":"+s.Port, config)
	if err != nil {
		log.Fatal("Failed to dial: ", err)
	}

	session, err := client.NewSession()
	if err != nil {
		log.Fatal("Failed to create session: ", err)
	}

	return session
}

// RunCommand allows to run a command via SSH
func (s *SSHSession) RunCommand(cmd string) (string, error) {
	log.Debugf("[SSHSession] %q", cmd)
	session := s.initSession()
	defer session.Close()
	var b bytes.Buffer
	session.Stderr = &b
	session.Stdout = &b

	err := session.Run(cmd)

	return b.String(), err
}
