// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/ystia/yorc/v4/log"
	"golang.org/x/crypto/ssh"
)

type execCommandHandler func(string) (string, uint32)

func defaultExecCommandHandler(command string) (string, uint32) {
	return command, 0
}

type serverConfig struct {
	*ssh.ServerConfig
	execCommandHandler execCommandHandler
}

type sshServerConfigCallback func(*serverConfig)

func newServer(ctx context.Context, cb sshServerConfigCallback) net.Addr {

	// An SSH server is represented by a ServerConfig, which holds
	// certificate details and handles authentication of ServerConns.
	config := &ssh.ServerConfig{
		// Remove to disable password auth.
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// Should use constant-time compare (or better, salt+hash) in
			// a production setting.
			if c.User() == "testuser" && string(pass) == "tiger" {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},

		// Remove to disable public key auth.
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			return &ssh.Permissions{}, nil

		},
	}

	config.SetDefaults()

	private, err := rsa.GenerateKey(config.Rand, 4096)
	if err != nil {
		log.Fatal("Failed to generate private key: ", err)
	}

	privateSigner, err := ssh.NewSignerFromKey(private)
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}

	config.AddHostKey(privateSigner)

	serverConfig := &serverConfig{
		ServerConfig:       config,
		execCommandHandler: defaultExecCommandHandler,
	}

	if cb != nil {
		cb(serverConfig)
	}

	// Once a ServerConfig has been configured, connections can be
	// accepted.

	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", "127.0.0.1:")
	if err != nil {
		log.Fatal("failed to listen for connection: ", err)
	}
	go func() {
		for {
			nConn, err := listener.Accept()
			if err != nil {
				log.Print("failed to accept incoming connection: ", err)
				continue
			}

			// Before use, a handshake must be performed on the incoming
			// net.Conn.
			conn, chans, reqs, err := ssh.NewServerConn(nConn, config)
			if err != nil {
				log.Print("failed to handshake: ", err)
				continue
			}

			// The incoming Request channel must be serviced.
			go ssh.DiscardRequests(reqs)

			go func() {
				defer conn.Close()

				// Service the incoming Channel channel.
				for {
					var newChannel ssh.NewChannel
					select {
					case newChannel = <-chans:
					case <-ctx.Done():
						return
					}
					// Channels have a type, depending on the application level
					// protocol intended. In the case of a shell, the type is
					// "session" and ServerShell may be used to present a simple
					// terminal interface.
					if newChannel.ChannelType() != "session" {
						newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
						continue
					}
					channel, requests, err := newChannel.Accept()
					if err != nil {
						log.Printf("Could not accept channel: %v", err)
						return
					}

					// Sessions have out-of-band requests such as "shell",
					// "pty-req" and "env".  Here we handle only the
					// "shell" request.
					go func(in <-chan *ssh.Request) {
						for req := range in {
							switch req.Type {
							case "exec":

								req.Reply(true, nil)
								var payload = struct{ Command string }{}

								ssh.Unmarshal(req.Payload, &payload)

								result, status := serverConfig.execCommandHandler(payload.Command)
								channel.Write([]byte(result))

								b := make([]byte, 4)
								binary.BigEndian.PutUint32(b, status)
								channel.SendRequest("exit-status", false, b)
								channel.CloseWrite()
								channel.Close()

							default:
								req.Reply(false, nil)
							}
						}
					}(requests)

				}

			}()
		}
	}()
	return listener.Addr()
}
