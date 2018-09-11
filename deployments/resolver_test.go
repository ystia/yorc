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
	"path"
	"reflect"
	"testing"

	yaml "gopkg.in/yaml.v2"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
	"github.com/ystia/yorc/testutil"
	"github.com/ystia/yorc/tosca"
	"github.com/ystia/yorc/vault"
)

func testResolver(t *testing.T, kv *api.KV) {
	log.SetDebug(true)

	t.Run("deployments/resolver/testGetOperationOutput", func(t *testing.T) {
		testGetOperationOutput(t, kv)
	})
	t.Run("deployments/resolver/testGetOperationOutputReal", func(t *testing.T) {
		testGetOperationOutputReal(t, kv)
	})
	t.Run("deployments/resolver/testResolveComplex", func(t *testing.T) {
		testResolveComplex(t, kv)
	})

	t.Run("TestResolveSecret", func(t *testing.T) {
		testResolveSecret(t, kv)
	})

}

func generateToscaValueAssignmentFromString(t *testing.T, valueAssignment string) *tosca.ValueAssignment {
	va := &tosca.ValueAssignment{}

	err := yaml.Unmarshal([]byte(valueAssignment), va)
	require.Nil(t, err)
	// require.NotNil(t, va.Expression)
	return va
}

func testGetOperationOutput(t *testing.T, kv *api.KV) {
	// t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/get_op_output.yaml")
	require.Nil(t, err, "Failed to parse testdata/get_op_output.yaml definition")

	_, err = kv.Put(&api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/GetOPOutputsNode/0/outputs/standard/configure/MY_OUTPUT"), Value: []byte("MY_RESULT")}, nil)
	require.Nil(t, err)
	r := resolver(kv, deploymentID)

	result, err := r.context(withNodeName("GetOPOutputsNode"), withInstanceName("0"), withRequirementIndex("")).resolveFunction(generateToscaValueAssignmentFromString(t, `{ get_operation_output: [ SELF, Standard, configure, MY_OUTPUT ] }`).GetFunction())
	require.Nil(t, err)
	require.NotNil(t, result)
	require.Equal(t, "MY_RESULT", result.RawString())

	_, err = kv.Put(&api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances/GetOPOutputsNode/0/0/outputs/configure/pre_configure_source/PARTITION_NAME"), Value: []byte("part1")}, nil)
	require.Nil(t, err)
	result, err = r.context(withNodeName("GetOPOutputsNode"), withInstanceName("0"), withRequirementIndex("0")).resolveFunction(generateToscaValueAssignmentFromString(t, `{ get_operation_output: [ SELF, Configure, pre_configure_source, PARTITION_NAME ] }`).GetFunction())
	require.Nil(t, err, "%+v", err)
	require.NotNil(t, result)
	require.Equal(t, "part1", result.RawString())

	result, err = r.context(withNodeName("GetOPOutputsNode"), withInstanceName("0"), withRequirementIndex("")).resolveFunction(generateToscaValueAssignmentFromString(t, `{ get_attribute: [ SELF, partition_name ] }`).GetFunction())
	require.Nil(t, err, "%+v", err)
	require.NotNil(t, result)
	require.Equal(t, "part1", result.RawString())

	_, err = kv.Put(&api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances/GetOPOutputsNodeFirstReq/0/0/outputs/configure/pre_configure_source/PARTITION_NAME"), Value: []byte("part2")}, nil)
	require.Nil(t, err)
	result, err = r.context(withNodeName("GetOPOutputsNodeFirstReq"), withInstanceName("0"), withRequirementIndex("")).resolveFunction(generateToscaValueAssignmentFromString(t, `{ get_attribute: [ SELF, partition_name ] }`).GetFunction())
	require.Nil(t, err, "%+v", err)
	require.NotNil(t, result)
	require.Equal(t, "part2", result.RawString())

	_, err = kv.Put(&api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances/GetOPOutputsNodeSecondReq/1/0/outputs/configure/pre_configure_source/PARTITION_NAME"), Value: []byte("part3")}, nil)
	require.Nil(t, err)
	result, err = r.context(withNodeName("GetOPOutputsNodeSecondReq"), withInstanceName("0"), withRequirementIndex("")).resolveFunction(generateToscaValueAssignmentFromString(t, `{ get_attribute: [ SELF, partition_name ] }`).GetFunction())
	require.Nil(t, err, "%+v", err)
	require.NotNil(t, result)
	require.Equal(t, "part3", result.RawString())

}

func testGetOperationOutputReal(t *testing.T, kv *api.KV) {
	// t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/get_op_output_real.yaml")
	require.Nil(t, err, "Failed to parse testdata/get_op_output_real.yaml definition: %+v", err)

	r := resolver(kv, deploymentID)

	_, err = kv.Put(&api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances/PublisherFromDockerVolume/0/0/outputs/configure/post_configure_target/HOST_PATH"), Value: []byte("/mypath")}, nil)
	require.Nil(t, err)

	result, err := r.context(withNodeName("PublisherFromDockerVolume"), withInstanceName("0"), withRequirementIndex("")).resolveFunction(generateToscaValueAssignmentFromString(t, `{ get_attribute: [ SELF, host_path ] }`).GetFunction())
	require.Nil(t, err)
	require.NotNil(t, result)
	require.Equal(t, "/mypath", result.RawString())

}

func testResolveComplex(t *testing.T, kv *api.KV) {
	// t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/value_assignments.yaml")
	require.Nil(t, err, "Failed to parse testdata/value_assignments.yaml definition: %+v", err)
	r := resolver(kv, deploymentID)

	type context struct {
		nodeName         string
		instanceName     string
		requirementIndex string
	}
	type args struct {
		functionAsString string
	}
	resolverTests := []struct {
		name      string
		context   context
		args      args
		wantErr   bool
		wantFound bool
		want      string
	}{
		{"ResolveGetPropertyListAll", context{"VANode1", "", ""}, args{`{get_property: [SELF, list]}`}, false, true, `["http://","yorc",".io"]`},
		{"ResolveGetPropertyListIndex0", context{"VANode1", "", ""}, args{`{get_property: [SELF, list, 0]}`}, false, true, `http://`},
		{"ResolveGetPropertyListIndex0Alien", context{"VANode1", "", ""}, args{`{get_property: [SELF, "list[0]"]}`}, false, true, `http://`},
		{"ResolveGetPropertyMapAll", context{"VANode1", "", ""}, args{`{get_property: [SELF, map]}`}, false, true, `{"one":"1","two":"2"}`},
		{"ResolveGetPropertyMapSubKey", context{"VANode1", "", ""}, args{`{get_property: [SELF, map, one]}`}, false, true, `1`},
		{"ResolveGetPropertyMapSubKeyAlien", context{"VANode1", "", ""}, args{`{get_property: [SELF, "map.one"]}`}, false, true, `1`},
		{"ResolveGetPropertyLiteralRelationship", context{"VANode2", "0", "0"}, args{`{get_property: [SELF, "literal"]}`}, false, true, `user rel literal`},
		{"ResolveEmpty", context{"VANode1", "", ""}, args{`{get_property: [SELF, empty]}`}, false, true, ``},
		// Attribute are resolvable even if absent - returns an empty string
		{"ResolveGetAttributeWithAbsent", context{"VANode1", "0", ""}, args{`{get_attribute: [SELF, absentAttr]}`}, false, false, ``},
		{"ResolveGetRequirementAttributeWithAbsent", context{"VANode1", "0", "0"}, args{`{get_attribute: [SELF, host, absentAttr]}`}, false, false, ``},
		{"ResolveGetPropertyWithAbsent", context{"VANode1", "", ""}, args{`{get_property: [SELF, absentAttr]}`}, true, false, ``},
		{"ResolveGetRequirementPropertyWithAbsent", context{"VANode1", "", "0"}, args{`{get_property: [SELF, host, absentAttr]}`}, true, false, ``},
	}
	for _, tt := range resolverTests {
		t.Run(tt.name, func(t *testing.T) {
			va := generateToscaValueAssignmentFromString(t, tt.args.functionAsString)
			require.Equal(t, tosca.ValueAssignmentFunction, va.Type)
			got, err := r.context(withNodeName(tt.context.nodeName), withInstanceName(tt.context.instanceName), withRequirementIndex(tt.context.requirementIndex)).resolveFunction(va.GetFunction())
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveFunction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got != nil) != tt.wantFound {
				t.Errorf("resolveFunction() found = %t, wantFound %t", got != nil, tt.wantFound)
			}

			if err == nil && (got != nil) && got.RawString() != tt.want {
				t.Errorf("resolveFunction() = %q, want %q", got, tt.want)
			}
		})
	}
}

type vaultClientMock struct {
	id              string
	result          string
	expectedOptions []string
}

func (m *vaultClientMock) GetSecret(id string, options ...string) (vault.Secret, error) {
	if len(m.expectedOptions) != len(options) {
		return nil, errors.Errorf("get_secret given options %+v mismatch the expected %+v", options, m.expectedOptions)
	}
	if m.expectedOptions != nil && options != nil && !reflect.DeepEqual(m.expectedOptions, options) {
		return nil, errors.Errorf("get_secret given options %+v mismatch the expected %+v", options, m.expectedOptions)
	}
	if id != m.id {
		return nil, errors.Errorf("get_secret given id %s mismatch the expected %s", id, m.id)
	}
	return &vaultSecretMock{m.result}, nil
}

func (m *vaultClientMock) Shutdown() error {
	return nil
}

type vaultSecretMock struct {
	result string
}

func (m *vaultSecretMock) String() string {
	return m.result
}
func (m *vaultSecretMock) Raw() interface{} {
	return m.result
}

func testResolveSecret(t *testing.T, kv *api.KV) {
	// t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/get_secrets.yaml")
	require.Nil(t, err, "Failed to parse testdata/value_assignments.yaml definition: %+v", err)
	_, grp, store := consulutil.WithContext(context.Background())
	createNodeInstance(kv, store, deploymentID, "Tomcat", "0")
	require.NoError(t, grp.Wait(), "Failed to create node instances")

	type context struct {
		nodeName         string
		instanceName     string
		requirementIndex string
	}
	type resolveFunc func(ctx context) (*TOSCAValue, error)

	opInputResolveFn := func(opName, implementedInType, requirementIndex, input string) resolveFunc {

		return func(ctx context) (*TOSCAValue, error) {
			op := prov.Operation{Name: opName, ImplementedInType: implementedInType}
			if requirementIndex != "" {
				op.RelOp.IsRelationshipOperation = true
				op.RelOp.RequirementIndex = requirementIndex
			}

			inputs, err := GetOperationInput(kv, deploymentID, ctx.nodeName, op, input)
			if err != nil {
				return nil, err
			}
			for i := range inputs {
				if inputs[i].InstanceName == ctx.instanceName {
					return &TOSCAValue{Value: inputs[i].Value, IsSecret: inputs[i].IsSecret}, nil
				}
			}
			return nil, nil
		}
	}

	resolverTests := []struct {
		name        string
		context     context
		vaultClient vault.Client
		resolveFn   resolveFunc
		wantErr     bool
		wantFound   bool
		want        string
	}{
		{"ResolvePropWithoutVault", context{"JDK", "", ""}, nil, func(ctx context) (*TOSCAValue, error) {
			return GetNodePropertyValue(kv, deploymentID, ctx.nodeName, "java_home")
		}, true, false, ""},
		{"ResolveCapabilityPropWithoutVault", context{"Tomcat", "", ""}, nil, func(ctx context) (*TOSCAValue, error) {
			return GetCapabilityPropertyValue(kv, deploymentID, ctx.nodeName, "data_endpoint", "port")
		}, true, false, ""},
		{"ResolveAttributeWithoutVault", context{"JDK", "0", ""}, nil, func(ctx context) (*TOSCAValue, error) {
			return GetInstanceAttributeValue(kv, deploymentID, ctx.nodeName, ctx.instanceName, "java_secret")
		}, true, false, ""},
		{"ResolveOperationInputWithoutVault", context{"Tomcat", "0", ""}, nil, opInputResolveFn("standard.create", "org.alien4cloud.lang.java.jdk.linux.nodes.OracleJDK", "", "JAVA_INPUT_SEC"), true, false, ""},
		{"ResolveOperationInputRelWithoutVault", context{"Tomcat", "0", "0"}, nil, opInputResolveFn("configure.post_configure_source", "org.alien4cloud.lang.java.pub.relationships.JavaSoftwareHostedOnJDK", "0", "TOMCAT_SEC"), true, false, ""},
		{"ResolvePropWithVault", context{"JDK", "", ""}, &vaultClientMock{"/secrets/myapp/javahome", "mysupersecret", []string{"java_opt1=1", "java_opt2=2"}}, func(ctx context) (*TOSCAValue, error) {
			return GetNodePropertyValue(kv, deploymentID, ctx.nodeName, "java_home")
		}, false, true, "mysupersecret"},
		{"ResolveCapabilityPropWithoutVault", context{"Tomcat", "", ""}, &vaultClientMock{"/secrets/myapp/tomcatport", "443", []string{"tom_opt1=1", "tom_opt2=2"}}, func(ctx context) (*TOSCAValue, error) {
			return GetCapabilityPropertyValue(kv, deploymentID, ctx.nodeName, "data_endpoint", "port")
		}, false, true, "443"},
		{"ResolveAttributeWithoutVault", context{"JDK", "0", ""}, &vaultClientMock{id: "/secrets/app/javatype", result: "java_supersecret"}, func(ctx context) (*TOSCAValue, error) {
			return GetInstanceAttributeValue(kv, deploymentID, ctx.nodeName, ctx.instanceName, "java_secret")
		}, false, true, "java_supersecret"},
		{"ResolveOperationInputWithoutVault", context{"Tomcat", "0", ""}, &vaultClientMock{"/secrets/app/javatype", "tomcat_supersecret_input", []string{"ji_o"}}, opInputResolveFn("standard.create", "org.alien4cloud.lang.java.jdk.linux.nodes.OracleJDK", "", "JAVA_INPUT_SEC"), false, true, "tomcat_supersecret_input"},
		{"ResolveOperationInputRelWithoutVault", context{"Tomcat", "0", "0"}, &vaultClientMock{id: "/secrets/app/tominput", result: "java_supersecret_rel_input"}, opInputResolveFn("configure.post_configure_source", "org.alien4cloud.lang.java.pub.relationships.JavaSoftwareHostedOnJDK", "0", "TOMCAT_SEC"), false, true, "java_supersecret_rel_input"},
	}
	for _, tt := range resolverTests {
		t.Run(tt.name, func(t *testing.T) {
			DefaultVaultClient = tt.vaultClient
			got, err := tt.resolveFn(tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveFunction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && (got != nil) != tt.wantFound {
				t.Errorf("resolveFunction() found = %t, wantFound %t", got != nil, tt.wantFound)
			}

			if err == nil && !got.IsSecret {
				t.Error("resolveFunction() secret expected")
			}

			if err == nil && got.RawString() != tt.want {
				t.Errorf("resolveFunction() = %q, want %q", got.RawString(), tt.want)
			}
		})
	}
}
