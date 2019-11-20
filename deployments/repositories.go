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
	"context"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"
	"path"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/helper/consulutil"
)

// DockerHubURL is the official URL for the docker hub
const DockerHubURL = "https://hub.docker.com/"

// SingularityHubURL is the official URL for the docker hub
const SingularityHubURL = "https://singularity-hub.org/"

func getRepository(ctx context.Context, deploymentID, repoName string) (bool, *tosca.Repository, error) {
	repository := new(tosca.Repository)
	repoPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "repositories", repoName)
	exist, err := storage.GetStore(types.StoreTypeDeployment).Get(repoPath, repository)

	// Default repository token is password
	if repository.Credit.TokenType == "" {
		repository.Credit.TokenType = "password"
	}
	return exist, repository, err
}

// GetRepositoryURLFromName allow you to retrieve the url of a repo from is name
func GetRepositoryURLFromName(ctx context.Context, deploymentID, repoName string) (string, error) {
	exist, repo, err := getRepository(ctx, deploymentID, repoName)
	if err != nil {
		return "", err
	}
	if !exist {
		return "", errors.Errorf("The repository %v has been not found", repoName)
	}
	return repo.URL, nil
}

// GetRepositoryTokenTypeFromName retrieves the token_type of credential for a given repoName
func GetRepositoryTokenTypeFromName(ctx context.Context, deploymentID, repoName string) (string, error) {
	exist, repo, err := getRepository(ctx, deploymentID, repoName)
	if err != nil {
		return "", err
	}
	if !exist {
		return "", errors.Errorf("The repository %v has been not found", repoName)
	}
	return repo.Credit.TokenType, nil
}

// GetRepositoryTokenUserFromName This function get the credentials (user/token) for a given repoName
func GetRepositoryTokenUserFromName(ctx context.Context, deploymentID, repoName string) (string, string, error) {
	exist, repo, err := getRepository(ctx, deploymentID, repoName)
	if err != nil {
		return "", "", err
	}
	if !exist {
		return "", "", errors.Errorf("The repository %v has been not found", repoName)
	}
	return repo.Credit.Token, repo.Credit.User, nil
}
