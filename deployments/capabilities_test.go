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
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/log"
)

func testCapabilities(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	log.SetDebug(true)

	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/capabilities.yaml")
	require.Nil(t, err)

	srv1.PopulateKV(t, map[string][]byte{
		// overwrite type to an non-existing one now we have passed StoreDeploymentDefinition verifications
		consulutil.DeploymentKVPrefix + "/" + deploymentID + "/topology/types/yorc.type.WithUndefCap/capabilities/udef/type": []byte("yorc.capabilities.Undefined"),

		consulutil.DeploymentKVPrefix + "/" + deploymentID + "/topology/instances/node1/0/capabilities/endpoint/attributes/ip_address":       []byte("0.0.0.0"),
		consulutil.DeploymentKVPrefix + "/" + deploymentID + "/topology/instances/node1/0/capabilities/endpoint/attributes/credentials/user": []byte("ubuntu"),
		consulutil.DeploymentKVPrefix + "/" + deploymentID + "/topology/instances/Compute/0/attributes/private_address":                      []byte("10.0.0.1"),
		consulutil.DeploymentKVPrefix + "/" + deploymentID + "/topology/instances/Compute/0/attributes/public_address":                       []byte("10.1.0.1"),
	})

	t.Run("groupDeploymentsCapabilities", func(t *testing.T) {
		t.Run("TestHasScalableCapability", func(t *testing.T) {
			testHasScalableCapability(t, kv, deploymentID)
		})
		t.Run("TestGetCapabilitiesOfType", func(t *testing.T) {
			testGetCapabilitiesOfType(t, kv, deploymentID)
		})
		t.Run("TestGetNodeCapabilityType", func(t *testing.T) {
			testGetNodeCapabilityType(t, kv, deploymentID)
		})
		t.Run("TestGetCapabilityProperty", func(t *testing.T) {
			testGetCapabilityProperty(t, kv, deploymentID)
		})
		t.Run("TestGetCapabilityPropertyType", func(t *testing.T) {
			testGetCapabilityPropertyType(t, kv, deploymentID)
		})
		t.Run("TestGetInstanceCapabilityAttribute", func(t *testing.T) {
			testGetInstanceCapabilityAttribute(t, kv, deploymentID)
		})
		t.Run("TestGetIPAddressFromHost", func(t *testing.T) {
			testGetIPAddressFromHost(t, kv, deploymentID)
		})
	})
}

func testHasScalableCapability(t *testing.T, kv *api.KV, deploymentID string) {
	type args struct {
		nodeName string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{"NodeHasExactCapInInheritance", args{"node1"}, true, false},
		{"NodeHasExactCapInSelfType", args{"node2"}, true, false},
		{"NodeHasNotCap", args{"node3"}, false, false},
		{"NodeHasInheritedCapInSelfType", args{"SuperScalableNode"}, true, false},
		{"NodeWithUndefCap", args{"NodeWithUndefCap"}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HasScalableCapability(kv, deploymentID, tt.args.nodeName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("HasScalableCapability() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Fatalf("HasScalableCapability() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testGetCapabilitiesOfType(t *testing.T, kv *api.KV, deploymentID string) {
	type args struct {
		typeName           string
		capabilityTypeName string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{"TypeScalableWithInheritance", args{"yorc.type.1", "tosca.capabilities.Scalable"}, []string{"scalable"}, false},
		{"TypeScalableInType", args{"yorc.type.2", "tosca.capabilities.Scalable"}, []string{"scalable"}, false},
		{"TypeNotFound", args{"yorc.type.3", "tosca.capabilities.Scalable"}, []string{}, false},
		{"EndpointWithInheritance", args{"yorc.type.1", "yorc.test.capabilities.Endpoint"}, []string{"endpoint", "binding"}, false},
		{"EndpointSingleWithInheritance", args{"yorc.type.2", "yorc.test.capabilities.Endpoint"}, []string{"binding"}, false},
		{"EndpointSingle", args{"yorc.type.3", "yorc.test.capabilities.Endpoint"}, []string{"binding"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCapabilitiesOfType(kv, deploymentID, tt.args.typeName, tt.args.capabilityTypeName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetCapabilitiesOfType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("GetCapabilitiesOfType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testGetCapabilityProperty(t *testing.T, kv *api.KV, deploymentID string) {
	type args struct {
		nodeName       string
		capabilityName string
		propertyName   string
	}
	tests := []struct {
		name      string
		args      args
		propFound bool
		propValue string
		wantErr   bool
	}{
		{"GetCapabilityPropertyDefinedInNode", args{"node1", "scalable", "min_instances"}, true, "10", false},
		{"GetCapabilityPropertyDefinedInNodeOfInheritedType", args{"node1", "scalable", "default_instances"}, true, "1", false},
		{"GetCapabilityPropertyDefinedAsCapTypeDefault", args{"node1", "scalable", "max_instances"}, true, "100", false},
		{"GetCapabilityPropertyUndefinedProp", args{"node1", "scalable", "udef"}, false, "", false},
		{"GetCapabilityPropertyUndefinedCap", args{"node1", "udef", "udef"}, false, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCapabilityPropertyValue(kv, deploymentID, tt.args.nodeName, tt.args.capabilityName, tt.args.propertyName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetCapabilityProperty() %q error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if err != nil && (got != nil) != tt.propFound {
				t.Fatalf("GetCapabilityProperty() %q propFound got = %v, want %v", tt.name, got, tt.propFound)
				return
			}
			if got != nil && got.RawString() != tt.propValue {
				t.Fatalf("GetCapabilityProperty() %q propValue got = %v, want %v", tt.name, got.RawString(), tt.propValue)
				return
			}
		})
	}
}

func testGetCapabilityPropertyType(t *testing.T, kv *api.KV, deploymentID string) {
	type args struct {
		nodeName       string
		capabilityName string
		propertyName   string
	}
	tests := []struct {
		name          string
		args          args
		propFound     bool
		propTypeValue string
		wantErr       bool
	}{
		{"GetCapabilityPropertyTypeDefinedInNode",
			args{"node1", "scalable", "min_instances"}, true, "integer", false},
		{"GetCapabilityPropertyTypeIntDefinedInNodeOfInheritedType",
			args{"node1", "scalable", "default_instances"}, true, "integer", false},
		{"GetCapabilityPropertyTypeStringDefinedInNodeOfInheritedType",
			args{"node1", "endpoint", "prop1"}, true, "string", false},
		{"GetCapabilityPropertyUndefinedProp",
			args{"node1", "scalable", "udef"}, false, "", false},
		{"GetCapabilityPropertyUndefinedCap",
			args{"node1", "udef", "udef"}, false, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propFound, propTypeValue, err := GetCapabilityPropertyType(kv, deploymentID,
				tt.args.nodeName, tt.args.capabilityName, tt.args.propertyName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetCapabilityPropertyType() %q error %v, wantErr %v",
					tt.name, err, tt.wantErr)
				return
			}
			if propFound != tt.propFound {
				t.Fatalf("GetCapabilityPropertyType() %q propFound got %v, want %v",
					tt.name, propFound, tt.propFound)
				return
			}
			if propTypeValue != tt.propTypeValue {
				t.Fatalf("GetCapabilityPropertyType() %q propTypeValue got %v, want %v",
					tt.name, propTypeValue, tt.propTypeValue)
				return
			}
		})
	}
}

func testGetInstanceCapabilityAttribute(t *testing.T, kv *api.KV, deploymentID string) {
	type args struct {
		nodeName       string
		instanceName   string
		capabilityName string
		attributeName  string
		nestedKeys     []string
	}
	tests := []struct {
		name      string
		args      args
		wantFound bool
		want      string
		wantErr   bool
	}{
		{"GetCapabilityAttributeInstancesScopped", args{"node1", "0", "endpoint", "ip_address", nil}, true, "0.0.0.0", false},
		{"GetCapabilityAttributeNodeScopped", args{"node1", "0", "endpoint", "attr1", nil}, true, "attr1", false},
		{"GetCapabilityAttributeNodeTypeScopped", args{"node1", "0", "endpoint", "attr2", nil}, true, "attr2", false},
		{"GetCapabilityAttributeNodeTypeScoppedInInheritance", args{"node1", "0", "binding", "bind1", nil}, true, "bind1", false},

		// Test cases for properties as attributes mapping
		{"GetCapabilityAttributeFromPropertyDefinedInNode", args{"node1", "0", "scalable", "min_instances", nil}, true, "10", false},
		{"GetCapabilityAttributeFromPropertyDefinedInNodeOfInheritedType", args{"node1", "0", "scalable", "default_instances", nil}, true, "1", false},
		{"GetCapabilityAttributeFromPropertyDefinedAsCapTypeDefault", args{"node1", "0", "scalable", "max_instances", nil}, true, "100", false},
		{"GetCapabilityAttributeFromPropertyUndefinedProp", args{"node1", "0", "scalable", "udef", nil}, false, "", false},
		{"GetCapabilityAttributeFromPropertyUndefinedCap", args{"node1", "0", "udef", "udef", nil}, false, "", false},

		// Test cases for properties as attributes mapping
		{"GetCapabilityAttributeKeyFromPropertyDefinedInInstance", args{"node1", "0", "endpoint", "credentials", []string{"user"}}, true, "ubuntu", false},
		{"GetEmptyCapabilityAttributeKeyFromNotSetProperty", args{"node1", "0", "endpoint", "credentials", []string{"token"}}, true, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetInstanceCapabilityAttributeValue(kv, deploymentID, tt.args.nodeName, tt.args.instanceName, tt.args.capabilityName, tt.args.attributeName, tt.args.nestedKeys...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetInstanceCapabilityAttribute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && (got != nil) != tt.wantFound {
				t.Fatalf("GetInstanceCapabilityAttribute() got = %v, wantFound %v", got, tt.wantFound)
			}
			if got != nil && got.RawString() != tt.want {
				t.Fatalf("GetInstanceCapabilityAttribute() got1 = %v, want %v", got.RawString(), tt.want)
			}
		})
	}
}

func testGetNodeCapabilityType(t *testing.T, kv *api.KV, deploymentID string) {
	type args struct {
		nodeName       string
		capabilityName string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"GetScalableCapTypeOnNode1", args{"node1", "scalable"}, "yorc.test.capabilities.Scalable", false},
		{"GetScalableCapTypeOnNode2", args{"node2", "scalable"}, "yorc.test.capabilities.Scalable", false},
		{"CapNotFoundOnNode3", args{"node3", "scalable"}, "", false},
		{"GetEndpointCapTypeOnNode1", args{"node1", "endpoint"}, "yorc.test.capabilities.Endpoint", false},
		{"GetEndpointCapTypeOnNode2", args{"node2", "binding"}, "yorc.test.capabilities.network.Bindable", false},
		{"UndefCapOnNodeWithUndefCap", args{"NodeWithUndefCap", "udef"}, "yorc.capabilities.Undefined", false},
		{"CapWithInheritance", args{"SuperScalableNode", "sups"}, "yorc.capabilities.SuperScalable", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetNodeCapabilityType(kv, deploymentID, tt.args.nodeName, tt.args.capabilityName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetNodeCapabilityType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Fatalf("GetNodeCapabilityType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testGetCapabilityProperties(t *testing.T, kv *api.KV) {
	// t.Parallel()
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/capabilities_properties.yaml")
	require.Nil(t, err)
	value, err := GetCapabilityPropertyValue(kv, deploymentID, "Tomcat", "data_endpoint", "port")
	require.NotNil(t, value)
	require.Equal(t, "8088", value.RawString())

	value, err = GetCapabilityPropertyValue(kv, deploymentID, "Tomcat", "data_endpoint", "protocol")
	require.NotNil(t, value)
	require.Equal(t, "http", value.RawString())

	value, err = GetCapabilityPropertyValue(kv, deploymentID, "Tomcat", "data_endpoint", "network_name")
	require.NotNil(t, value)
	require.Equal(t, "PRIVATE", value.RawString())
}

func testGetIPAddressFromHost(t *testing.T, kv *api.KV, deploymentID string) {
	type args struct {
		hostName       string
		hostInstance   string
		nodeName       string
		instanceName   string
		capabilityName string
	}
	tests := []struct {
		name    string
		args    args
		want    *TOSCAValue
		wantErr bool
	}{
		{"TestGetPrivateEndpointAddress", args{"Compute", "0", "EPCapNodePrivate", "0", "myep"}, &TOSCAValue{"10.0.0.1", false}, false},
		{"TestGetPublicEndpointAddress", args{"Compute", "0", "EPCapNodePublic", "0", "myep"}, &TOSCAValue{"10.1.0.1", false}, false},
		{"TestGetNamedNetworkEndpointAddress", args{"Compute", "0", "EPCapNodeNamedNet", "0", "myep"}, &TOSCAValue{"10.0.0.1", false}, false},
		{"TestOnNodeThatDoesNotExist", args{"", "", "nodenotexists", "0", "endpoint"}, nil, true},
		// TODO(loicalbertin) no error on this?
		{"TestOnNodeNotHosted", args{"", "", "node1", "0", "endpoint"}, nil, true},
		// Defaults to the private_address anyway even if endpoint doesn't exit
		// TODO(loicalbertin) is that a good idea?
		{"TestEndpointDoesntExist", args{"Compute", "0", "EPCapNodeNamedNet", "0", "myendpoint"}, &TOSCAValue{"10.0.0.1", false}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getIPAddressFromHost(kv, deploymentID, tt.args.hostName, tt.args.hostInstance, tt.args.nodeName, tt.args.instanceName, tt.args.capabilityName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getIPAddressFromHost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getIPAddressFromHost() = %v, want %v", got, tt.want)
			}
		})
	}
}
