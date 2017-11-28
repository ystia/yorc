package slurm

import (
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"novaforge.bull.com/starlings-janus/janus/log"
	"regexp"
	"strings"
)

func (g *slurmGenerator) generateSlurmNode(kv *api.KV, url, deploymentID string) (ComputeInstance, error) {
	log.Printf("generateSlurmNode begin")
	nodeType, err := g.getStringFormConsul(kv, url, "type")
	if err != nil {
		return ComputeInstance{}, err
	}
	if nodeType != "janus.nodes.slurm.Compute" {
		return ComputeInstance{}, errors.Errorf("In slurm/generateSlurmNode : Unsupported node type for %s: %s", url, nodeType)
	}
	instance := ComputeInstance{}

	// Set the Compute instance CPU and memory property from Tosca Compute 'host' capability property
	cpu, err := g.getStringFormConsul(kv, url, "capabilities/host/properties/num_cpus")
	if err != nil {
		return ComputeInstance{}, errors.Errorf("Missing parameter 'num_cpus' for %s", url)
	}
	instance.CPU = cpu

	memory, err := g.getStringFormConsul(kv, url, "capabilities/host/properties/mem_size")
	if err != nil {
		return ComputeInstance{}, errors.Errorf("Missing parameter 'memory' for %s", url)
	}

	// Only take one letter for defining memory unit and remove the blank space between number and unit.
	re := regexp.MustCompile("[[:digit:]]*[[:upper:]]{1}")
	instance.Memory = re.FindString(strings.Replace(memory, " ", "", -1))

	// Set the Compute instance gres property from Tosca slurm.Compute property
	gres, err := g.getStringFormConsul(kv, url, "properties/gres")
	if err != nil {
		return ComputeInstance{}, errors.Errorf("Missing parameter 'gres' for %s", url)
	}
	instance.Gres = gres

	//Set the Compute instance partition property from Tosca slurm.Compute property
	partition, err := g.getStringFormConsul(kv, url, "properties/partition")
	if err != nil {
		return ComputeInstance{}, errors.Errorf("Missing parameter 'partition' for %s", url)
	}
	instance.Partition = partition
	return instance, nil
}
