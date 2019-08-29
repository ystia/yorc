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
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/ystia/yorc/v3/log"
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
