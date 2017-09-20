package kubernetes

import "novaforge.bull.com/starlings-janus/janus/registry"

const (
	kubernetesArtifactImplementation = "tosca.artifacts.Deployment.Image.Container.Docker.Kubernetes"
)

func init() {
	reg := registry.GetRegistry()
	reg.RegisterOperationExecutor(
		[]string{
			kubernetesArtifactImplementation,
		}, &defaultExecutor{}, registry.BuiltinOrigin)
}
