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

// import (
// 	"context"
// 	"sync"
// 	"testing"

// 	"golang.org/x/crypto/ssh"

// 	"github.com/ystia/yorc/log"
// )

// var sc = &SSHClient{
// 	Config: &ssh.ClientConfig{
// 		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
// 		User:            "user",
// 		Auth:            []ssh.AuthMethod{ssh.Password("*pass*")},
// 	},
// 	Host: "host",
// 	Port: 22,
// }

// func TestConnectionSW(t *testing.T) {

// 	var wg sync.WaitGroup
// 	for i := 0; i < 55; i++ {
// 		sw, err := sc.GetSessionWrapper()
// 		if err != nil {
// 			t.Errorf("%+v", err)
// 		}
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			err := sw.RunCommand(context.Background(), "sleep 6")
// 			if err != nil {
// 				t.Errorf("%+v", err)
// 			}
// 		}()
// 	}
// 	wg.Wait()
// }

// func TestConnectionDirect(t *testing.T) {
// 	log.SetDebug(true)
// 	var wg sync.WaitGroup
// 	for i := 0; i < 55; i++ {
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			_, err := sc.RunCommand("sleep 10")
// 			if err != nil {
// 				t.Errorf("Err: %+v", err)
// 			}
// 		}()
// 	}
// 	wg.Wait()
// }
