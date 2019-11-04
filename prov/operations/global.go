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
	"path/filepath"
	"strings"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/provutil"
	"github.com/ystia/yorc/v4/tasks"
)

// GetInstanceName returns the built instance name from nodeName and instanceID
func GetInstanceName(nodeName, instanceID string) string {
	return provutil.SanitizeForShell(nodeName + "_" + instanceID)
}

// GetInstanceID returns the instance ID from an instance name
func GetInstanceID(instanceName string) string {

	lastIndex := strings.LastIndex(instanceName, "_")
	return instanceName[lastIndex+1 : len(instanceName)]
}

// GetOverlayPath returns the overlay path
// In case of taskTypeRemoveNodes, it refers the backup overlay
func GetOverlayPath(cfg config.Configuration, taskID, deploymentID string) (string, error) {
	taskType, err := tasks.GetTaskType(taskID)
	if err != nil {
		return "", err
	}
	var prefix string
	if taskType == tasks.TaskTypeRemoveNodes {
		prefix = "."
	}
	p, err := filepath.Abs(filepath.Join(cfg.WorkingDirectory, "deployments", prefix+deploymentID, "overlay"))
	if err != nil {
		return "", err
	}
	return p, nil
}
