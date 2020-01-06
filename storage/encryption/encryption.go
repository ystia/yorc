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

package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"io"

	"github.com/pkg/errors"
)

// Encryptor allows to encrypt/Decrypt any date with AES tools
type Encryptor struct {
	Key string // encryption 32-bits key encoded in hexadecimal
	gcm cipher.AEAD
}

// NewEncryptor allows to instantiate a new encryptor with an existing key or a new one if no key is provided
func NewEncryptor(key string) (*Encryptor, error) {
	var err error
	if key == "" {
		key, err = buildEncryptionKey()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate encryption key")
		}
	}
	// Build GCM for all encryption/decryption
	decoded, err := hex.DecodeString(key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode hexadecimal key")
	}
	block, err := aes.NewCipher(decoded)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to instantiate new cipher block")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to instantiate new cipher GCM")
	}
	return &Encryptor{
		Key: key,
		gcm: gcm,
	}, nil
}

// Create 256-bits encryption key encoded in hexadecimal
func buildEncryptionKey() (string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

// Encrypt allows to encrypt data with the encryptor GCM
// encrypted data is prefixed with the nonce
func (e *Encryptor) Encrypt(data []byte) ([]byte, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, errors.Wrapf(err, "failed to read nonce from cipher GCM")
	}
	return e.gcm.Seal(nonce, nonce, data, nil), nil
}

// Decrypt allows to decrypt previously encrypted data
// We need to retrieve the nonce defined as the encrypted data prefix
func (e *Encryptor) Decrypt(data []byte) ([]byte, error) {
	nonceSize := e.gcm.NonceSize()
	nonce, encrypted := data[:nonceSize], data[nonceSize:]
	decrypted, err := e.gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decrypt data from cipher GCM")
	}
	return decrypted, nil
}
