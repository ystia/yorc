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
