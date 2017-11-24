package deployments

import (
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"

	"path"
)

// DockerHubURL is the official URL for the docker hub
const DockerHubURL string = "https://hub.docker.com/"

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
