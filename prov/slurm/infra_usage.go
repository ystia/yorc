package slurm

import (
	"context"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/helper/sshutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

type slurmInfraUsageCollector struct {
	client *sshutil.SSHClient
}

func newInfraUsageCollector() prov.InfrastructureUsageCollector {
	return &slurmInfraUsageCollector{}
}

func (s *slurmInfraUsageCollector) GetUsageInfo(ctx context.Context, cfg config.Configuration, taskID string) (map[string]string, error) {
	var err error
	m := make(map[string]string)

	s.client, err = getSSHClient(cfg)
	if err != nil {
		log.Printf("Unable to get usage info due to:%+v", err)
		return nil, err
	}
	if err = getCPUInfo(m, s.client); err != nil {
		log.Printf("Unable to get cpu usage info due to:%+v", err)
		return nil, err
	}
	if err = getJobInfo(m, s.client); err != nil {
		log.Printf("Unable to get job states info due to:%+v", err)
		return nil, err
	}
	return m, nil
}
