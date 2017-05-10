package deployments

import (
	"reflect"
	"testing"

	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
)

func TestCapabilities(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)
	srv1, err := testutil.NewTestServer()
	if err != nil {
		t.Fatalf("Failed to create consul server: %v", err)
	}
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	require.Nil(t, err)

	kv := client.KV()

	srv1.PopulateKV(t, map[string][]byte{
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.type.1/derived_from": []byte("janus.type.2"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.type.1/name":         []byte("janus.type.1"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.type.2/derived_from": []byte("janus.type.3"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.type.2/name":         []byte("janus.type.2"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.type.3/name":         []byte("janus.type.3"),

		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.type.WithUndefCap/name":                   []byte("janus.type.WithUndefCap"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.type.WithUndefCap/capabilities/udef/name": []byte("udef"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.type.WithUndefCap/capabilities/udef/type": []byte("janus.capabilities.Undefined"),

		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.type.SuperScalable/name":                   []byte("janus.type.SuperScalable"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.type.SuperScalable/capabilities/sups/name": []byte("sups"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.type.SuperScalable/capabilities/sups/type": []byte("janus.capabilities.SuperScalable"),

		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.type.1/capabilities/endpoint/name": []byte("endpoint"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.type.1/capabilities/endpoint/type": []byte("tosca.capabilities.Endpoint"),

		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.type.2/capabilities/scalable/name": []byte("scalable"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.type.2/capabilities/scalable/type": []byte("tosca.capabilities.Scalable"),

		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.type.3/capabilities/binding/name": []byte("binding"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.type.3/capabilities/binding/type": []byte("tosca.capabilities.network.Bindable"),

		consulutil.DeploymentKVPrefix + "/cap1/topology/types/tosca.capabilities.network.Bindable/name":                     []byte("tosca.capabilities.network.Bindable"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/tosca.capabilities.network.Bindable/derived_from":             []byte("tosca.capabilities.Endpoint"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/tosca.capabilities.network.Bindable/attributes/bind1/default": []byte("bind1"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/tosca.capabilities.Endpoint/name":                             []byte("tosca.capabilities.Endpoint"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/tosca.capabilities.Endpoint/attributes/attr2/default":         []byte("attr2"),

		consulutil.DeploymentKVPrefix + "/cap1/topology/types/tosca.capabilities.Scalable/name":                                 []byte("tosca.capabilities.Scalable"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/tosca.capabilities.Scalable/properties/min_instances/default":     []byte("1"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/tosca.capabilities.Scalable/properties/max_instances/default":     []byte("100"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/tosca.capabilities.Scalable/properties/default_instances/default": []byte("1"),

		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.capabilities.SuperScalable/name":         []byte("janus.capabilities.SuperScalable"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/types/janus.capabilities.SuperScalable/derived_from": []byte("tosca.capabilities.Scalable"),

		consulutil.DeploymentKVPrefix + "/cap1/topology/nodes/node1/name":                                           []byte("node1"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/nodes/node1/type":                                           []byte("janus.type.1"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/nodes/node1/capabilities/scalable/properties/min_instances": []byte("10"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/nodes/node1/capabilities/endpoint/attributes/attr1":         []byte("attr1"),

		consulutil.DeploymentKVPrefix + "/cap1/topology/nodes/node2/name":                                               []byte("node2"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/nodes/node2/type":                                               []byte("janus.type.2"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/nodes/node2/capabilities/scalable/properties/default_instances": []byte("5"),

		consulutil.DeploymentKVPrefix + "/cap1/topology/nodes/node3/name": []byte("node3"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/nodes/node3/type": []byte("janus.type.3"),

		consulutil.DeploymentKVPrefix + "/cap1/topology/nodes/SuperScalableNode/name": []byte("SuperScalableNode"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/nodes/SuperScalableNode/type": []byte("janus.type.SuperScalable"),

		consulutil.DeploymentKVPrefix + "/cap1/topology/nodes/NodeWithUndefCap/name": []byte("NodeWithUndefCap"),
		consulutil.DeploymentKVPrefix + "/cap1/topology/nodes/NodeWithUndefCap/type": []byte("janus.type.WithUndefCap"),

		consulutil.DeploymentKVPrefix + "/cap1/topology/instances/node1/0/capabilities/endpoint/attributes/ip_address": []byte("0.0.0.0"),
	})

	t.Run("deployments/capabilities", func(t *testing.T) {
		t.Run("HasScalableCapability", func(t *testing.T) {
			testHasScalableCapability(t, kv)
		})
		t.Run("GetCapabilitiesOfType", func(t *testing.T) {
			testGetCapabilitiesOfType(t, kv)
		})
		t.Run("GetNodeCapabilityType", func(t *testing.T) {
			testGetNodeCapabilityType(t, kv)
		})
		t.Run("GetCapabilityProperty", func(t *testing.T) {
			testGetCapabilityProperty(t, kv)
		})
		t.Run("GetInstanceCapabilityAttribute", func(t *testing.T) {
			testGetInstanceCapabilityAttribute(t, kv)
		})

	})
}

func testHasScalableCapability(t *testing.T, kv *api.KV) {
	type args struct {
		kv           *api.KV
		deploymentID string
		nodeName     string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{"NodeHasExactCapInInheritance", args{kv, "cap1", "node1"}, true, false},
		{"NodeHasExactCapInSelfType", args{kv, "cap1", "node2"}, true, false},
		{"NodeHasNotCap", args{kv, "cap1", "node3"}, false, false},
		{"NodeHasInheritedCapInSelfType", args{kv, "cap1", "SuperScalableNode"}, true, false},
		{"NodeWithUndefCap", args{kv, "cap1", "NodeWithUndefCap"}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HasScalableCapability(tt.args.kv, tt.args.deploymentID, tt.args.nodeName)
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

func testGetCapabilitiesOfType(t *testing.T, kv *api.KV) {
	type args struct {
		kv                 *api.KV
		deploymentID       string
		typeName           string
		capabilityTypeName string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{"TypeScalableWithInheritance", args{kv, "cap1", "janus.type.1", "tosca.capabilities.Scalable"}, []string{"scalable"}, false},
		{"TypeScalableInType", args{kv, "cap1", "janus.type.2", "tosca.capabilities.Scalable"}, []string{"scalable"}, false},
		{"TypeNotFound", args{kv, "cap1", "janus.type.3", "tosca.capabilities.Scalable"}, []string{}, false},
		{"EndpointWithInheritance", args{kv, "cap1", "janus.type.1", "tosca.capabilities.Endpoint"}, []string{"endpoint", "binding"}, false},
		{"EndpointSingleWithInheritance", args{kv, "cap1", "janus.type.2", "tosca.capabilities.Endpoint"}, []string{"binding"}, false},
		{"EndpointSingle", args{kv, "cap1", "janus.type.3", "tosca.capabilities.Endpoint"}, []string{"binding"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCapabilitiesOfType(tt.args.kv, tt.args.deploymentID, tt.args.typeName, tt.args.capabilityTypeName)
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

func testGetCapabilityProperty(t *testing.T, kv *api.KV) {
	type args struct {
		kv             *api.KV
		deploymentID   string
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
		{"GetCapabilityPropertyDefinedInNode", args{kv, "cap1", "node1", "scalable", "min_instances"}, true, "10", false},
		{"GetCapabilityPropertyDefinedInNodeOfInheritedType", args{kv, "cap1", "node1", "scalable", "default_instances"}, true, "1", false},
		{"GetCapabilityPropertyDefinedAsCapTypeDefault", args{kv, "cap1", "node1", "scalable", "max_instances"}, true, "100", false},
		{"GetCapabilityPropertyUndefinedProp", args{kv, "cap1", "node1", "scalable", "udef"}, false, "", false},
		{"GetCapabilityPropertyUndefinedCap", args{kv, "cap1", "node1", "udef", "udef"}, false, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := GetCapabilityProperty(tt.args.kv, tt.args.deploymentID, tt.args.nodeName, tt.args.capabilityName, tt.args.propertyName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetCapabilityProperty() %q error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if got != tt.propFound {
				t.Fatalf("GetCapabilityProperty() %q propFound got = %v, want %v", tt.name, got, tt.propFound)
				return
			}
			if got1 != tt.propValue {
				t.Fatalf("GetCapabilityProperty() %q propValue got = %v, want %v", tt.name, got1, tt.propValue)
				return
			}
		})
	}
}

func testGetInstanceCapabilityAttribute(t *testing.T, kv *api.KV) {
	type args struct {
		kv             *api.KV
		deploymentID   string
		nodeName       string
		instanceName   string
		capabilityName string
		attributeName  string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		want1   string
		wantErr bool
	}{
		{"GetCapabilityAttributeInstancesScopped", args{kv, "cap1", "node1", "0", "endpoint", "ip_address"}, true, "0.0.0.0", false},
		{"GetCapabilityAttributeNodeScopped", args{kv, "cap1", "node1", "0", "endpoint", "attr1"}, true, "attr1", false},
		{"GetCapabilityAttributeNodeTypeScopped", args{kv, "cap1", "node1", "0", "endpoint", "attr2"}, true, "attr2", false},
		{"GetCapabilityAttributeNodeTypeScoppedInInheritance", args{kv, "cap1", "node1", "0", "binding", "bind1"}, true, "bind1", false},

		// Test cases for properties as attributes mapping
		{"GetCapabilityAttributeFromPropertyDefinedInNode", args{kv, "cap1", "node1", "0", "scalable", "min_instances"}, true, "10", false},
		{"GetCapabilityAttributeFromPropertyDefinedInNodeOfInheritedType", args{kv, "cap1", "node1", "0", "scalable", "default_instances"}, true, "1", false},
		{"GetCapabilityAttributeFromPropertyDefinedAsCapTypeDefault", args{kv, "cap1", "node1", "0", "scalable", "max_instances"}, true, "100", false},
		{"GetCapabilityAttributeFromPropertyUndefinedProp", args{kv, "cap1", "node1", "0", "scalable", "udef"}, false, "", false},
		{"GetCapabilityAttributeFromPropertyUndefinedCap", args{kv, "cap1", "node1", "0", "udef", "udef"}, false, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := GetInstanceCapabilityAttribute(tt.args.kv, tt.args.deploymentID, tt.args.nodeName, tt.args.instanceName, tt.args.capabilityName, tt.args.attributeName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetInstanceCapabilityAttribute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Fatalf("GetInstanceCapabilityAttribute() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Fatalf("GetInstanceCapabilityAttribute() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func testGetNodeCapabilityType(t *testing.T, kv *api.KV) {
	type args struct {
		kv             *api.KV
		deploymentID   string
		nodeName       string
		capabilityName string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"GetScalableCapTypeOnNode1", args{kv, "cap1", "node1", "scalable"}, "tosca.capabilities.Scalable", false},
		{"GetScalableCapTypeOnNode2", args{kv, "cap1", "node2", "scalable"}, "tosca.capabilities.Scalable", false},
		{"CapNotFoundOnNode3", args{kv, "cap1", "node3", "scalable"}, "", false},
		{"GetEndpointCapTypeOnNode1", args{kv, "cap1", "node1", "endpoint"}, "tosca.capabilities.Endpoint", false},
		{"GetEndpointCapTypeOnNode2", args{kv, "cap1", "node2", "binding"}, "tosca.capabilities.network.Bindable", false},
		{"UndefCapOnNodeWithUndefCap", args{kv, "cap1", "NodeWithUndefCap", "udef"}, "janus.capabilities.Undefined", false},
		{"CapWithInheritance", args{kv, "cap1", "SuperScalableNode", "sups"}, "janus.capabilities.SuperScalable", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetNodeCapabilityType(tt.args.kv, tt.args.deploymentID, tt.args.nodeName, tt.args.capabilityName)
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
