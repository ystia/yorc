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
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/armon/go-metrics"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"

	"github.com/ystia/yorc/helper/metricsutil"
)

type pool struct {
	tab map[string]*conn
	mu  sync.Mutex
}

// Open starts a new SSH session on the given server, reusing
// an existing connection if possible. If no connection exists,
// or if opening the session fails, Open attempts to dial a new
// connection. If dialing fails, Open returns the error from Dial.
func (p *pool) openSession(client *SSHClient) (*sshSession, error) {
	addr := fmt.Sprintf("%s:%d", client.Host, client.Port)
	k := getUserKey(addr, client.Config)
	for {
		// The algorithm here is to get a connection by reusing existing if any
		// Then if we open a session. Sometimes open fails due to too many sessions open
		// in this case we remove the connection from cache and teardown it (wait for other sessions
		// to be closed and finally close the underlying connection) on next try a new connection is created.
		c := p.getConn(k, addr, client.Config)
		if c.err != nil {
			// Can't open connection stop here
			p.removeConn(k, c)
			return nil, c.err
		}
		s, err := c.newSession()
		if err == nil {
			return s, nil
		}
		metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"ssh-connections-pool", c.name, "sessions", "open-failed"}), 1)
		// can't open session this is probably due to too many session open
		// remove this connection from cache
		p.removeConn(k, c)
		// Gracefully teardown it wait for all sessions to finish and then close
		// the connection. Teardown is asynchronous/non-blocking
		c.teardown()
	}
}

type sshSession struct {
	*ssh.Session

	conn *conn
}

func (s *sshSession) Close() error {
	err := s.Session.Close()
	s.conn.lockSessionsCount.Lock()
	defer s.conn.lockSessionsCount.Unlock()
	s.conn.opennedSessions--
	metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"ssh-connections-pool", s.conn.name, "sessions", "closes"}), 1)
	metrics.SetGauge(metricsutil.CleanupMetricKey([]string{"ssh-connections-pool", s.conn.name, "sessions", "open"}), float32(s.conn.opennedSessions))
	return errors.Wrap(err, "failed to close ssh session")
}

type conn struct {
	name              string
	netC              net.Conn
	c                 *ssh.Client
	ok                chan bool
	err               error
	lockSessionsCount sync.Mutex
	opennedSessions   uint
}

// closes the ssh client
func (c *conn) close() {
	c.c.Close()
	metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"ssh-connections-pool", "closes", c.name}), 1)
}

// asynchronously wait for all openned sessions to finish and then close the connection.
// Actually it closes the ssh client which in turn closes the network connection.
func (c *conn) teardown() {
	go func(c *conn) {

		for {
			<-time.After(5 * time.Second)
			c.lockSessionsCount.Lock()
			if c.opennedSessions == 0 {
				c.close()
				c.lockSessionsCount.Unlock()
				return
			}
			c.lockSessionsCount.Unlock()
		}
	}(c)
}

func (c *conn) newSession() (*sshSession, error) {
	ss, err := c.c.NewSession()
	if err != nil {
		return nil, errors.Wrap(err, "failed to open session")
	}
	c.lockSessionsCount.Lock()
	defer c.lockSessionsCount.Unlock()
	c.opennedSessions++
	metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"ssh-connections-pool", c.name, "sessions", "creations"}), 1)
	metrics.SetGauge(metricsutil.CleanupMetricKey([]string{"ssh-connections-pool", c.name, "sessions", "open"}), float32(c.opennedSessions))

	return &sshSession{Session: ss, conn: c}, nil
}

// getConn gets an ssh connection from the pool for key.
// If none is available, it dials anew.
func (p *pool) getConn(k, addr string, config *ssh.ClientConfig) *conn {
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
	c.netC, c.c, c.err = p.dial("tcp", addr, config)
	c.name = fmt.Sprintf("%s-%s-%x", addr, config.User, c.c.SessionID())
	metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"ssh-connections-pool", "creations", c.name}), 1)
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

func (p *pool) dial(network, addr string, config *ssh.ClientConfig) (net.Conn, *ssh.Client, error) {
	dialer := net.Dialer{}
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
