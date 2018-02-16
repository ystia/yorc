package kubernetes

import "novaforge.bull.com/starlings-janus/janus/registry"

const (
	kubernetesArtifactImplementation           = "tosca.artifacts.Deployment.Image.Container.Docker.Kubernetes"
	kubernetesDeploymentArtifactImplementation = "janus.artifacts.Deployment.Kubernetes"
)

// Default executor is registered to treat kubernetes artifacts deployment
func init() {
	reg := registry.GetRegistry()
	reg.RegisterOperationExecutor(
		[]string{
			kubernetesArtifactImplementation,
			kubernetesDeploymentArtifactImplementation,
		}, &defaultExecutor{}, registry.BuiltinOrigin)
}
