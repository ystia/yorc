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

package deployments

import (
	"path"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v3/helper/consulutil"

	"github.com/hashicorp/consul/api"
)

func TestBuildAttributeData(t *testing.T) {
	type args struct {
		outputPath string
	}
	tests := []struct {
		name string
		args args
		want *AttributeData
	}{
		{"PathWithInstanceAttribute", args{"_yorc/deployments/welcomeApp-Environment/topology/instances/Compute/0/attributes/public_ip_address"}, &AttributeData{DeploymentID: "welcomeApp-Environment", NodeName: "Compute", InstanceName: "0", CapabilityName: "", Name: "public_ip_address"}},
		{"PathWithNodeAttribute", args{"_yorc/deployments/welcomeApp-Environment/topology/instances/Compute//attributes/public_ip_address"}, &AttributeData{DeploymentID: "welcomeApp-Environment", NodeName: "Compute", InstanceName: "", CapabilityName: "", Name: "public_ip_address"}},
		{"PathWithCapabilityAttribute", args{"_yorc/deployments/welcomeApp-Environment/topology/instances/Compute/0/capabilities/endpoint/attributes/ip_address"}, &AttributeData{DeploymentID: "welcomeApp-Environment", NodeName: "Compute", InstanceName: "0", CapabilityName: "endpoint", Name: "ip_address"}},
		{"PathWithRelationshipAttribute", args{"_yorc/deployments/welcomeApp-Environment/topology/relationship_instances/MyVolume/0/1/attributes/device"}, &AttributeData{DeploymentID: "welcomeApp-Environment", NodeName: "MyVolume", InstanceName: "1", RequirementIndex: "0", CapabilityName: "", Name: "device"}},
		{"AnotherPathWithAttribute", args{"_yorc/deployments/slurmApp-Environment/topology/instances/ComputeSlurmCtl_address/0/attributes/ip_address"}, &AttributeData{DeploymentID: "slurmApp-Environment", NodeName: "ComputeSlurmCtl_address", InstanceName: "0", CapabilityName: "", Name: "ip_address"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildAttributeDataFromPath(tt.args.outputPath)
			if err != nil {
				t.Errorf("TestBuildAttributeData() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toAttribute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func assertOneConsulKeyExistAndHasValueInList(t *testing.T, kv *api.KV, keyPrefix string, expectedValue []byte) {
	kvps, _, err := kv.List(keyPrefix, nil)
	require.NoError(t, err, "Consul error on listing keys under %q", keyPrefix)
	var found bool
	for _, kvp := range kvps {
		if reflect.DeepEqual(kvp.Value, expectedValue) {
			found = true
		}
	}
	assert.True(t, found, "Cant find key with value %s under prefix %q", expectedValue, keyPrefix)
}

func testAddSubstitutionMappingAttributeHostNotification(t *testing.T, kv *api.KV, deploymentID string) {
	depPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID)
	instancesPath := path.Join(depPath, "topology/instances")
	// attr3 of SrvImpl1Instance need to be notified of private_ip due to {get_attribute: [SELF, cap2, ip_address]}
	assertOneConsulKeyExistAndHasValueInList(t, kv, path.Join(instancesPath, "Compute/0/attribute_notifications/private_address/"), []byte("SrvImpl1Instance/0/attributes/a3"))

	// a1 of SrvImpl1Instance need to be notified of attr1 due to {get_attribute: [SELF, attr1]}
	assertOneConsulKeyExistAndHasValueInList(t, kv, path.Join(instancesPath, "SrvImpl1Instance/0/attribute_notifications/attr1/"), []byte("SrvImpl1Instance/0/attributes/a1"))

	// a2 of SrvImpl1Instance need to be notified of cap1_attr1 due to {get_attribute: [SELF, cap1, cap1_attr1]}
	assertOneConsulKeyExistAndHasValueInList(t, kv, path.Join(instancesPath, "SrvImpl1Instance/0/capabilities/cap1/attribute_notifications/cap1_attr1/"), []byte("SrvImpl1Instance/0/attributes/a2"))
	// capabilities.cap1.cap1_attr1 of SrvImpl1Instance need to be notified of cap1_attr1 due to substition mapping
	assertOneConsulKeyExistAndHasValueInList(t, kv, path.Join(instancesPath, "SrvImpl1Instance/0/capabilities/cap1/attribute_notifications/cap1_attr1/"), []byte("SrvImpl1Instance/0/attributes/capabilities.cap1.cap1_attr1"))
	// capabilities.cap1.cap1_attr1 of SrvImpl2Instance need to be notified of cap1_attr1 due to substition mapping
	assertOneConsulKeyExistAndHasValueInList(t, kv, path.Join(instancesPath, "SrvImpl1Instance/0/capabilities/cap1/attribute_notifications/cap1_attr1/"), []byte("SrvImpl2Instance/0/attributes/capabilities.cap1.cap1_attr1"))

	// capabilities.cap1.cap1_attr1 of SrvImpl2Instance need to be notified of cap1_attr1 due to substition mapping
	assertOneConsulKeyExistAndHasValueInList(t, kv, path.Join(instancesPath, "SrvImpl2Instance/0/capabilities/cap1/attribute_notifications/cap1_attr1/"), []byte("SrvImpl2Instance/0/attributes/capabilities.cap1.cap1_attr1"))

}
