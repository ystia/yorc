package slurm

import (
	"fmt"

	"novaforge.bull.com/starlings-janus/janus/log"
)

func (g *slurmGenerator) generateSlurmCntk(url, deploymentID string) (Cntk, error) {
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
	partition, err := g.getStringFormConsul(url, "properties/partition")
	if err != nil {
		return Cntk{}, fmt.Errorf("Missing mandatory parameter 'partition' for %s", url)
	}
	instance.Partition = partition

	runAsUser, err := g.getStringFormConsul(url, "properties/runAsUser")
	if err != nil {
		return Cntk{}, fmt.Errorf("Missing mandatory parameter 'runAsUser' for %s", url)
	}
	instance.RunAsUser = runAsUser

	modelPath, err := g.getStringFormConsul(url, "properties/modelPath")
	if err != nil {
		return Cntk{}, fmt.Errorf("Missing mandatory parameter 'modelPath' for %s", url)
	}
	instance.ModelPath = modelPath

	modelFile, err := g.getStringFormConsul(url, "properties/modelFile")
	if err != nil {
		return Cntk{}, fmt.Errorf("Missing mandatory parameter 'modelFile' for %s", url)
	}
	instance.ModelFile = modelFile

	imgPath, err := g.getStringFormConsul(url, "properties/imgPath")
	if err != nil {
		return Cntk{}, fmt.Errorf("Missing mandatory parameter 'imgPath' for %s", url)
	}
	instance.ImgPath = imgPath

	nbNode, err := g.getStringFormConsul(url, "properties/nbNode")
	if err != nil {
		return Cntk{}, fmt.Errorf("Missing mandatory parameter 'nbNode' for %s", url)
	}
	instance.NbNode = nbNode

	nodesName, err := g.getStringFormConsul(url, "properties/nodesName")
	if err != nil {
		return Cntk{}, fmt.Errorf("Missing mandatory parameter 'nodesName' for %s", url)
	}
	instance.NodesName = nodesName

	nbMpiProcess, err := g.getStringFormConsul(url, "properties/nbMpiProcess")
	if err != nil {
		return Cntk{}, fmt.Errorf("Missing mandatory parameter 'nbMpiProcess' for %s", url)
	}
	instance.NbMpiProcess = nbMpiProcess

	return instance, nil
}
