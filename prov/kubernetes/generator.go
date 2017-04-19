package kubernetes

import (
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"

	"github.com/hashicorp/consul/api"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"

	"encoding/json"
	"fmt"
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
		return nil, err
	}

	if memLimitStr == "" {
		memLimitStr = "0"
	}
	memLimit, err := resource.ParseQuantity(memLimitStr)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	if memShareStr == "" {
		memShareStr = "0"
	}
	memShare, err := resource.ParseQuantity(memShareStr)
	if err != nil {
		return nil, err
	}

	if memShare == (nilValue) {
		return v1.ResourceList{v1.ResourceCPU: cpuShare}, nil
	} else if cpuShare == (nilValue) {
		return v1.ResourceList{v1.ResourceMemory: memShare}, nil
	}

	return v1.ResourceList{v1.ResourceCPU: cpuShare, v1.ResourceMemory: memShare}, nil

}

func (k8s *K8sGenerator) GeneratePod(deploymentID, nodeName string) (v1.Pod, error) {
	found, dockerImage, err := deployments.GetNodeProperty(k8s.kv, deploymentID, nodeName, "image")
	if err != nil {
		return v1.Pod{}, err
	}
	if !found || dockerImage == "" {
		return v1.Pod{}, fmt.Errorf("Property image not found on node %s", nodeName)
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
			Name:   strings.ToLower(nodeName),
			Labels: map[string]string{"name": strings.ToLower(nodeName)},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            strings.ToLower(nodeName),
					Image:           dockerImage,
					ImagePullPolicy: v1.PullPolicy(imagePullPolicy),
					Command:         strings.Fields(dockerRunCmd),
					Resources: v1.ResourceRequirements{
						Requests: requests,
						Limits:   limits,
					},
				},
			},
		},
	}

	val, _ := json.Marshal(pod)
	fmt.Println(string(val))
	return pod, nil
}
