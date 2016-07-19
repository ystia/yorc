package slurm

import (
	"fmt"
)

func (g *Generator) generateOSInstance(url, deploymentId string) (ComputeInstance, error) {
	var nodeType string
	var err error
	if nodeType, err = g.getStringFormConsul(url, "type"); err != nil {
		return ComputeInstance{}, err
	}
	if nodeType != "janus.nodes.slurm.Compute" {
		return ComputeInstance{}, fmt.Errorf("Unsupported node type for %s: %s", url, nodeType)
	}
	instance := ComputeInstance{}
	if nodeName, err := g.getStringFormConsul(url, "name"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.Name = nodeName
	}
	if gpuType, err := g.getStringFormConsul(url, "gpuType"); err != nil {
		return ComputeInstance{}, fmt.Errorf("Missing mandatory parameter 'gpuType' for %s", url)
	} else {
		instance.GpuType = gpuType
	}

	// Do this in order to be sure that ansible will be able to log on the instance
	instance.Provisioners = make(map[string]interface{})
	return instance, nil
}

