package kubernetes

import (
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
)

// A k8sGenerator is used to generate the Kubernetes objects for a given TOSCA node
type k8sGenerator struct {
	kv  *api.KV
	cfg config.Configuration
}

// newGenerator create a K8sGenerator
func newGenerator(kv *api.KV, cfg config.Configuration) *k8sGenerator {
	return &k8sGenerator{kv: kv, cfg: cfg}
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

//GenerateNewRepoSecret generate a new struct for secret docker repo and fill it
func (k8s *k8sGenerator) createNewRepoSecret(client *kubernetes.Clientset, namespace, name string, data []byte) (*v1.Secret, error) {
	mySecret := &v1.Secret{}
	mySecret.Name = name
	mySecret.Type = v1.SecretTypeDockercfg
	mySecret.Data = map[string][]byte{}
	mySecret.Data[v1.DockerConfigKey] = data

	return client.CoreV1().Secrets(strings.ToLower(namespace)).Create(mySecret)
}

// CreateNamespaceIfMissing create a kubernetes namespace (only if missing)
func (k8s *k8sGenerator) createNamespaceIfMissing(deploymentID, namespaceName string, client *kubernetes.Clientset) error {
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

// generatePodName by replaceing '_' by '-'
func generatePodName(nodeName string) string {
	return strings.Replace(nodeName, "_", "-", -1)
}

func genereateVolumeMount(volumeName string, readOnly bool, mountPath, subPath string) v1.VolumeMount {
	return v1.VolumeMount{
		Name:      volumeName,
		ReadOnly:  readOnly,
		MountPath: mountPath,
		SubPath:   subPath,
	}
}

func (k8s *k8sGenerator) generateContainer(nodeName, dockerImage, imagePullPolicy, dockerRunCmd string, requests, limits v1.ResourceList, inputs []v1.EnvVar, volumeMounts []v1.VolumeMount) v1.Container {
	return v1.Container{
		Name:  strings.ToLower(k8s.cfg.ResourcesPrefix + nodeName),
		Image: dockerImage,
		//ImagePullPolicy: v1.PullIfNotPresent,
		ImagePullPolicy: v1.PullPolicy(imagePullPolicy),
		Command:         strings.Fields(dockerRunCmd),
		Resources: v1.ResourceRequirements{
			Requests: requests,
			Limits:   limits,
		},
		VolumeMounts: volumeMounts,
		Env:          inputs,
	}
}

// TODO treat errors
func (k8s *k8sGenerator) generateVolume(deploymentID, volumeNodeName string) (v1.Volume, error) {
	var err error
	_, vname, _ := deployments.GetNodeProperty(k8s.kv, deploymentID, volumeNodeName, "name")
	volume := v1.Volume{
		Name: vname,
	}
	_, vtype, _ := deployments.GetNodeProperty(k8s.kv, deploymentID, volumeNodeName, "volume_type")
	volumeSource := v1.VolumeSource{}
	switch vtype {
	case "emptyDir":
		emptyDirVolumeSource := k8s.generateEmptyDirVolumeSource(deploymentID, volumeNodeName)
		volumeSource.EmptyDir = &emptyDirVolumeSource
		volume.VolumeSource = volumeSource
		err = nil
	default:
		err = errors.Errorf("Unsupported volume type %q", vtype)
	}
	return volume, err
}

func (k8s *k8sGenerator) generateEmptyDirVolumeSource(deploymentID, volumeNodeName string) v1.EmptyDirVolumeSource {
	found, mediumVal, _ := deployments.GetNodeProperty(k8s.kv, deploymentID, volumeNodeName, "medium")
	if !found {
		mediumVal = ""
	}
	return v1.EmptyDirVolumeSource{
		Medium: v1.StorageMedium(mediumVal),
	}
}

// Get the names of the nodes corresponding to volumes mounted by the node 'nodeName'
func getUsedVolumeNodesNames(kv *api.KV, deploymentID, nodeName string) ([]string, error) {
	useVolumeKeys, err := deployments.GetRequirementsKeysByNameForNode(kv, deploymentID, nodeName, "use_volume")
	if err != nil {
		return nil, err
	}
	usedVolumeNodesNames := make([]string, 0)
	for _, useVolumeReqPrefix := range useVolumeKeys {
		requirementIndex := deployments.GetRequirementIndexFromRequirementKey(useVolumeReqPrefix)
		volumeNodeName, err := deployments.GetTargetNodeForRequirement(kv, deploymentID, nodeName, requirementIndex)
		log.Printf("###################### Node %s has requirement use_volume satisfyed by node %s", nodeName, volumeNodeName)
		if err != nil {
			return nil, err
		}
		usedVolumeNodesNames = append(usedVolumeNodesNames, volumeNodeName)
	}
	return usedVolumeNodesNames, nil
}

// generateDeployment generate Kubernetes Pod and Service to deploy based of given Node
func (k8s *k8sGenerator) generateDeployment(deploymentID, nodeName, operation, nodeType, repoName string, inputs []v1.EnvVar, nbInstances int32) (v1beta1.Deployment, v1.Service, error) {
	imgName, err := deployments.GetOperationImplementationFile(k8s.kv, deploymentID, nodeType, operation)
	if err != nil {
		return v1beta1.Deployment{}, v1.Service{}, err
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
		return v1beta1.Deployment{}, v1.Service{}, err
	}

	requests, err := generateRequestRessources(cpuShareStr, memShareStr)
	if err != nil {
		return v1beta1.Deployment{}, v1.Service{}, err
	}

	metadata := metav1.ObjectMeta{
		Name:   strings.ToLower(generatePodName(k8s.cfg.ResourcesPrefix + nodeName)),
		Labels: map[string]string{"name": strings.ToLower(nodeName), "nodeId": deploymentID + "-" + generatePodName(nodeName)},
	}

	var usedVolumeNodesNames []string
	var usedVolumes []v1.Volume
	var volumeMounts []v1.VolumeMount

	usedVolumeNodesNames, err = getUsedVolumeNodesNames(k8s.kv, deploymentID, nodeName)
	if err != nil {
		return v1beta1.Deployment{}, v1.Service{}, err
	}
	if len(usedVolumeNodesNames) == 0 {
		usedVolumes = nil
		volumeMounts = nil

		log.Printf("###################### Node %s uses no volume", nodeName)
	} else {
		log.Printf("###################### Node %s uses %d volumes", nodeName, len(usedVolumeNodesNames))
		//allNodeNames, err := deployments.GetNodes(k8s.kv, deploymentID)
		allNodeNames, _ := deployments.GetNodes(k8s.kv, deploymentID)
		// TODO treat err
		// treat volume nodes
		for _, aNodename := range allNodeNames {
			for _, volumeNodeName := range usedVolumeNodesNames {
				if 0 == strings.Compare(aNodename, volumeNodeName) {
					_, volumeName, _ := deployments.GetNodeProperty(k8s.kv, deploymentID, volumeNodeName, "name")
					log.Printf("###################### Found volume node %s containing volume %s used by node %s", volumeNodeName, volumeName, nodeName)

					// Found a volume node mounted by the current node
					// Generate the Kubernetes Volume object (instance of v1.Volume)
					volume, _ := k8s.generateVolume(deploymentID, volumeNodeName)

					// TODO treat error case
					usedVolumes = append(usedVolumes, volume)

					// Get the 'mount' capability of the volume
					found, mountPath, _ := deployments.GetCapabilityProperty(k8s.kv, deploymentID, volumeNodeName, "mount", "mountPath")
					if !found {
						// TODO treat error case in which mountPath property of mount capability not set
						// should never heapen as this is a required
					}
					found, subPath, _ := deployments.GetCapabilityProperty(k8s.kv, deploymentID, volumeNodeName, "mount", "subPath")
					if !found {
						// TODO treat error case in which subPath property of mount capability not set
					}
					// TODO Get readOnly bool property
					readOnly := false

					// Create a VolumeMount
					volumeMount := v1.VolumeMount{
						Name:      volumeName,
						ReadOnly:  readOnly,
						MountPath: mountPath,
						SubPath:   subPath,
					}
					volumeMounts = append(volumeMounts, volumeMount)
				}
			}
		}
	}

	container := k8s.generateContainer(nodeName, imgName, imagePullPolicy, dockerRunCmd, requests, limits, inputs, volumeMounts)

	var pullRepo []v1.LocalObjectReference

	if repoName != "" {
		pullRepo = append(pullRepo, v1.LocalObjectReference{Name: repoName})
	}

	deployment := v1beta1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metadata,
		Spec: v1beta1.DeploymentSpec{
			Replicas: &nbInstances,
			Selector: &metav1.LabelSelector{
				MatchLabels: metadata.Labels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metadata,
				Spec: v1.PodSpec{
					Volumes: usedVolumes,
					Containers: []v1.Container{
						container,
					},
					ImagePullSecrets: pullRepo,
				},
			},
		},
	}

	service := v1.Service{}

	if dockerPorts != "" {
		servicePorts := []v1.ServicePort{}
		portMaps := strings.Split(strings.Replace(dockerPorts, "\"", "", -1), " ")

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
				Selector: map[string]string{"nodeId": deploymentID + "-" + generatePodName(nodeName)},
				Ports:    servicePorts,
			},
		}
	}

	return deployment, service, nil
}
