package slurm

import (
	"context"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/helper/sshutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

type slurmResourcesProvider struct {
	client *sshutil.SSHClient
}

func newResourcesProvider() prov.ResourcesProvider {
	return &slurmResourcesProvider{}
}

func (s *slurmResourcesProvider) GetResourcesUsage(ctx context.Context, cfg config.Configuration) (map[string]string, error) {
	var err error
	m := make(map[string]string)

	s.client, err = getSSHClient(cfg)
	if err != nil {
		log.Printf("Unable to get resources usage due to:+v", err)
		return nil, err
	}
	if err = getCpuInfo(m, s.client); err != nil {
		log.Printf("Unable to get cpu usage due to:+v", err)
		return nil, err
	}
	if err = getJobInfo(m, s.client); err != nil {
		log.Printf("Unable to get job states due to:+v", err)
		return nil, err
	}
	return m, nil
}
