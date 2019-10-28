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

package deployments

import (
	"path"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/helper/consulutil"
)

// DockerHubURL is the official URL for the docker hub
const DockerHubURL = "https://hub.docker.com/"

// SingularityHubURL is the official URL for the docker hub
const SingularityHubURL = "https://singularity-hub.org/"

// GetRepositoryURLFromName allow you to retrieve the url of a repo from is name
func GetRepositoryURLFromName(deploymentID, repoName string) (string, error) {
	repositoriesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "repositories")
	exist, res, err := consulutil.GetStringValue(path.Join(repositoriesPath, repoName, "url"))
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist {
		return "", errors.Errorf("The repository %v has been not found", repoName)
	}
	return res, nil
}

// GetRepositoryTokenTypeFromName retrieves the token_type of credential for a given repoName
func GetRepositoryTokenTypeFromName(deploymentID, repoName string) (string, error) {
	repositoriesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "repositories")
	exist, res, err := consulutil.GetStringValue(path.Join(repositoriesPath, repoName, "credentials", "token_type"))
	if err != nil {
		return "", errors.Wrap(err, "An error has occurred when trying to get repository token_type")
	}

	if !exist {
		return "", errors.Errorf("The repository %v has been not found", repoName)
	}

	return res, nil
}

// GetRepositoryTokenUserFromName This function get the credentials (user/token) for a given repoName
func GetRepositoryTokenUserFromName(deploymentID, repoName string) (string, string, error) {
	repositoriesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "repositories")
	existToken, token, err := consulutil.GetStringValue(path.Join(repositoriesPath, repoName, "credentials", "token"))
	if err != nil {
		return "", "", errors.Wrap(err, "An error has occurred when trying to get repository token")
	}
	if !existToken || token == "" {
		return "", "", errors.Errorf("The token for repository %v has been not found", repoName)
	}

	existUser, user, err := consulutil.GetStringValue(path.Join(repositoriesPath, repoName, "credentials", "user"))
	if err != nil {
		return "", "", errors.Wrap(err, "An error has occurred when trying to get repository user")
	}
	if !existUser || user == "" {
		return "", "", errors.Errorf("The user for repository %v has been not found", repoName)
	}

	return token, user, nil
}
