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

package bootstrap

import (
	"fmt"
	"time"

	"github.com/ystia/yorc/commands/deployments"
	"github.com/ystia/yorc/helper/ziputil"
)

// deployTopology deploys a topology provided under deploymentPath.
// Return the the ID of the deployment
func deployTopology(deploymentPath string) (string, error) {

	// Download Alien4Cloud whose zip is expected to be provided in the
	// deployment
	url := inputValues.Alien4cloud.DownloadURL
	if _, err := download(url, deploymentPath); err != nil {
		return "", err
	}

	csarZip, err := ziputil.ZipPath(deploymentPath)
	if err != nil {
		return "", err
	}

	t := time.Now()
	deploymentID := fmt.Sprintf("bootstrap-%d-%02d-%02d--%02d-%02d-%02d",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	client, err := getYorcClient()
	if err != nil {
		return "", err
	}

	_, err = deployments.SubmitCSAR(csarZip, client, deploymentID)
	if err != nil {
		return "", err
	}
	return deploymentID, err
}

// followDeployment prints and updates the deployment status until its end
func followDeployment(deploymentID string) error {
	client, err := getYorcClient()
	if err != nil {
		return err
	}

	err = deployments.DisplayInfo(client, deploymentID, false, true, 3*time.Second)

	return err
}
