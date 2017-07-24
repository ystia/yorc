package deployments

import (
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"

	"path"
)

const DockerHubURL string = "https://hub.docker.com/"


//This function allow you to retrieve the url of a repo from is name
func GetRepositoryUrlFromName(kv *api.KV, deploymentID, repoName string) (url string, err error) {
	repositoriesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "repositories")
	res, _, err := kv.Get(path.Join(repositoriesPath, repoName, "url"), nil)
	if err != nil {
		err = errors.Wrap(err, "An error has occured when trying to get repository url")
		return
	}

	if res == nil {
		err = errors.Errorf("The repository %v has been not found", repoName)
		return
	}

	url = string(res.Value)
	return
}


func GetRepositoryTokenTypeFromName(kv *api.KV, deploymentID, repoName string) (token_type string, err error) {
	repositoriesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "repositories")
	res, _, err := kv.Get(path.Join(repositoriesPath, repoName, "credentials" ,"token_type"), nil)
	if err != nil {
		err = errors.Wrap(err, "An error has occured when trying to get repository token_type")
		return
	}

	if res == nil {
		err = errors.Errorf("The repository %v has been not found", repoName)
		return
	}

	token_type = string(res.Value)
	return
}


func GetRepositoryTokenUserFromName(kv *api.KV, deploymentID, repoName string) (token string, user string, err error) {
	repositoriesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "repositories")
	res, _, err := kv.Get(path.Join(repositoriesPath, repoName, "credentials" ,"token"), nil)
	if err != nil {
		err = errors.Wrap(err, "An error has occured when trying to get repository token")
		return
	}

	resUser, _, err := kv.Get(path.Join(repositoriesPath, repoName, "credentials" ,"user"), nil)
	if err != nil {
		err = errors.Wrap(err, "An error has occured when trying to get repository user")
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