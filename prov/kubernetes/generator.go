// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/prov"
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

func generateLimitsResources(cpuLimitStr, memLimitStr string) (v1.ResourceList, error) {
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

func generateRequestResources(cpuShareStr, memShareStr string) (v1.ResourceList, error) {
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
func (k8s *k8sGenerator) createNewRepoSecret(client kubernetes.Interface, namespace, name string, data []byte) (*v1.Secret, error) {
	mySecret := &v1.Secret{}
	mySecret.Name = name
	mySecret.Type = v1.SecretTypeDockercfg
	mySecret.Data = map[string][]byte{}
	mySecret.Data[v1.DockerConfigKey] = data

	return client.CoreV1().Secrets(strings.ToLower(namespace)).Create(mySecret)
}

// generatePodName by replacing '_' by '-'
func generatePodName(nodeName string) string {
	return strings.Replace(nodeName, "_", "-", -1)
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

// Generate all the Kubernetes Volumes used by a node
func (k8s *k8sGenerator) generateUsedVolumes(deploymentID, nodeName string) ([]v1.Volume, error) {
	usedVolumeNodeNames, err := getUsedVolumeNodesNames(k8s.kv, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	err = nil
	var usedVolumes []v1.Volume
	for _, volumeNodeName := range usedVolumeNodeNames {
		volume, err := k8s.generateVolume(deploymentID, volumeNodeName)
		if err == nil {
			usedVolumes = append(usedVolumes, volume)
		}
	}
	return usedVolumes, err
}

// Generate the kubernetes Volume matched by a K8s Volume Node
func (k8s *k8sGenerator) generateVolume(deploymentID, volumeNodeName string) (v1.Volume, error) {
	vname, err := deployments.GetNodePropertyValue(k8s.kv, deploymentID, volumeNodeName, "name")
	vtype, err := deployments.GetNodePropertyValue(k8s.kv, deploymentID, volumeNodeName, "volume_type")
	volume := v1.Volume{}
	if vname != nil {
		volume.Name = vname.RawString()
	}
	volumeSource := v1.VolumeSource{}
	if vtype != nil {
		switch vtype.RawString() {
		case "emptyDir":
			emptyDirVolumeSource := k8s.generateEmptyDirVolumeSource(deploymentID, volumeNodeName)
			volumeSource.EmptyDir = &emptyDirVolumeSource
			volume.VolumeSource = volumeSource
			err = nil
		default:
			err = errors.Errorf("Unsupported volume type %q", vtype)
		}
	}
	return volume, err
}

// Generate an emptyDir Kubernetes Volume
func (k8s *k8sGenerator) generateEmptyDirVolumeSource(deploymentID, volumeNodeName string) v1.EmptyDirVolumeSource {
	//TODO is this a good idea to ignore returned error?
	mediumVal, _ := deployments.GetNodePropertyValue(k8s.kv, deploymentID, volumeNodeName, "medium")
	medium := ""
	if mediumVal != nil {
		medium = mediumVal.RawString()
	}
	return v1.EmptyDirVolumeSource{
		Medium: v1.StorageMedium(medium),
	}
}

// Generate a kubernetes VolumeMount corresponding to a used volume
// Need properties of the 'mount' capability of the used volume node
func (k8s *k8sGenerator) generateVolumeMount(deploymentID, volumeNodeName string) (v1.VolumeMount, error) {
	// create default structure to be returned in case of error
	volumeMount := v1.VolumeMount{}
	volumeName, err := deployments.GetNodePropertyValue(k8s.kv, deploymentID, volumeNodeName, "name")
	if err != nil {
		return volumeMount, err
	}
	if volumeName == nil {
		return volumeMount, errors.Errorf("Volume node %q needs a name property", volumeNodeName)
	}
	volumeMount.Name = volumeName.RawString()
	mountPath, err := deployments.GetCapabilityPropertyValue(k8s.kv, deploymentID, volumeNodeName, "mount", "mount_path")
	if err != nil {
		return volumeMount, err
	}
	if mountPath == nil {
		return volumeMount, errors.Errorf("Volume node %q needs mount capability with mount_path property", volumeNodeName)
	}
	volumeMount.MountPath = mountPath.RawString()
	subPath, err := deployments.GetCapabilityPropertyValue(k8s.kv, deploymentID, volumeNodeName, "mount", "sub_path")
	if err != nil {
		return volumeMount, err
	}
	if subPath != nil {
		volumeMount.SubPath = subPath.RawString()
	}
	ro, err := deployments.GetCapabilityPropertyValue(k8s.kv, deploymentID, volumeNodeName, "mount", "read_only")
	if err != nil {
		return volumeMount, err
	}
	if ro != nil {
		// TODO should we sillently ignore an error
		volumeMount.ReadOnly, _ = strconv.ParseBool(ro.RawString())
	}
	return volumeMount, nil
}

// Generate the kubernetes VolumeMounts for a node
func (k8s *k8sGenerator) generateVolumeMounts(deploymentID, nodeName string) ([]v1.VolumeMount, error) {
	usedVolumeNodeNames, err := getUsedVolumeNodesNames(k8s.kv, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	var volumeMounts []v1.VolumeMount

	for _, volumeNodeName := range usedVolumeNodeNames {
		volumeMount, err := k8s.generateVolumeMount(deploymentID, volumeNodeName)
		if err != nil {
			return nil, err
		}
		volumeMounts = append(volumeMounts, volumeMount)
	}
	return volumeMounts, err
}

// Get the node names corresponding to volumes mounted by the node 'nodeName'
// Used volumes are obtained based on the requirements named 'use_volume'
func getUsedVolumeNodesNames(kv *api.KV, deploymentID, nodeName string) ([]string, error) {
	useVolumeKeys, err := deployments.GetRequirementsKeysByTypeForNode(kv, deploymentID, nodeName, "use_volume")
	if err != nil {
		return nil, err
	}
	usedVolumeNodesNames := make([]string, 0)
	for _, useVolumeReqPrefix := range useVolumeKeys {
		requirementIndex := deployments.GetRequirementIndexFromRequirementKey(useVolumeReqPrefix)
		volumeNodeName, err := deployments.GetTargetNodeForRequirement(kv, deploymentID, nodeName, requirementIndex)

		log.Printf("Node %s has requirement use_volume satisfyed by node %s", nodeName, volumeNodeName)
		if err != nil {
			return nil, err
		}
		usedVolumeNodesNames = append(usedVolumeNodesNames, volumeNodeName)
	}
	return usedVolumeNodesNames, nil
}

// GenerateDeployment generate Kubernetes Pod and Service to deploy based of given Node
func (k8s *k8sGenerator) generateDeployment(deploymentID, nodeName string, operation prov.Operation, nodeType, repoName string, inputs []v1.EnvVar, nbInstances int32) (v1beta1.Deployment, v1.Service, error) {
	imgName, err := deployments.GetOperationImplementationFile(k8s.kv, deploymentID, operation.ImplementedInNodeTemplate, nodeType, operation.Name)
	if err != nil {
		return v1beta1.Deployment{}, v1.Service{}, err
	}

	// TODO make these properties coherent with Tosca node type properties
	// Currently only mem_limit and cpu_share are defined
	cpuShareStr := ""
	cpuShareValue, err := deployments.GetNodePropertyValue(k8s.kv, deploymentID, nodeName, "cpu_share")
	if err != nil {
		return v1beta1.Deployment{}, v1.Service{}, err
	}
	if cpuShareValue != nil {
		cpuShareStr = cpuShareValue.RawString()
	}
	// not implemented
	// cpuLimitValue, err := deployments.GetNodePropertyValue(k8s.kv, deploymentID, nodeName, "cpu_limit")
	cpuLimitStr := ""
	// mem_share does not exist neither in docker nor in K8s
	// memShareValue, err := deployments.GetNodePropertyValue(k8s.kv, deploymentID, nodeName, "mem_share")
	memShareStr := ""
	memLimitStr := ""
	memLimitValue, err := deployments.GetNodePropertyValue(k8s.kv, deploymentID, nodeName, "mem_limit")
	if err != nil {
		return v1beta1.Deployment{}, v1.Service{}, err
	}
	if memLimitValue != nil {
		memLimitStr = memLimitValue.RawString()
	}
	imagePullPolicy := ""
	imagePullPolicyValue, err := deployments.GetNodePropertyValue(k8s.kv, deploymentID, nodeName, "imagePullPolicy")
	if err != nil {
		return v1beta1.Deployment{}, v1.Service{}, err
	}
	if imagePullPolicyValue != nil {
		imagePullPolicy = imagePullPolicyValue.RawString()
	}
	dockerRunCmd := ""
	dockerRunCmdValue, err := deployments.GetNodePropertyValue(k8s.kv, deploymentID, nodeName, "docker_run_cmd")
	if err != nil {
		return v1beta1.Deployment{}, v1.Service{}, err
	}
	if dockerRunCmdValue != nil {
		dockerRunCmd = dockerRunCmdValue.RawString()
	}
	dockerPorts := ""
	dockerPortsValue, err := deployments.GetNodePropertyValue(k8s.kv, deploymentID, nodeName, "docker_ports")
	if err != nil {
		return v1beta1.Deployment{}, v1.Service{}, err
	}
	if dockerPortsValue != nil {
		dockerPorts = dockerPortsValue.RawString()
	}
	limits, err := generateLimitsResources(cpuLimitStr, memLimitStr)
	if err != nil {
		return v1beta1.Deployment{}, v1.Service{}, err
	}

	// mem_share does not exist neither in docker nor in K8s
	// maybe should be replaced with mem_requests
	requests, err := generateRequestResources(cpuShareStr, memShareStr)
	if err != nil {
		return v1beta1.Deployment{}, v1.Service{}, err
	}

	metadata := metav1.ObjectMeta{
		Name:   strings.ToLower(generatePodName(k8s.cfg.ResourcesPrefix + nodeName)),
		Labels: map[string]string{"name": strings.ToLower(nodeName), "nodeId": deploymentID + "-" + generatePodName(nodeName)},
	}

	volumeMounts, err := k8s.generateVolumeMounts(deploymentID, nodeName)
	if err != nil {
		return v1beta1.Deployment{}, v1.Service{}, err
	}

	container := k8s.generateContainer(nodeName, imgName, imagePullPolicy, dockerRunCmd, requests, limits, inputs, volumeMounts)

	var pullRepo []v1.LocalObjectReference

	if repoName != "" {
		pullRepo = append(pullRepo, v1.LocalObjectReference{Name: repoName})
	}

	usedVolumes, err := k8s.generateUsedVolumes(deploymentID, nodeName)
	if err != nil {
		return v1beta1.Deployment{}, v1.Service{}, err
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
