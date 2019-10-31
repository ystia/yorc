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
	"context"
	"testing"

	ctu "github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
)

func testsController(t *testing.T, srv *ctu.TestServer) {
	log.SetDebug(true)

	srv.PopulateKV(t, map[string][]byte{
		consulutil.DeploymentKVPrefix + "/dep-id/topology/nodes/node-deploy/type":                                                    []byte("yorc.nodes.kubernetes.api.types.DeploymentResource"),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/nodes/node-deploy/properties/resource_type":                                []byte(""),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/nodes/node-deploy/properties/resource_spec":                                []byte("{}"),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/nodes/node-deploy/properties/service_dependency_lookups":                   []byte(""),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/types/org.alien4cloud.kubernetes.api.types.DeploymentResource/name":        []byte("org.alien4cloud.kubernetes.api.types.DeploymentResource"),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/types/org.alien4cloud.kubernetes.api.types.DeploymentResource/.existFlag":  []byte(""),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/nodes/node-service/type":                                                   []byte("yorc.nodes.kubernetes.api.types.ServiceResource"),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/nodes/node-service/properties/resource_type":                               []byte(""),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/nodes/node-service/properties/resource_spec":                               []byte("{}"),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/types/org.alien4cloud.kubernetes.api.types.ServiceResource/name":           []byte("org.alien4cloud.kubernetes.api.types.ServiceResource"),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/types/org.alien4cloud.kubernetes.api.types.ServiceResource/.existFlag":     []byte(""),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/nodes/node-simpleresource/type":                                            []byte("yorc.nodes.kubernetes.api.types.SimpleResource"),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/nodes/node-simpleresource/properties/resource_type":                        []byte("pvc"),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/nodes/node-simpleresource/properties/resource_spec":                        []byte("{}"),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/types/org.alien4cloud.kubernetes.api.types.SimpleResource/name":            []byte("org.alien4cloud.kubernetes.api.types.ServiceResource"),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/types/org.alien4cloud.kubernetes.api.types.SimpleResource/.existFlag":      []byte(""),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/nodes/node-simpleresource-badresource/type":                                []byte("yorc.nodes.kubernetes.api.types.SimpleResource"),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/nodes/node-simpleresource-badresource/properties/resource_type":            []byte("bad"),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/nodes/node-simpleresource-badresource/properties/resource_spec":            []byte("{}"),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/nodes/node-statefulset/type":                                               []byte("yorc.nodes.kubernetes.api.types.StatefulSetResource"),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/nodes/node-statefulset/properties/resource_type":                           []byte(""),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/nodes/node-statefulset/properties/resource_spec":                           []byte("{}"),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/types/org.alien4cloud.kubernetes.api.types.StatefulSetResource/name":       []byte("org.alien4cloud.kubernetes.api.types.StatefulSetResource"),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/types/org.alien4cloud.kubernetes.api.types.StatefulSetResource/.existFlag": []byte(""),
		consulutil.DeploymentKVPrefix + "/dep-id/topology/nodes/node-deploy-nores-props/type":                                        []byte("yorc.nodes.kubernetes.api.types.DeploymentResource"),
	})

	ctx := context.Background()
	k8s := newTestK8s()
	tests := []struct {
		name     string
		nodeName string
		nodeType string
		wantErr  bool
	}{
		{
			"test k8sDeploymentResourceType",
			"node-deploy",
			"yorc.nodes.kubernetes.api.types.DeploymentResource",
			false,
		},
		{
			"test k8sServiceResourceType",
			"node-service",
			"yorc.nodes.kubernetes.api.types.ServiceResource",
			false,
		},
		{
			"test k8sSimpleResourceType",
			"node-simpleresource",
			"yorc.nodes.kubernetes.api.types.SimpleResource",
			false,
		},
		{
			"test k8sStatefulSetResourceType",
			"node-statefulset",
			"yorc.nodes.kubernetes.api.types.StatefulSetResource",
			false,
		},
		{
			"test unsupported k8s simple resource type",
			"node-simpleresource-badresource",
			"yorc.nodes.kubernetes.api.types.SimpleResource",
			true,
		},
		{
			"test unsupported k8s resource type",
			"node-badresource",
			"yorc.nodes.kubernetes.api.types.BadResource",
			true,
		},
		{
			"test no resource properties in k8s type",
			"node-deploy-nores-props",
			"yorc.nodes.kubernetes.api.types.DeploymentResource",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &execution{
				deploymentID: "dep-id",
				nodeName:     tt.nodeName,
				nodeType:     tt.nodeType,
			}

			k8sObject, err := e.getYorcK8sObject(ctx, k8s.clientset)
			if (err != nil) != tt.wantErr {
				t.Errorf("Failed %s : %s", tt.name, err)
			}
			if !tt.wantErr {
				require.NotNil(t, k8sObject)
			}

			if tt.nodeName == "node-deploy" {
				replacedRSpec, err := replaceServiceIPInResourceSpec(ctx, k8s.clientset, "dep-id", tt.nodeName, "", "{}")
				// TODO test replaceRSpec value
				require.Nil(t, err)
				require.NotNil(t, replacedRSpec)
			}
		})
	}

}
