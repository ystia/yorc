package helper

import (
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/prov"
	"novaforge.bull.com/starlings-janus/janus/prov/ansible"
	"novaforge.bull.com/starlings-janus/janus/prov/kubernetes"
)

func GetOperationExecutor(kv *api.KV, deploymentID, nodeName string) prov.OperationExecutor {
	nType, _ := deployments.GetNodeType(kv, deploymentID, nodeName)
	otype, _ := deployments.GetOperationImplementationType(kv, deploymentID, nType, "tosca.interfaces.node.lifecycle.standard.start")

	if otype == "tosca.artifacts.Deployment.Image.Container.Docker.Kubernetes" {
		return kubernetes.NewExecutor()
	}

	return ansible.NewExecutor()
}
