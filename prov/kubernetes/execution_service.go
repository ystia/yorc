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
	"strconv"
	"strings"

	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tosca"
)

const (
	// KubernetesServicePortMapping is the capability attribute of a public
	// endpoint specifying the the port provided by Kubernetes to expose to
	// external clients a docker host port.
	// This is a capability attribute set at runtime, while the container port
	// is a property attribute
	KubernetesServicePortMapping = "port"

	// DockerBridgePortMapping is the capability property of a endpoint
	// specifying the docker host port associated to a container port
	DockerBridgePortMapping = "docker_bridge_port_mapping"
)

// updatePortMappingPublicEndpoints updates public endpoint capabilities
// referencing a given port with the Kubernetes service Port Mapping infos
// so that this endpoint can be used by clients outside of the cluster
func (e *execution) updatePortMappingPublicEndpoints(port int32, ipAddress string, k8sPort int32) error {
	// Get endpoint capabilities for the node

	capNames, _ := deployments.GetCapabilitiesOfType(e.kv, e.deploymentID, e.nodeType, tosca.PublicEndpointCapability)

	if len(capNames) == 0 {
		// Nothing to update
		return nil
	}

	instances, err := deployments.GetNodeInstancesIds(e.kv, e.deploymentID, e.nodeName)
	if err != nil || len(instances) == 0 {
		return err
	}

	// Keep instances exposing the port
	instancesToUpdate := make([]string, 0)
	for _, instance := range instances {
		ports, err := deployments.GetInstanceAttributeValue(e.kv, e.deploymentID, e.nodeName, instance, "docker_ports")
		if err != nil {
			return err
		}
		if ports != nil && strings.Contains(ports.RawString(), strconv.Itoa(int(port))) {
			instancesToUpdate = append(instancesToUpdate, instance)
		}

	}

	for _, instance := range instancesToUpdate {
		for _, capName := range capNames {

			// First check this is an endpoint declaring this port
			// the port provided here is the docker mapping port, not
			// the port in the container
			result, err := deployments.GetInstanceCapabilityAttributeValue(e.kv, e.deploymentID,
				e.nodeName, instance, capName, DockerBridgePortMapping)
			if err != nil {
				return err
			}
			if result == nil || result.RawString() != strconv.Itoa(int(port)) {
				// This endpoint is using another port
				continue
			}
			err = deployments.SetInstanceCapabilityAttribute(e.deploymentID,
				e.nodeName, instance, capName, "ip_address", ipAddress)
			if err != nil {
				return err
			}
			err = deployments.SetInstanceCapabilityAttribute(e.deploymentID,
				e.nodeName, instance, capName, KubernetesServicePortMapping, strconv.Itoa(int(k8sPort)))
			if err != nil {
				return err
			}
			log.Debugf("Updated %s %s %s %s port mapping %d -> %s %d\n",
				e.deploymentID, e.nodeName, instance, capName, port, ipAddress, k8sPort)
		}
	}

	return nil

}
