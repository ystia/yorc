package slurm

import (
	"fmt"

	"novaforge.bull.com/starlings-janus/janus/log"
)

func (g *Generator) generateSlurmNode(url, deploymentId string) (ComputeInstance, error) {
	log.Printf("generateSlurmNode begin")
	nodeType, err := g.getStringFormConsul(url, "type")
	if err != nil {
		return ComputeInstance{}, err
	}
	if nodeType != "janus.nodes.slurm.Compute" {
		return ComputeInstance{}, fmt.Errorf("In slurm/generateOSInstance : Unsupported node type for %s: %s", url, nodeType)
	}
	instance := ComputeInstance{}
	gpuType, err := g.getStringFormConsul(url, "properties/gpuType")
	if err != nil {
		return ComputeInstance{}, fmt.Errorf("Missing mandatory parameter 'gpuType' for %s", url)
	}
	instance.GpuType = gpuType
	return instance, nil
}
