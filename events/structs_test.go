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

package events

import "testing"

func TestStatusUpdateType_String(t *testing.T) {
	tests := []struct {
		name string
		i    StatusUpdateType
		want string
	}{
		{"InstanceToString", InstanceStatusChangeType, "instance"},
		{"DeploymentToString", DeploymentStatusChangeType, "deployment"},
		{"CustomCommandToString", CustomCommandStatusChangeType, "custom-command"},
		{"ScalingToString", ScalingStatusChangeType, "scaling"},
		{"WorkflowToString", WorkflowStatusChangeType, "workflow"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.i.String(); got != tt.want {
				t.Errorf("StatusUpdateType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatusUpdateTypeString(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    StatusUpdateType
		wantErr bool
	}{
		{"InstanceFromString", args{"instance"}, InstanceStatusChangeType, false},
		{"DeploymentFromString", args{"deployment"}, DeploymentStatusChangeType, false},
		{"CustomCommandFromString", args{"custom-command"}, CustomCommandStatusChangeType, false},
		{"ScalingFromString", args{"scaling"}, ScalingStatusChangeType, false},
		{"WorkflowFromString", args{"workflow"}, WorkflowStatusChangeType, false},
		{"UnknownFromString", args{"err"}, InstanceStatusChangeType, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StatusUpdateTypeString(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("StatusUpdateTypeString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("StatusUpdateTypeString() = %v, want %v", got, tt.want)
			}
		})
	}
}
