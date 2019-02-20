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
	"reflect"
	"testing"
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
