package kubernetes

import "novaforge.bull.com/starlings-janus/janus/registry"

const (
	ImplementationKubernetes = "tosca.artifacts.Deployment.Image.Container.Kubernetes"
)

func init() {
	reg := registry.GetRegistry()
	reg.RegisterOperationExecutor(
		[]string{
			ImplementationKubernetes,
		}, NewExecutor(), registry.BuiltinOrigin)
}
