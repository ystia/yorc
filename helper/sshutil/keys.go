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
	"io/ioutil"
	"os"

	"github.com/hashicorp/consul/api"
	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/deployments"
	"github.com/ystia/yorc/v3/tosca/datatypes"
)

// DefaultSSHPrivateKeyFilePath is the default SSH private Key file path
// used to connect to provisioned resources
const DefaultSSHPrivateKeyFilePath = "~/.ssh/yorc.pem"

// PrivateKey represent a parsed ssh Private Key.
// Content is always set but Path is populated only if the key content was read from a filesystem path (not provided directly)
type PrivateKey struct {
	Content []byte
	Path    string
}

// ReadPrivateKey returns an authentication method relying on private/public key pairs
// The argument is :
// - either a path to the private key file,
// - or the content or this private key file
func ReadPrivateKey(pk string) (ssh.AuthMethod, error) {
	raw, err := ToPrivateKeyContent(pk)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(raw)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to parse key file %q", pk)
	}
	return ssh.PublicKeys(signer), nil
}

// ReadSSHPrivateKey returns an authentication method relying on private/public key pairs
func ReadSSHPrivateKey(pk *PrivateKey) (ssh.AuthMethod, error) {
	signer, err := ssh.ParsePrivateKey(pk.Content)
	if err != nil {
		p := pk.Path
		if p == "" {
			p = "<private key content redacted>"
		}
		return nil, errors.Wrapf(err, "Failed to parse key file %q", p)
	}
	return ssh.PublicKeys(signer), nil
}

// ToPrivateKeyContent allows to convert private key content or file to byte array
func ToPrivateKeyContent(pk string) ([]byte, error) {
	var p []byte
	// check if pk is a path
	keyPath, err := homedir.Expand(pk)
	if err != nil {
		return nil, errors.Wrap(err, "failed to expand key path")
	}
	if _, err := os.Stat(keyPath); err == nil {
		p, err = ioutil.ReadFile(keyPath)
		if err != nil {
			p = []byte(pk)
		}
	} else {
		p = []byte(pk)
	}
	return p, nil
}

func isPrivateKey(key []byte) bool {
	_, err := ssh.ParsePrivateKey(key)
	return err == nil
}

func isFilePath(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetPrivateKey returns a parsed PrivateKey
//
// The argument is :
// - either a path to the private key file,
// - or the content or this private key file
func GetPrivateKey(pathOrContent string) (*PrivateKey, error) {
	// By default consider that it is a key content
	k := &PrivateKey{Content: []byte(pathOrContent)}
	path, err := homedir.Expand(pathOrContent)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read private key file, error in fs home expansion")
	}
	if isFilePath(path) {
		// Well in fact it is a valid file path so read its content
		k.Path = path
		var err error
		k.Content, err = ioutil.ReadFile(k.Path)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read file")
		}
	}
	if !isPrivateKey(k.Content) {
		// not a valid key
		return nil, errors.New(`invalid key content`)
	}
	return k, nil
}

// GetDefaultKey returns Yorc's default private Key
func GetDefaultKey() (*PrivateKey, error) {
	k, err := GetPrivateKey(DefaultSSHPrivateKeyFilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse default Yorc private key %q", DefaultSSHPrivateKeyFilePath)
	}
	return k, nil
}

// GetKeysFromCredentialsDataType returns a map of PrivateKey contained in a Credential datatype
func GetKeysFromCredentialsDataType(creds *datatypes.Credential) (map[string]*PrivateKey, error) {
	keys := make(map[string]*PrivateKey, len(creds.Keys))
	for keyName, key := range creds.Keys {
		// If keys are comming from a config file (typically for hostspool) it may contain templates to interact with Vault
		key = config.DefaultConfigTemplateResolver.ResolveValueWithTemplates("credentials.key", key).(string)
		k, err := GetPrivateKey(key)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse key %q", keyName)
		}
		keys[keyName] = k
	}

	return keys, nil
}

// GetKeysFromCredentialsAttribute returns a map of PrivateKey by analysing a given credentials attribute of a given capability
func GetKeysFromCredentialsAttribute(kv *api.KV, deploymentID, nodeName, instanceID, capabilityName string) (map[string]*PrivateKey, error) {
	if instanceID == "" {
		instanceID = "0"
	}
	credentialsValue, err := deployments.GetInstanceCapabilityAttributeValue(kv, deploymentID, nodeName, instanceID, capabilityName, "credentials")
	if err != nil {
		return nil, err
	}
	if credentialsValue != nil && credentialsValue.RawString() != "" {
		creds := new(datatypes.Credential)
		err = mapstructure.Decode(credentialsValue.Value, creds)
		if err != nil {
			return nil, errors.Wrapf(err, `invalid credential datatype for attribute "credentials" for node %q, instance %q, capability %q`, nodeName, instanceID, capabilityName)
		}
		return GetKeysFromCredentialsDataType(creds)
	}
	return nil, nil
}

// SelectPrivateKeyOnName select a PrivateKey when several keys are available.
//
// This method is for backward compatibility when the ssh-agent is disable and only a single key can be used.
// The Selection algorithm is first to check a key named "0", then "yorc" and finally "default" if none of these
// are present then a random one is chosen.
//
// If shouldHavePath parameter is true then only keys having a valid file path can be returned (that's mean that
// keys provided only with their content are ignored)
//
// If there is no key or none of them matche the requirements then nil is returned
func SelectPrivateKeyOnName(keys map[string]*PrivateKey, shouldHavePath bool) *PrivateKey {
	if len(keys) > 1 {
		// For backward compatibility "0" was the default key name
		if key, ok := keys["0"]; ok {
			if !shouldHavePath || key.Path != "" {
				return key
			}
		}
		// Also support "yorc"
		if key, ok := keys["yorc"]; ok {
			if !shouldHavePath || key.Path != "" {
				return key
			}
		}
		// Also support "default"
		if key, ok := keys["default"]; ok {
			if !shouldHavePath || key.Path != "" {
				return key
			}
		}
	}
	// Pick a random or the only one
	for _, key := range keys {
		if !shouldHavePath || key.Path != "" {
			return key
		}
	}
	return nil
}
