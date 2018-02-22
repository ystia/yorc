package kubernetes

import "github.com/ystia/yorc/registry"

const (
	kubernetesArtifactImplementation           = "tosca.artifacts.Deployment.Image.Container.Docker.Kubernetes"
	kubernetesDeploymentArtifactImplementation = "yorc.artifacts.Deployment.Kubernetes"
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
