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
	"errors"
	"io"
	"testing"
)

func TestMockSSHClient_RunCommand(t *testing.T) {

	type args struct {
		cmd string
	}
	tests := []struct {
		name           string
		MockRunCommand func(string) (string, error)
		args           args
		want           string
		wantErr        bool
	}{
		{"MockWithNilFn", nil, args{""}, "", false},
		{"MockWithFn", func(cmd string) (string, error) {
			if cmd == "fail" {
				return "", errors.New("expected error")
			}
			return "ok", nil
		}, args{"fail"}, "", true},
		{"MockWithFn", func(cmd string) (string, error) {
			if cmd == "fail" {
				return "", errors.New("expected error")
			}
			return "ok", nil
		}, args{""}, "ok", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MockSSHClient{
				MockRunCommand: tt.MockRunCommand,
			}
			got, err := s.RunCommand(tt.args.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("MockSSHClient.RunCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MockSSHClient.RunCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMockSSHClient_CopyFile(t *testing.T) {
	type args struct {
		source      io.Reader
		remotePath  string
		permissions string
	}
	tests := []struct {
		name         string
		MockCopyFile func(source io.Reader, remotePath string, permissions string) error
		args         args
		wantErr      bool
	}{
		{"MockWithNilFn", nil, args{nil, "", ""}, false},
		{"MockWithFn", func(source io.Reader, remotePath string, permissions string) error {
			return errors.New("expected error")
		}, args{nil, "", ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MockSSHClient{
				MockCopyFile: tt.MockCopyFile,
			}
			if err := s.CopyFile(tt.args.source, tt.args.remotePath, tt.args.permissions); (err != nil) != tt.wantErr {
				t.Errorf("MockSSHClient.CopyFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
