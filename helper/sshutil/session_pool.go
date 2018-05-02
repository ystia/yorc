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
	"fmt"
	"golang.org/x/crypto/ssh"
	"net"
	"strconv"
	"sync"
	"time"
)

type pool struct {
	// Timeout for Open (for both new and existing
	// connections). If Dial is not nil, it is up to the Dial func
	// to enforce the timeout for new connections.
	Timeout time.Duration

	tab map[string]*conn
	mu  sync.Mutex
}

// Open starts a new SSH session on the given server, reusing
// an existing connection if possible. If no connection exists,
// or if opening the session fails, Open attempts to dial a new
// connection. If dialing fails, Open returns the error from Dial.
func (p *pool) openSession(client *SSHClient) (*ssh.Session, error) {
	var deadline, sessionDeadline time.Time
	if p.Timeout > 0 {
		now := time.Now()
		deadline = now.Add(p.Timeout)

		// First time, use a NewSession deadline at half of the
		// overall timeout, to try to leave time for a subsequent
		// Dial and NewSession.
		sessionDeadline = now.Add(p.Timeout / 2)
	}

	addr := fmt.Sprintf("%s:%d", client.Host, client.Port)
	k := getUserKey(addr, client.Config)
	for {
		c := p.getConn(k, addr, client.Config, deadline)
		if c.err != nil {
			p.removeConn(k, c)
			return nil, c.err
		}
		s, err := c.newSession(sessionDeadline)
		if err == nil {
			return s, nil
		}
		sessionDeadline = deadline
		p.removeConn(k, c)
		c.c.Close()
		if p.Timeout > 0 && time.Now().After(deadline) {
			return nil, err
		}
	}
}

type conn struct {
	netC net.Conn
	c    *ssh.Client
	ok   chan bool
	err  error
}

func (c *conn) newSession(deadline time.Time) (*ssh.Session, error) {
	if !deadline.IsZero() {
		c.netC.SetDeadline(deadline)
		defer c.netC.SetDeadline(time.Time{})
	}
	return c.c.NewSession()
}

// getConn gets an ssh connection from the pool for key.
// If none is available, it dials anew.
func (p *pool) getConn(k, addr string, config *ssh.ClientConfig, deadline time.Time) *conn {
	p.mu.Lock()
	if p.tab == nil {
		p.tab = make(map[string]*conn)
	}
	c, ok := p.tab[k]
	if ok {
		p.mu.Unlock()
		<-c.ok
		return c
	}
	c = &conn{ok: make(chan bool)}
	p.tab[k] = c
	p.mu.Unlock()
	c.netC, c.c, c.err = p.dial("tcp", addr, config, deadline)
	close(c.ok)
	return c
}

// removeConn removes c1 from the pool if present.
func (p *pool) removeConn(k string, c1 *conn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	c, ok := p.tab[k]
	if ok && c == c1 {
		delete(p.tab, k)
	}
}

func (p *pool) dial(network, addr string, config *ssh.ClientConfig, deadline time.Time) (net.Conn, *ssh.Client, error) {
	dialer := net.Dialer{Deadline: deadline}
	dial := dialer.Dial
	netC, err := dial(network, addr)
	if err != nil {
		return nil, nil, err
	}
	conn, chans, reqs, err := ssh.NewClientConn(netC, addr, config)
	if err != nil {
		netC.Close()
		return nil, nil, err
	}
	sshC := ssh.NewClient(conn, chans, reqs)
	return netC, sshC, nil
}

func getUserKey(addr string, config *ssh.ClientConfig) string {
	return strconv.Quote(addr) + "-" + strconv.Quote(config.User)
}
