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
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/helper/consulutil"

	"path"
)

// DockerHubURL is the official URL for the docker hub
const DockerHubURL = "https://hub.docker.com/"

// SingularityHubURL is the official URL for the docker hub
const SingularityHubURL = "https://singularity-hub.org/"

// GetRepositoryURLFromName allow you to retrieve the url of a repo from is name
func GetRepositoryURLFromName(kv *api.KV, deploymentID, repoName string) (url string, err error) {
	repositoriesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "repositories")
	res, _, err := kv.Get(path.Join(repositoriesPath, repoName, "url"), nil)
	if err != nil {
		err = errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		return
	}

	if res == nil {
		err = errors.Errorf("The repository %v has been not found", repoName)
		return
	}

	url = string(res.Value)
	return
}

// GetRepositoryTokenTypeFromName retrieves the token_type of credential for a given repoName
func GetRepositoryTokenTypeFromName(kv *api.KV, deploymentID, repoName string) (string, error) {
	repositoriesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "repositories")
	res, _, err := kv.Get(path.Join(repositoriesPath, repoName, "credentials", "token_type"), nil)
	if err != nil {
		return "", errors.Wrap(err, "An error has occurred when trying to get repository token_type")
	}

	if res == nil {
		return "", errors.Errorf("The repository %v has been not found", repoName)
	}

	return string(res.Value), nil
}

// GetRepositoryTokenUserFromName This function get the credentials (user/token) for a given repoName
func GetRepositoryTokenUserFromName(kv *api.KV, deploymentID, repoName string) (token string, user string, err error) {
	repositoriesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "repositories")
	res, _, err := kv.Get(path.Join(repositoriesPath, repoName, "credentials", "token"), nil)
	if err != nil {
		err = errors.Wrap(err, "An error has occurred when trying to get repository token")
		return
	}

	resUser, _, err := kv.Get(path.Join(repositoriesPath, repoName, "credentials", "user"), nil)
	if err != nil {
		err = errors.Wrap(err, "An error has occurred when trying to get repository user")
		return
	}

	if res == nil {
		err = errors.Errorf("The repository %v has been not found", repoName)
		return
	}

	token = string(res.Value)
	if resUser != nil {
		user = string(resUser.Value)
	} else {
		user = ""
	}

	return
}
