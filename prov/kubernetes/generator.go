package kubernetes

import (
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"

	"github.com/hashicorp/consul/api"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"

	"fmt"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"strings"
)

type K8sGenerator struct {
	kv  *api.KV
	cfg config.Configuration
}

func NewGenerator(kv *api.KV, cfg config.Configuration) *K8sGenerator {
	return &K8sGenerator{kv: kv, cfg: cfg}
}

func generateLimitsRessources(cpuLimitStr, memLimitStr string) (v1.ResourceList, error) {
	if cpuLimitStr == "" && memLimitStr == "" {
		return nil, nil
	}

	nilValue, _ := resource.ParseQuantity("0")
	if cpuLimitStr == "" {
		cpuLimitStr = "0"
	}
	cpuLimit, err := resource.ParseQuantity(cpuLimitStr)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse cpuLimit quantity")
	}

	if memLimitStr == "" {
		memLimitStr = "0"
	}
	memLimit, err := resource.ParseQuantity(memLimitStr)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse memLimit quantity")
	}

	if memLimit == (nilValue) {
		return v1.ResourceList{v1.ResourceCPU: cpuLimit}, nil
	} else if cpuLimit == (nilValue) {
		return v1.ResourceList{v1.ResourceMemory: memLimit}, nil
	}

	return v1.ResourceList{v1.ResourceCPU: cpuLimit, v1.ResourceMemory: memLimit}, nil

}

func generateRequestRessources(cpuShareStr, memShareStr string) (v1.ResourceList, error) {
	if cpuShareStr == "" && memShareStr == "" {
		return nil, nil
	}

	nilValue, _ := resource.ParseQuantity("0")
	if cpuShareStr == "" {
		cpuShareStr = "0"
	}
	cpuShare, err := resource.ParseQuantity(cpuShareStr)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse cpuShare quantity")
	}

	if memShareStr == "" {
		memShareStr = "0"
	}
	memShare, err := resource.ParseQuantity(memShareStr)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse memShare quantity")
	}

	if memShare == (nilValue) {
		return v1.ResourceList{v1.ResourceCPU: cpuShare}, nil
	} else if cpuShare == (nilValue) {
		return v1.ResourceList{v1.ResourceMemory: memShare}, nil
	}

	return v1.ResourceList{v1.ResourceCPU: cpuShare, v1.ResourceMemory: memShare}, nil

}

func (k8s *K8sGenerator) CreateNamespaceIfMissing(deploymentId, namespaceName string, client *kubernetes.Clientset) error {
	_, err := client.CoreV1().Namespaces().Get(namespaceName, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			_, err := client.CoreV1().Namespaces().Create(&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: namespaceName},
			})
			if err != nil {
				return errors.Wrap(err, "Failed to create namespace")
			}
		} else {
			return errors.Wrap(err, "Failed to create namespace")
		}
	}
	return nil
}

func (k8s *K8sGenerator) GeneratePod(deploymentID, nodeName, operation, nodeType string, inputs []v1.EnvVar) (v1.Pod, error) {
	imgName, err := deployments.GetOperationImplementationFile(k8s.kv, deploymentID, nodeType, operation)
	if err != nil {
		return v1.Pod{}, errors.Errorf("Property image not found on node %s", nodeName)
	}

	_, cpuShareStr, err := deployments.GetNodeProperty(k8s.kv, deploymentID, nodeName, "cpu_share")
	_, cpuLimitStr, err := deployments.GetNodeProperty(k8s.kv, deploymentID, nodeName, "cpu_limit")
	_, memShareStr, err := deployments.GetNodeProperty(k8s.kv, deploymentID, nodeName, "mem_share")
	_, memLimitStr, err := deployments.GetNodeProperty(k8s.kv, deploymentID, nodeName, "mem_limit")
	_, imagePullPolicy, err := deployments.GetNodeProperty(k8s.kv, deploymentID, nodeName, "imagePullPolicy")
	_, dockerRunCmd, err := deployments.GetNodeProperty(k8s.kv, deploymentID, nodeName, "docker_run_cmd")

	limits, err := generateLimitsRessources(cpuLimitStr, memLimitStr)
	if err != nil {
		return v1.Pod{}, err
	}

	requests, err := generateRequestRessources(cpuShareStr, memShareStr)
	if err != nil {
		return v1.Pod{}, err
	}

	pod := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   strings.ToLower(k8s.cfg.ResourcesPrefix + nodeName),
			Labels: map[string]string{"name": strings.ToLower(nodeName)},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            strings.ToLower(k8s.cfg.ResourcesPrefix + nodeName),
					Image:           imgName,
					ImagePullPolicy: v1.PullPolicy(imagePullPolicy),
					Command:         strings.Fields(dockerRunCmd),
					Resources: v1.ResourceRequirements{
						Requests: requests,
						Limits:   limits,
					},
					Env: inputs,
				},
			},
		},
	}

	return pod, nil
}
