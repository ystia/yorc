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
	"fmt"
	"testing"

	"github.com/hashicorp/consul/api"
	ctu "github.com/hashicorp/consul/testutil"

	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v3/deployments/internal"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/log"
)

func testDeploymentNodes(t *testing.T, srv1 *ctu.TestServer, kv *api.KV) {
	log.SetDebug(true)

	srv1.PopulateKV(t, map[string][]byte{
		// Test testIsNodeTypeDerivedFrom
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/yorc.type.1/" + internal.TypeExistsFlagName:                  []byte(""),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/yorc.type.1/derived_from":                                    []byte("yorc.type.2"),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/yorc.type.1/name":                                            []byte("yorc.type.1"),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/yorc.type.2/" + internal.TypeExistsFlagName:                  []byte(""),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/yorc.type.2/derived_from":                                    []byte("yorc.type.3"),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/yorc.type.2/name":                                            []byte("yorc.type.2"),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/yorc.type.3/" + internal.TypeExistsFlagName:                  []byte(""),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/yorc.type.3/derived_from":                                    []byte("tosca.relationships.HostedOn"),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/yorc.type.3/name":                                            []byte("yorc.type.3"),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/tosca.relationships.HostedOn/" + internal.TypeExistsFlagName: []byte(""),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/tosca.relationships.HostedOn/name":                           []byte("tosca.relationships.HostedOn"),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/tosca.relationships.HostedOn/derived_from":                   []byte("tosca.relationships.Root"),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/tosca.relationships.Root/" + internal.TypeExistsFlagName:     []byte(""),
		consulutil.DeploymentKVPrefix + "/testIsNodeTypeDerivedFrom/topology/types/tosca.relationships.Root/name":                               []byte("tosca.relationships.Root"),

		// Test testGetNbInstancesForNode
		// Default case type "tosca.nodes.Compute" default_instance specified
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Compute1/type":                                               []byte("tosca.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Compute1/attributes/id":                                      []byte("Not Used as it exists in instances"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Compute1/capabilities/scalable/properties/default_instances": []byte("10"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Compute1/capabilities/scalable/properties/max_instances":     []byte("20"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Compute1/capabilities/scalable/properties/min_instances":     []byte("2"),
		// Case type "tosca.nodes.Compute" default_instance not specified (1 assumed)
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Compute2/type": []byte("tosca.nodes.Compute"),
		// Error case default_instance specified but not an uint
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Compute3/type":                                               []byte("tosca.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Compute3/capabilities/scalable/properties/default_instances": []byte("-10"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Compute3/capabilities/scalable/properties/max_instances":     []byte("-15"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Compute3/capabilities/scalable/properties/min_instances":     []byte("-15"),
		// Case Node Hosted on another node
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.1/" + internal.TypeExistsFlagName:                  []byte(""),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.1/derived_from":                                    []byte("yorc.type.2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.1/name":                                            []byte("yorc.type.1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.2/" + internal.TypeExistsFlagName:                  []byte(""),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.2/derived_from":                                    []byte("yorc.type.3"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.2/name":                                            []byte("yorc.type.2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.3/" + internal.TypeExistsFlagName:                  []byte(""),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.3/derived_from":                                    []byte("tosca.relationships.HostedOn"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.3/name":                                            []byte("yorc.type.3"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.relationships.HostedOn/" + internal.TypeExistsFlagName: []byte(""),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.relationships.HostedOn/name":                           []byte("tosca.relationships.HostedOn"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.relationships.HostedOn/derived_from":                   []byte("tosca.relationships.Root"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.relationships.Root/" + internal.TypeExistsFlagName:     []byte(""),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.relationships.Root/name":                               []byte("tosca.relationships.Root"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.Compute/" + internal.TypeExistsFlagName: []byte(""),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.Compute/name":                           []byte("tosca.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.Compute/capabilities/scalable/type":     []byte("tosca.capabilities.Scalable"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.Compute/attributes/id/default":          []byte("DefaultComputeTypeid"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.Compute/attributes/ip/default":          []byte(""),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.capabilities.Scalable/" + internal.TypeExistsFlagName: []byte(""),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.capabilities.Scalable/name":                           []byte("tosca.capabilities.Scalable"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.DerivedSC1/" + internal.TypeExistsFlagName: []byte(""),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.DerivedSC1/derived_from":                   []byte("tosca.nodes.SoftwareComponent"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.DerivedSC1/name":                           []byte("yorc.type.DerivedSC1"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.DerivedSC2/" + internal.TypeExistsFlagName: []byte(""),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.DerivedSC2/derived_from":                   []byte("tosca.nodes.SoftwareComponent"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.DerivedSC2/name":                           []byte("yorc.type.DerivedSC2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.DerivedSC2/attributes/dsc2/default":        []byte("yorc.type.DerivedSC2"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.DerivedSC3/" + internal.TypeExistsFlagName: []byte(""),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.DerivedSC3/derived_from":                   []byte("yorc.type.DerivedSC2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.DerivedSC3/name":                           []byte("yorc.type.DerivedSC3"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.DerivedSC4/" + internal.TypeExistsFlagName: []byte(""),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.DerivedSC4/derived_from":                   []byte("yorc.type.DerivedSC3"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.DerivedSC4/name":                           []byte("yorc.type.DerivedSC4"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/yorc.type.DerivedSC4/attributes/dsc4/default":        []byte("yorc.type.DerivedSC4"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.Root/" + internal.TypeExistsFlagName:                 []byte(""),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.Root/name":                                           []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.SoftwareComponent/" + internal.TypeExistsFlagName:    []byte(""),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.SoftwareComponent/properties/parenttypeprop/default": []byte("RootComponentTypeProp"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.SoftwareComponent/name":                        []byte("tosca.nodes.SoftwareComponent"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.SoftwareComponent/derived_from":                []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.SoftwareComponent/properties/typeprop/default": []byte("SoftwareComponentTypeProp"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.SoftwareComponent/attributes/id/default":       []byte("DefaultSoftwareComponentTypeid"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/types/tosca.nodes.SoftwareComponent/attributes/type/default":     []byte("DefaultSoftwareComponentTypeid"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/type":                        []byte("tosca.nodes.SoftwareComponent"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/requirements/0/relationship": []byte("tosca.relationships.Root"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/requirements/0/name":         []byte("req1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/requirements/1/relationship": []byte("tosca.relationships.Root"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/requirements/1/name":         []byte("req2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/requirements/2/relationship": []byte("yorc.type.1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/requirements/2/node":         []byte("Node2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/requirements/2/name":         []byte("req3"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/requirements/3/relationship": []byte("tosca.relationships.Root"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/requirements/3/name":         []byte("req4"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/properties/simple":           []byte("simple"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node1/attributes/id":               []byte("Node1-id"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node2/type":                        []byte("tosca.nodes.SoftwareComponent"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node2/requirements/0/relationship": []byte("tosca.relationships.Root"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node2/requirements/0/name":         []byte("req1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node2/requirements/1/relationship": []byte("tosca.relationships.HostedOn"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node2/requirements/1/node":         []byte("Compute1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node2/requirements/1/name":         []byte("req2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node2/properties/recurse":          []byte("Node2"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node3/type":                        []byte("tosca.nodes.SoftwareComponent"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node3/requirements/0/relationship": []byte("yorc.type.3"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node3/requirements/0/node":         []byte("Compute2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node3/requirements/0/name":         []byte("req1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node3/attributes/simple":           []byte("simple"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node4/type":                        []byte("tosca.nodes.SoftwareComponent"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node4/requirements/0/relationship": []byte("tosca.relationships.HostedOn"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node4/requirements/0/node":         []byte("Node2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/nodes/Node4/requirements/0/name":         []byte("host"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/0/attributes/id": []byte("Compute1-0"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/1/attributes/id": []byte("Compute1-1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/2/attributes/id": []byte("Compute1-2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/3/attributes/id": []byte("Compute1-3"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/4/attributes/id": []byte("Compute1-4"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/5/attributes/id": []byte("Compute1-5"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/6/attributes/id": []byte("Compute1-6"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/7/attributes/id": []byte("Compute1-7"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/8/attributes/id": []byte("Compute1-8"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/9/attributes/id": []byte("Compute1-9"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/0/attributes/recurse": []byte("Recurse-Compute1-0"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/1/attributes/recurse": []byte("Recurse-Compute1-1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/2/attributes/recurse": []byte("Recurse-Compute1-2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/3/attributes/recurse": []byte("Recurse-Compute1-3"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/4/attributes/recurse": []byte("Recurse-Compute1-4"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/5/attributes/recurse": []byte("Recurse-Compute1-5"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/6/attributes/recurse": []byte("Recurse-Compute1-6"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/7/attributes/recurse": []byte("Recurse-Compute1-7"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/8/attributes/recurse": []byte("Recurse-Compute1-8"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/9/attributes/recurse": []byte("Recurse-Compute1-9"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/0/attributes/id": []byte("Node2-0"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/1/attributes/id": []byte("Node2-1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/2/attributes/id": []byte("Node2-2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/3/attributes/id": []byte("Node2-3"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/4/attributes/id": []byte("Node2-4"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/5/attributes/id": []byte("Node2-5"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/6/attributes/id": []byte("Node2-6"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/7/attributes/id": []byte("Node2-7"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/8/attributes/id": []byte("Node2-8"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/9/attributes/id": []byte("Node2-9"),

		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/0/attributes/id":  []byte("Node1-0"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/1/attributes/id":  []byte("Node1-1"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/10/attributes/id": []byte("Node1-10"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/11/attributes/id": []byte("Node1-11"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/20/attributes/id": []byte("Node1-20"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/2/attributes/id":  []byte("Node1-2"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/3/attributes/id":  []byte("Node1-3"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/4/attributes/id":  []byte("Node1-4"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/5/attributes/id":  []byte("Node1-5"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/6/attributes/id":  []byte("Node1-6"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/7/attributes/id":  []byte("Node1-7"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/8/attributes/id":  []byte("Node1-8"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/9/attributes/id":  []byte("Node1-9"),

		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node2/ab0/attributes/id":   []byte("Node1-0"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node2/ab1/attributes/id":   []byte("Node1-1"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node2/ab10/attributes/id":  []byte("Node1-10"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node2/za11/attributes/id":  []byte("Node1-11"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node2/ab20a/attributes/id": []byte("Node1-20"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node2/ab2/attributes/id":   []byte("Node1-2"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node2/za3/attributes/id":   []byte("Node1-3"),
	})

	t.Run("groupDeploymentsNodes", func(t *testing.T) {
		t.Run("TestIsNodeTypeDerivedFrom", func(t *testing.T) {
			testIsNodeTypeDerivedFrom(t, kv)
		})
		t.Run("TestGetDefaultNbInstancesForNode", func(t *testing.T) {
			testGetDefaultNbInstancesForNode(t, kv)
		})
		t.Run("TestGetMaxNbInstancesForNode", func(t *testing.T) {
			testGetMaxNbInstancesForNode(t, kv)
		})
		t.Run("TesttestGetMinNbInstancesForNode", func(t *testing.T) {
			testGetMinNbInstancesForNode(t, kv)
		})
		t.Run("TestGetNodeProperty", func(t *testing.T) {
			testGetNodeProperty(t, kv)
		})
		// t.Run("TestGetNodeAttributes", func(t *testing.T) {
		// 	testGetNodeAttributes(t, kv)
		// })
		t.Run("TestGetNodeAttributesNames", func(t *testing.T) {
			testGetNodeAttributesNames(t, kv)
		})
		t.Run("TestGetNodeInstancesIds", func(t *testing.T) {
			testGetNodeInstancesIds(t, kv)
		})
		t.Run("TestGetTypeAttributesNames", func(t *testing.T) {
			testGetTypeAttributesNames(t, kv)
		})
	})
}

func testIsNodeTypeDerivedFrom(t *testing.T, kv *api.KV) {
	// t.Parallel()

	ok, err := IsTypeDerivedFrom(kv, "testIsNodeTypeDerivedFrom", "yorc.type.1", "tosca.relationships.HostedOn")
	require.Nil(t, err)
	require.True(t, ok)

	ok, err = IsTypeDerivedFrom(kv, "testIsNodeTypeDerivedFrom", "yorc.type.1", "tosca.relationships.ConnectsTo")
	require.Nil(t, err)
	require.False(t, ok)

	ok, err = IsTypeDerivedFrom(kv, "testIsNodeTypeDerivedFrom", "yorc.type.1", "yorc.type.1")
	require.Nil(t, err)
	require.True(t, ok)
}

func testGetDefaultNbInstancesForNode(t *testing.T, kv *api.KV) {
	// t.Parallel()

	nb, err := GetDefaultNbInstancesForNode(kv, "testGetNbInstancesForNode", "Compute1")
	require.Nil(t, err)
	require.Equal(t, uint32(10), nb)

	nb, err = GetDefaultNbInstancesForNode(kv, "testGetNbInstancesForNode", "Compute2")
	require.Nil(t, err)
	require.Equal(t, uint32(1), nb)

	_, err = GetDefaultNbInstancesForNode(kv, "testGetNbInstancesForNode", "Compute3")
	require.NotNil(t, err)

	nb, err = GetDefaultNbInstancesForNode(kv, "testGetNbInstancesForNode", "Node1")
	require.Nil(t, err)
	require.Equal(t, uint32(10), nb)

	nb, err = GetDefaultNbInstancesForNode(kv, "testGetNbInstancesForNode", "Node2")
	require.Nil(t, err)
	require.Equal(t, uint32(10), nb)

	nb, err = GetDefaultNbInstancesForNode(kv, "testGetNbInstancesForNode", "Node3")
	require.Nil(t, err)
	require.Equal(t, uint32(1), nb)
}

func testGetMaxNbInstancesForNode(t *testing.T, kv *api.KV) {
	// t.Parallel()

	nb, err := GetMaxNbInstancesForNode(kv, "testGetNbInstancesForNode", "Compute1")
	require.Nil(t, err)
	require.Equal(t, uint32(20), nb)

	nb, err = GetMaxNbInstancesForNode(kv, "testGetNbInstancesForNode", "Compute2")
	require.Nil(t, err)
	require.Equal(t, uint32(1), nb)

	_, err = GetMaxNbInstancesForNode(kv, "testGetNbInstancesForNode", "Compute3")
	fmt.Println(err)
	require.NotNil(t, err)

	nb, err = GetMaxNbInstancesForNode(kv, "testGetNbInstancesForNode", "Node1")
	require.Nil(t, err)
	require.Equal(t, uint32(20), nb)

	nb, err = GetMaxNbInstancesForNode(kv, "testGetNbInstancesForNode", "Node2")
	require.Nil(t, err)
	require.Equal(t, uint32(20), nb)

	nb, err = GetMaxNbInstancesForNode(kv, "testGetNbInstancesForNode", "Node3")
	require.Nil(t, err)
	require.Equal(t, uint32(1), nb)
}

func testGetMinNbInstancesForNode(t *testing.T, kv *api.KV) {
	// t.Parallel()

	nb, err := GetMinNbInstancesForNode(kv, "testGetNbInstancesForNode", "Compute1")
	require.Nil(t, err)
	require.Equal(t, uint32(2), nb)

	nb, err = GetMinNbInstancesForNode(kv, "testGetNbInstancesForNode", "Compute2")
	require.Nil(t, err)
	require.Equal(t, uint32(1), nb)

	_, err = GetMinNbInstancesForNode(kv, "testGetNbInstancesForNode", "Compute3")
	fmt.Println(err)
	require.NotNil(t, err)

	nb, err = GetMinNbInstancesForNode(kv, "testGetNbInstancesForNode", "Node1")
	require.Nil(t, err)
	require.Equal(t, uint32(2), nb)

	nb, err = GetMinNbInstancesForNode(kv, "testGetNbInstancesForNode", "Node2")
	require.Nil(t, err)
	require.Equal(t, uint32(2), nb)

	nb, err = GetMinNbInstancesForNode(kv, "testGetNbInstancesForNode", "Node3")
	require.Nil(t, err)
	require.Equal(t, uint32(1), nb)
}

func testGetNodeProperty(t *testing.T, kv *api.KV) {
	// t.Parallel()

	// Property is directly in node
	value, err := GetNodePropertyValue(kv, "testGetNbInstancesForNode", "Node1", "simple")
	require.Nil(t, err)
	require.NotNil(t, value)
	require.Equal(t, "simple", value.RawString())

	// Property is in a parent node we found it with recurse
	value, err = GetNodePropertyValue(kv, "testGetNbInstancesForNode", "Node4", "recurse")
	require.Nil(t, err)
	require.NotNil(t, value)
	require.Equal(t, "Node2", value.RawString())

	// Property has a default in node type
	value, err = GetNodePropertyValue(kv, "testGetNbInstancesForNode", "Node4", "typeprop")
	require.Nil(t, err)
	require.NotNil(t, value)
	require.Equal(t, "SoftwareComponentTypeProp", value.RawString())

	value, err = GetNodePropertyValue(kv, "testGetNbInstancesForNode", "Node4", "typeprop")
	require.Nil(t, err)
	require.NotNil(t, value)
	require.Equal(t, "SoftwareComponentTypeProp", value.RawString())

	// Property has a default in a parent of the node type
	value, err = GetNodePropertyValue(kv, "testGetNbInstancesForNode", "Node4", "parenttypeprop")
	require.Nil(t, err)
	require.NotNil(t, value)
	require.Equal(t, "RootComponentTypeProp", value.RawString())

	value, err = GetNodePropertyValue(kv, "testGetNbInstancesForNode", "Node4", "parenttypeprop")
	require.Nil(t, err)
	require.NotNil(t, value)
	require.Equal(t, "RootComponentTypeProp", value.RawString())

}

func testGetNodeAttributes(t *testing.T, kv *api.KV) {
	// t.Parallel()
	// Attribute is directly in node
	// res, instancesValues, err := GetNodeAttributes(kv, "testGetNbInstancesForNode", "Node3", "simple")
	// require.Nil(t, err)
	// require.True(t, res)
	// require.Len(t, instancesValues, 1)
	// require.Equal(t, "simple", instancesValues[""])

	// Attribute is directly in instances
	instancesValues, err := GetNodeAttributesValues(kv, "testGetNbInstancesForNode", "Compute1", "id")
	require.Nil(t, err)
	require.Len(t, instancesValues, 10)
	require.NotNil(t, instancesValues["0"])
	require.Equal(t, "Compute1-0", instancesValues["0"].RawString())
	require.NotNil(t, instancesValues["1"])
	require.Equal(t, "Compute1-1", instancesValues["1"].RawString())
	require.NotNil(t, instancesValues["2"])
	require.Equal(t, "Compute1-2", instancesValues["2"].RawString())
	require.NotNil(t, instancesValues["3"])
	require.Equal(t, "Compute1-3", instancesValues["3"].RawString())

	// Look at generic node attribute before parents
	// res, instancesValues, err = GetNodeAttributes(kv, "testGetNbInstancesForNode", "Node1", "id")
	// require.Nil(t, err)
	// require.True(t, res)
	// require.Len(t, instancesValues, 1)
	// require.Equal(t, "Node1-id", instancesValues[""])

	// Look at generic node type attribute before parents
	// res, instancesValues, err = GetNodeAttributes(kv, "testGetNbInstancesForNode", "Node3", "id")
	// require.Nil(t, err)
	// require.True(t, res)
	// require.Len(t, instancesValues, 1)
	// require.Equal(t, "DefaultSoftwareComponentTypeid", instancesValues[""])

	// Look at generic node type attribute before parents
	instancesValues, err = GetNodeAttributesValues(kv, "testGetNbInstancesForNode", "Node2", "type")
	require.Nil(t, err)
	require.Len(t, instancesValues, 10)
	require.NotNil(t, instancesValues["0"])
	require.Equal(t, "DefaultSoftwareComponentTypeid", instancesValues["0"].RawString())
	require.NotNil(t, instancesValues["3"])
	require.Equal(t, "DefaultSoftwareComponentTypeid", instancesValues["3"].RawString())
	require.NotNil(t, instancesValues["6"])
	require.Equal(t, "DefaultSoftwareComponentTypeid", instancesValues["6"].RawString())

	//
	instancesValues, err = GetNodeAttributesValues(kv, "testGetNbInstancesForNode", "Node2", "recurse")
	require.Nil(t, err)
	require.Len(t, instancesValues, 10)
	require.NotNil(t, instancesValues["0"])
	require.Equal(t, "Recurse-Compute1-0", instancesValues["0"].RawString())
	require.NotNil(t, instancesValues["3"])
	require.Equal(t, "Recurse-Compute1-3", instancesValues["3"].RawString())
	require.NotNil(t, instancesValues["6"])
	require.Equal(t, "Recurse-Compute1-6", instancesValues["6"].RawString())

	//
	instancesValues, err = GetNodeAttributesValues(kv, "testGetNbInstancesForNode", "Node1", "recurse")
	require.Nil(t, err)
	require.Len(t, instancesValues, 10)
	require.NotNil(t, instancesValues["0"])
	require.Equal(t, "Recurse-Compute1-0", instancesValues["0"].RawString())
	require.NotNil(t, instancesValues["3"])
	require.Equal(t, "Recurse-Compute1-3", instancesValues["3"].RawString())
	require.NotNil(t, instancesValues["6"])
	require.Equal(t, "Recurse-Compute1-6", instancesValues["6"].RawString())
}

func testGetNodeAttributesNames(t *testing.T, kv *api.KV) {
	// t.Parallel()

	attrNames, err := GetNodeAttributesNames(kv, "testGetNbInstancesForNode", "Compute1")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 3)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "ip")
	require.Contains(t, attrNames, "recurse")

	attrNames, err = GetNodeAttributesNames(kv, "testGetNbInstancesForNode", "Node1")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 2)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "type")

	attrNames, err = GetNodeAttributesNames(kv, "testGetNbInstancesForNode", "Node2")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 2)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "type")

	attrNames, err = GetNodeAttributesNames(kv, "testGetNbInstancesForNode", "Node3")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 3)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "type")
	require.Contains(t, attrNames, "simple")

	attrNames, err = GetNodeAttributesNames(kv, "testGetNbInstancesForNode", "Node4")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 2)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "type")
}

func testGetTypeAttributesNames(t *testing.T, kv *api.KV) {
	// t.Parallel()

	attrNames, err := GetTypeAttributesNames(kv, "testGetNbInstancesForNode", "tosca.nodes.SoftwareComponent")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 2)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "type")

	attrNames, err = GetTypeAttributesNames(kv, "testGetNbInstancesForNode", "yorc.type.DerivedSC1")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 2)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "type")

	attrNames, err = GetTypeAttributesNames(kv, "testGetNbInstancesForNode", "yorc.type.DerivedSC2")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 3)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "type")
	require.Contains(t, attrNames, "dsc2")

	attrNames, err = GetTypeAttributesNames(kv, "testGetNbInstancesForNode", "yorc.type.DerivedSC3")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 3)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "type")
	require.Contains(t, attrNames, "dsc2")

	attrNames, err = GetTypeAttributesNames(kv, "testGetNbInstancesForNode", "yorc.type.DerivedSC4")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 4)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "type")
	require.Contains(t, attrNames, "dsc2")
	require.Contains(t, attrNames, "dsc4")
}

func testGetNodeInstancesIds(t *testing.T, kv *api.KV) {
	// t.Parallel()

	node1ExpectedResult := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "20"}
	instancesIDs, err := GetNodeInstancesIds(kv, "testGetNodeInstancesIds", "Node1")
	require.NoError(t, err)
	require.Equal(t, node1ExpectedResult, instancesIDs)

	node2ExpectedResult := []string{"ab0", "ab1", "ab2", "ab10", "ab20a", "za3", "za11"}
	instancesIDs, err = GetNodeInstancesIds(kv, "testGetNodeInstancesIds", "Node2")
	require.NoError(t, err)
	require.Equal(t, node2ExpectedResult, instancesIDs)
}

func testNodeHasAttribute(t *testing.T, kv *api.KV, deploymentID string) {
	type args struct {
		nodeName       string
		attributeName  string
		exploreParents bool
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{"NodeHasAttribute", args{"TestCompute", "public_address", true}, true, false},
		{"NodeDoesntHaveAttribute", args{"TestCompute", "missing_attribute", true}, false, false},
		{"NodeHasAttribute", args{"TestCompute", "public_address", false}, false, false},
		{"NodeDoesntExist", args{"DoNotExist", "missing_attribute", true}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NodeHasAttribute(kv, deploymentID, tt.args.nodeName, tt.args.attributeName, tt.args.exploreParents)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeHasAttribute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NodeHasAttribute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testNodeHasProperty(t *testing.T, kv *api.KV, deploymentID string) {
	type args struct {
		nodeName       string
		propertyName   string
		exploreParents bool
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{"NodeHasProperty", args{"TestModule", "component_version", true}, true, false},
		{"NodeDoesntHaveProperty", args{"TestModule", "missing_property", true}, false, false},
		{"NodeHasPropertyOnlyOnParentType", args{"TestModule", "admin_credential", false}, false, false},
		{"NodeDoesntExist", args{"DoNotExist", "missing_property", true}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NodeHasProperty(kv, deploymentID, tt.args.nodeName, tt.args.propertyName, tt.args.exploreParents)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeHasProperty() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NodeHasProperty() = %v, want %v", got, tt.want)
			}
		})
	}
}
