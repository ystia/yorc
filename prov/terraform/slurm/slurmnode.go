package slurm

import (
	"fmt"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func (g *Generator) generateSlurmNode(url, deploymentId string) (ComputeInstance, error) {
	var nodeType string
	var err error
	log.Printf("generateSlurmNode begin")
	if nodeType, err = g.getStringFormConsul(url, "type"); err != nil {
		return ComputeInstance{}, err
	}
	if nodeType != "janus.nodes.slurm.Compute" {
		return ComputeInstance{}, fmt.Errorf("In slurm/generateOSInstance : Unsupported node type for %s: %s", url, nodeType)
	}
	instance := ComputeInstance{}
	if gpuType, err := g.getStringFormConsul(url, "properties/gpuType"); err != nil {
		return ComputeInstance{}, fmt.Errorf("Missing mandatory parameter 'gpuType' for %s", url)
	} else {
		instance.GpuType = gpuType
	}
	return instance, nil
}

