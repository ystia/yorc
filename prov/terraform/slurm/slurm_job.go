package slurm

import (
	"fmt"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func (g *Generator) generateSlurmJob(url, deploymentId string) (Job, error) {
	var nodeType string
	var err error
	log.Printf("generateSlurmNode begin")
	if nodeType, err = g.getStringFormConsul(url, "type"); err != nil {
		return Job{}, err
	}
	if nodeType != "janus.nodes.slurm.Job" {
		return Job{}, fmt.Errorf("In slurm/generateSlurmJob : Unsupported node type for %s: %s", url, nodeType)
	}

	instance := Job{}

	if scriptPath, err := g.getStringFormConsul(url, "properties/scriptPath"); err != nil {
		return Job{}, fmt.Errorf("Missing mandatory parameter 'scriptPath' for %s", url)
	} else {
		instance.ScriptPath = scriptPath
	}

	if imgPath, err := g.getStringFormConsul(url, "properties/imgPath"); err != nil {
		return Job{}, fmt.Errorf("Missing mandatory parameter 'imgPath' for %s", url)
	} else {
		instance.ImgPath = imgPath
	}

	if nbNode, err := g.getStringFormConsul(url, "properties/nbNode"); err != nil {
		return Job{}, fmt.Errorf("Missing mandatory parameter 'nbNode' for %s", url)
	} else {
		instance.NbNode = nbNode
	}

	if nodesName, err := g.getStringFormConsul(url, "properties/nodesName"); err != nil {
		return Job{}, fmt.Errorf("Missing mandatory parameter 'nodesName' for %s", url)
	} else {
		instance.NodesName = nodesName
	}

	if nbMpiProcess, err := g.getStringFormConsul(url, "properties/nbMpiProcess"); err != nil {
		return Job{}, fmt.Errorf("Missing mandatory parameter 'nbMpiProcess' for %s", url)
	} else {
		instance.NbMpiProcess = nbMpiProcess
	}

	return instance, nil
}
