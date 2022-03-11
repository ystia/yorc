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

package validation

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/tasks"
	"github.com/ystia/yorc/v4/tasks/workflow/builder"
	"github.com/ystia/yorc/v4/tosca"
)

type mockActivity struct {
	t builder.ActivityType
	v string
}

func (m *mockActivity) Type() builder.ActivityType {
	return m.t
}

func (m *mockActivity) Value() string {
	return m.v
}

func (m *mockActivity) Inputs() map[string]tosca.ParameterDefinition {
	return nil
}
func testPostComputeCreationHook(t *testing.T, cfg config.Configuration) {
	ctx := context.Background()

	target := "Compute"
	type args struct {
		taskStatus   tasks.TaskStatus
		activityType builder.ActivityType
		endpointIPs  []string
		attributes   []map[string]string
	}
	tests := []struct {
		name   string
		args   args
		checks []string
	}{
		{"TestEndpointFromPublicIP", args{tasks.TaskStatusRUNNING, builder.ActivityTypeDelegate, nil, []map[string]string{
			map[string]string{"public_ip_address": "10.0.0.1", "public_address": "10.0.0.2", "private_address": "10.0.0.3", "ip_address": "10.0.0.4"},
			map[string]string{"public_ip_address": "10.0.1.1", "public_address": "10.0.1.2", "private_address": "10.0.1.3", "ip_address": "10.0.1.4"},
			map[string]string{"public_ip_address": "10.0.2.1", "public_address": "10.0.2.2", "private_address": "10.0.2.3", "ip_address": "10.0.2.4"},
			map[string]string{"public_ip_address": "10.0.3.1", "public_address": "10.0.3.2", "private_address": "10.0.3.3", "ip_address": "10.0.3.4"},
			map[string]string{"public_ip_address": "10.0.4.1", "public_address": "10.0.4.2", "private_address": "10.0.4.3", "ip_address": "10.0.4.4"},
		}},
			[]string{"10.0.0.1", "10.0.1.1", "10.0.2.1", "10.0.3.1", "10.0.4.1"},
		},
		{"TestEndpointFromPublicAddr", args{tasks.TaskStatusRUNNING, builder.ActivityTypeCallOperation, nil, []map[string]string{
			map[string]string{"public_address": "10.0.0.2", "private_address": "10.0.0.3", "ip_address": "10.0.0.4"},
			map[string]string{"public_address": "10.0.1.2", "private_address": "10.0.1.3", "ip_address": "10.0.1.4"},
			map[string]string{"public_address": "10.0.2.2", "private_address": "10.0.2.3", "ip_address": "10.0.2.4"},
			map[string]string{"public_address": "10.0.3.2", "private_address": "10.0.3.3", "ip_address": "10.0.3.4"},
			map[string]string{"public_address": "10.0.4.2", "private_address": "10.0.4.3", "ip_address": "10.0.4.4"},
		}},
			[]string{"10.0.0.2", "10.0.1.2", "10.0.2.2", "10.0.3.2", "10.0.4.2"},
		},
		{"TestEndpointFromPrivateAdd", args{tasks.TaskStatusRUNNING, builder.ActivityTypeDelegate, nil, []map[string]string{
			map[string]string{"private_address": "10.0.0.3", "ip_address": "10.0.0.4"},
			map[string]string{"private_address": "10.0.1.3", "ip_address": "10.0.1.4"},
			map[string]string{"private_address": "10.0.2.3", "ip_address": "10.0.2.4"},
			map[string]string{"private_address": "10.0.3.3", "ip_address": "10.0.3.4"},
			map[string]string{"private_address": "10.0.4.3", "ip_address": "10.0.4.4"},
		}},
			[]string{"10.0.0.3", "10.0.1.3", "10.0.2.3", "10.0.3.3", "10.0.4.3"},
		},
		{"TestEndpointFromIPAdd", args{tasks.TaskStatusRUNNING, builder.ActivityTypeCallOperation, nil, []map[string]string{
			map[string]string{"ip_address": "10.0.0.4"},
			map[string]string{"ip_address": "10.0.1.4"},
			map[string]string{"ip_address": "10.0.2.4"},
			map[string]string{"ip_address": "10.0.3.4"},
			map[string]string{"ip_address": "10.0.4.4"},
		}},
			[]string{"10.0.0.4", "10.0.1.4", "10.0.2.4", "10.0.3.4", "10.0.4.4"},
		},
		{"TestEndpointAlreadySet", args{tasks.TaskStatusRUNNING, builder.ActivityTypeDelegate, []string{"10.1.0.4", "10.1.1.4", "10.1.2.4", "10.1.3.4", "10.1.4.4"}, []map[string]string{
			map[string]string{"ip_address": "10.0.0.4"},
			map[string]string{"ip_address": "10.0.1.4"},
			map[string]string{"ip_address": "10.0.2.4"},
			map[string]string{"ip_address": "10.0.3.4"},
			map[string]string{"ip_address": "10.0.4.4"},
		}},
			[]string{"10.1.0.4", "10.1.1.4", "10.1.2.4", "10.1.3.4", "10.1.4.4"},
		},
		{"TestEndpointTaskFailed", args{tasks.TaskStatusFAILED, builder.ActivityTypeDelegate, nil, []map[string]string{
			map[string]string{"ip_address": "10.0.0.4"},
			map[string]string{"ip_address": "10.0.1.4"},
			map[string]string{"ip_address": "10.0.2.4"},
			map[string]string{"ip_address": "10.0.3.4"},
			map[string]string{"ip_address": "10.0.4.4"},
		}},
			[]string{"", "", "", "", ""},
		},
		{"TestEndpointTaskCancelled", args{tasks.TaskStatusCANCELED, builder.ActivityTypeDelegate, nil, []map[string]string{
			map[string]string{"ip_address": "10.0.0.4"},
			map[string]string{"ip_address": "10.0.1.4"},
			map[string]string{"ip_address": "10.0.2.4"},
			map[string]string{"ip_address": "10.0.3.4"},
			map[string]string{"ip_address": "10.0.4.4"},
		}},
			[]string{"", "", "", "", ""},
		},
		{"TestEndpointInlineActivity", args{tasks.TaskStatusRUNNING, builder.ActivityTypeInline, nil, []map[string]string{
			map[string]string{"ip_address": "10.0.0.4"},
			map[string]string{"ip_address": "10.0.1.4"},
			map[string]string{"ip_address": "10.0.2.4"},
			map[string]string{"ip_address": "10.0.3.4"},
			map[string]string{"ip_address": "10.0.4.4"},
		}},
			[]string{"", "", "", "", ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deploymentID := strings.Replace(t.Name(), "/", "_", -1)

			err := deployments.StoreDeploymentDefinition(ctx, deploymentID, "testdata/compute.yaml")
			require.Nil(t, err)

			u, err := uuid.NewV4()
			require.NoError(t, err, "Failed to generate a UUID")
			taskID := u.String()

			p := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, taskID, "status"), Value: []byte(strconv.Itoa(int(tt.args.taskStatus)))}
			_, err = consulutil.GetKV().Put(p, nil)
			require.NoError(t, err)

			activity := &mockActivity{
				t: tt.args.activityType,
				v: target,
			}

			for i, eip := range tt.args.endpointIPs {
				err = deployments.SetInstanceCapabilityAttribute(ctx, deploymentID, target, fmt.Sprint(i), "endpoint", "ip_address", eip)
			}
			for i, attrs := range tt.args.attributes {
				for k, v := range attrs {
					err = deployments.SetInstanceAttribute(ctx, deploymentID, target, fmt.Sprint(i), k, v)
					require.NoError(t, err)
				}
			}
			postComputeCreationHook(ctx, cfg, taskID, deploymentID, target, activity)

			for i, check := range tt.checks {
				actualIP, err := deployments.GetInstanceCapabilityAttributeValue(ctx, deploymentID, target, fmt.Sprint(i), "endpoint", "ip_address")
				require.NoError(t, err)
				if check != "" {
					require.NotNil(t, actualIP, "postComputeCreationHook: expecting a value for endpoint.ip_address attribute")
					assert.Equal(t, check, actualIP.RawString(), "postComputeCreationHook: Unexpected value for endpoint.ip_address attribute")
				}
			}
		})
	}
}
