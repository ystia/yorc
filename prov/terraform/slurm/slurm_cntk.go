package slurm

import (
	"fmt"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func (g *Generator) generateSlurmCntk(url, deploymentId string) (Cntk, error) {
	var nodeType string
	var err error
	log.Printf("generateSlurmNode begin")
	if nodeType, err = g.getStringFormConsul(url, "type"); err != nil {
		return Cntk{}, err
	}
	if nodeType != "janus.nodes.slurm.Cntk" {
		return Cntk{}, fmt.Errorf("In slurm/generateSlurmCntk : Unsupported node type for %s: %s", url, nodeType)
	}

	instance := Cntk{}

	if partition, err := g.getStringFormConsul(url, "properties/partition"); err != nil {
		return Cntk{}, fmt.Errorf("Missing mandatory parameter 'partition' for %s", url)
	} else {
		instance.Partition = partition
	}

	if runAsUser, err := g.getStringFormConsul(url, "properties/runAsUser"); err != nil {
		return Cntk{}, fmt.Errorf("Missing mandatory parameter 'runAsUser' for %s", url)
	} else {
		instance.RunAsUser = runAsUser
	}

	if modelPath, err := g.getStringFormConsul(url, "properties/modelPath"); err != nil {
		return Cntk{}, fmt.Errorf("Missing mandatory parameter 'modelPath' for %s", url)
	} else {
		instance.ModelPath = modelPath
	}

	if modelFile, err := g.getStringFormConsul(url, "properties/modelFile"); err != nil {
		return Cntk{}, fmt.Errorf("Missing mandatory parameter 'modelFile' for %s", url)
	} else {
		instance.ModelFile = modelFile
	}

	if imgPath, err := g.getStringFormConsul(url, "properties/imgPath"); err != nil {
		return Cntk{}, fmt.Errorf("Missing mandatory parameter 'imgPath' for %s", url)
	} else {
		instance.ImgPath = imgPath
	}

	if nbNode, err := g.getStringFormConsul(url, "properties/nbNode"); err != nil {
		return Cntk{}, fmt.Errorf("Missing mandatory parameter 'nbNode' for %s", url)
	} else {
		instance.NbNode = nbNode
	}

	if nodesName, err := g.getStringFormConsul(url, "properties/nodesName"); err != nil {
		return Cntk{}, fmt.Errorf("Missing mandatory parameter 'nodesName' for %s", url)
	} else {
		instance.NodesName = nodesName
	}

	if nbMpiProcess, err := g.getStringFormConsul(url, "properties/nbMpiProcess"); err != nil {
		return Cntk{}, fmt.Errorf("Missing mandatory parameter 'nbMpiProcess' for %s", url)
	} else {
		instance.NbMpiProcess = nbMpiProcess
	}

	return instance, nil
}
