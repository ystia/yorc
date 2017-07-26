package kubernetes

import (
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"

	"github.com/hashicorp/consul/api"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"strconv"
	"strings"
)

// A K8sGenerator is used to generate the Kubernetes objects for a given TOSCA node
type K8sGenerator struct {
	kv  *api.KV
	cfg config.Configuration
}

// NewGenerator create a K8sGenerator
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

// CreateNamespaceIfMissing create a kubernetes namespace (only if missing)
func (k8s *K8sGenerator) CreateNamespaceIfMissing(deploymentID, namespaceName string, client *kubernetes.Clientset) error {
	_, err := client.CoreV1().Namespaces().Get(namespaceName, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			_, err := client.CoreV1().Namespaces().Create(&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: namespaceName},
			})
			if err != nil && !strings.Contains(err.Error(), "already exists") {
				return errors.Wrap(err, "Failed to create namespace")
			}
		} else {
			return errors.Wrap(err, "Failed to create namespace")
		}
	}
	return nil
}

// GeneratePodName by replaceing '_' by '-'
func GeneratePodName(nodeName string) string {
	return strings.Replace(nodeName, "_", "-", -1)
}

// GeneratePod generate Kubernetes Pod and Service to deploy based of given Node
func (k8s *K8sGenerator) GeneratePod(deploymentID, nodeName, operation, nodeType string, inputs []v1.EnvVar) (v1.Pod, v1.Service, error) {
	imgName, err := deployments.GetOperationImplementationFile(k8s.kv, deploymentID, nodeType, operation)
	if err != nil {
		return v1.Pod{}, v1.Service{}, err
	}

	_, cpuShareStr, err := deployments.GetNodeProperty(k8s.kv, deploymentID, nodeName, "cpu_share")
	_, cpuLimitStr, err := deployments.GetNodeProperty(k8s.kv, deploymentID, nodeName, "cpu_limit")
	_, memShareStr, err := deployments.GetNodeProperty(k8s.kv, deploymentID, nodeName, "mem_share")
	_, memLimitStr, err := deployments.GetNodeProperty(k8s.kv, deploymentID, nodeName, "mem_limit")
	_, imagePullPolicy, err := deployments.GetNodeProperty(k8s.kv, deploymentID, nodeName, "imagePullPolicy")
	_, dockerRunCmd, err := deployments.GetNodeProperty(k8s.kv, deploymentID, nodeName, "docker_run_cmd")
	_, dockerPorts, err := deployments.GetNodeProperty(k8s.kv, deploymentID, nodeName, "docker_ports")

	limits, err := generateLimitsRessources(cpuLimitStr, memLimitStr)
	if err != nil {
		return v1.Pod{}, v1.Service{}, err
	}

	requests, err := generateRequestRessources(cpuShareStr, memShareStr)
	if err != nil {
		return v1.Pod{}, v1.Service{}, err
	}

	metadata := metav1.ObjectMeta{
		Name:   strings.ToLower(GeneratePodName(k8s.cfg.ResourcesPrefix + nodeName)),
		Labels: map[string]string{"name": strings.ToLower(nodeName), "nodeId": deploymentID + "-" + GeneratePodName(nodeName)},
	}

	pod := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metadata,
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            strings.ToLower(GeneratePodName(k8s.cfg.ResourcesPrefix + nodeName)),
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
	service := v1.Service{}

	if dockerPorts != "" {
		servicePorts := []v1.ServicePort{}
		portMaps := strings.Split(dockerPorts, " ")

		for i, portMap := range portMaps {
			ports := strings.Split(portMap, ":")
			port, _ := strconv.Atoi(ports[0])
			var targetPort int32
			if len(ports) > 1 {
				p, _ := strconv.Atoi(ports[1])
				targetPort = int32(p)
			} else {
				p, _ := strconv.Atoi(ports[0])
				targetPort = int32(p)
			}
			servicePorts = append(servicePorts, v1.ServicePort{
				Name:       "port-" + strconv.Itoa(i),
				Port:       int32(port),
				TargetPort: intstr.IntOrString{IntVal: targetPort},
			})
		}

		service = v1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Service",
				APIVersion: "v1",
			},
			ObjectMeta: metadata,
			Spec: v1.ServiceSpec{
				Type:     v1.ServiceTypeNodePort,
				Selector: map[string]string{"nodeId": deploymentID + "-" + GeneratePodName(nodeName)},
				Ports:    servicePorts,
			},
		}
	}

	return pod, service, nil
}
