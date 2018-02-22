package kubernetes

import "github.com/ystia/yorc/registry"

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
