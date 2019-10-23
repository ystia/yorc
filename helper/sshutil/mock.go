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

import "io"

// MockSSHClient allows to mock an SSH Client
type MockSSHClient struct {
	MockRunCommand func(string) (string, error)
	MockCopyFile   func(source io.Reader, remotePath string, permissions string) error
}

// RunCommand to mock a command ran via SSH
func (s *MockSSHClient) RunCommand(cmd string) (string, error) {
	if s.MockRunCommand != nil {
		return s.MockRunCommand(cmd)
	}
	return "", nil
}

// CopyFile to mock a file copy via SSH
func (s *MockSSHClient) CopyFile(source io.Reader, remotePath string, permissions string) error {
	if s.MockCopyFile != nil {
		return s.MockCopyFile(source, remotePath, permissions)
	}
	return nil
}
