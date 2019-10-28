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

package operations

import (
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ystia/yorc/v4/config"
)

const (
	workDirTest = "test"
)

func testGetOverlayPath(t *testing.T) {

	cfg := config.Configuration{
		WorkingDirectory: workDirTest,
	}

	deploymentID := path.Base(t.Name())
	overlayPath, _ := filepath.Abs(filepath.Join(workDirTest, "deployments", deploymentID, "overlay"))
	backupOverlayPath := strings.ReplaceAll(overlayPath, deploymentID, "."+deploymentID)

	tests := []struct {
		name    string
		taskID  string
		want    string
		wantErr bool
	}{
		{"TestWrongTaskID", "wrongTaskID", "", true},
		{"TestRegularTask", "deployTask", overlayPath, false},
		{"TestRemoveNodeTask", "removeTask", backupOverlayPath, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := GetOverlayPath(cfg, tt.taskID, deploymentID)
			if tt.wantErr && err == nil {
				t.Errorf("%s: Expected an error getting overlay path for task %s", tt.name, tt.taskID)
			} else if !tt.wantErr && err != nil {
				t.Errorf("%s: Expected no error getting overlay path for task %s", tt.name, tt.taskID)
			} else if tt.want != path {
				t.Errorf("%s: Wrong overloay path for task %s, expected %s, got %s",
					tt.name, tt.taskID, tt.want, path)
			}
		})
	}
}

func TestGetInstanceID(t *testing.T) {

	instanceID := "1243"
	instanceName := GetInstanceName("myNode", instanceID)

	computedID := GetInstanceID(instanceName)
	assert.Equal(t, instanceID, computedID, "Wrong instance ID for instance %s", instanceName)
}
