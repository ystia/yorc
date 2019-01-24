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

package terraform

import (
	"reflect"
	"testing"
)

func TestRetrieveAttributeInfo(t *testing.T) {
	type args struct {
		outputPath string
	}
	tests := []struct {
		name string
		args args
		want *attrInfo
	}{
		{"PathWithInstanceAttribute", args{"_yorc/deployments/welcomeApp-Environment/topology/instances/Compute/0/attributes/public_ip_address"}, &attrInfo{deploymentID: "welcomeApp-Environment", nodeName: "Compute", instanceName: "0", capabilityName: "", attributeName: "public_ip_address"}},
		{"PathWithNodeAttribute", args{"_yorc/deployments/welcomeApp-Environment/topology/instances/Compute//attributes/public_ip_address"}, &attrInfo{deploymentID: "welcomeApp-Environment", nodeName: "Compute", instanceName: "", capabilityName: "", attributeName: "public_ip_address"}},
		{"PathWithCapabilityAttribute", args{"_yorc/deployments/welcomeApp-Environment/topology/instances/Compute/0/capabilities/endpoint/attributes/ip_address"}, &attrInfo{deploymentID: "welcomeApp-Environment", nodeName: "Compute", instanceName: "0", capabilityName: "endpoint", attributeName: "ip_address"}},
		{"PathWithRelationshipAttribute", args{"_yorc/deployments/welcomeApp-Environment/topology/relationship_instances/MyVolume/0/1/attributes/device"}, &attrInfo{deploymentID: "welcomeApp-Environment", nodeName: "MyVolume", instanceName: "1", requirementIndex: "0", capabilityName: "", attributeName: "device"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := retrieveAttributeInfo(tt.args.outputPath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("retrieveAttributeInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}
