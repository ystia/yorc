package slurm

import (
	"novaforge.bull.com/starlings-janus/janus/helper/sshutil"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

type slurmResourcesProvider struct {
	client *sshutil.SSHClient
}

func newResourcesProvider() prov.ResourcesProvider {
	return &slurmResourcesProvider{}
}

func (s *slurmResourcesProvider) GetResourcesUsage() (map[string]string, error) {
	m := make(map[string]string)
	//TODO retrieve real values
	m["key1"] = "value1"
	return m, nil
}
