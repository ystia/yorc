package helper

import (
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/prov"
	"novaforge.bull.com/starlings-janus/janus/prov/ansible"
	"novaforge.bull.com/starlings-janus/janus/prov/kubernetes"
)

func GetOperationExecutor(kv *api.KV, deploymentID, nodeName, operation string) prov.OperationExecutor {
	nType, _ := deployments.GetNodeType(kv, deploymentID, nodeName)
	otype, _ := deployments.GetOperationImplementationType(kv, deploymentID, nType, operation)

	if otype == "tosca.artifacts.Deployment.Image.Container.Kubernetes" {
		return kubernetes.NewExecutor()
	}

	return ansible.NewExecutor()
}
