// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package openstack

import (
	"fmt"
	"os"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/log"
)

// Openstack resources that can be created on demand
const (
	blockStorageVolume         = "blockstorage_volume"
	computeInstance            = "compute_instance"
	computeFloatingIP          = "compute_floatingip"
	computeFloatingIPAssociate = "compute_floatingip_associate"
	computeServerGroup         = "compute_servergroup"
	computeVolumeAttach        = "compute_volume_attach"
	networkingNetwork          = "networking_network"
	networkingSubnet           = "networking_subnet"
	resourceTypeFormat         = "openstack_%s_%s"
	version2                   = "v2"
	latestBlockStorageVersion  = "v3"
	userDomainProperty         = "user_domain_name"
	userDomainEnvVar           = "OS_USER_DOMAIN_NAME"
)

// getOpenstackResourceTypes returns resource types supported by a given
// OpenStack infrastructure, for each resource that can be created on demand
// by the orchestrator
func getOpenstackResourceTypes(locationProps config.DynamicMap) map[string]string {

	resourceTypes := make(map[string]string)

	// Resources for which there is just one possible version supported
	v2Resources := []string{
		computeInstance, computeFloatingIP, computeFloatingIPAssociate, computeServerGroup,
		computeVolumeAttach, networkingNetwork, networkingSubnet,
	}

	for _, resource := range v2Resources {
		resourceTypes[resource] = fmt.Sprintf(resourceTypeFormat, resource, version2)
	}

	// Resources for which the version supported depends on the
	// openstack version installed on the selected infrastructure
	resourceTypes[blockStorageVolume] = fmt.Sprintf(resourceTypeFormat, blockStorageVolume,
		getBlockStorageAPIVersion(locationProps))

	return resourceTypes
}

// getBlockStorageAPIVersion returns the latest version supported on a given
// OpenStack infrastructure
func getBlockStorageAPIVersion(locationProps config.DynamicMap) string {

	version := latestBlockStorageVersion

	// The supported block storage API version is v3.
	// However for tests purposes, it is possible to use obsolete OpenStack
	// versions providing only block storage API version v1.
	// A basic check is performed here to detect if we are on an obsolete
	// OpenStack version:
	// If the User Domain mandatory property required by the Identity v3 API is not set,
	// then considering we are on an obsolete version of OpenStack providing
	// only block storage v1 API.
	// While on a supported OpenStack version, the user domain should be specified,
	// in which case, we can rely on the supported block storage v3 API
	if locationProps.GetString(userDomainProperty) == "" && os.Getenv(userDomainProperty) == "" {

		version = "v1"
		log.Printf("No OpenStack user domain specified in infrastructure => using obsolete Block Storage API %s",
			version)
	}

	return version

}
