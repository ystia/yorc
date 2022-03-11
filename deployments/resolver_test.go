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

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/testutil"
	"github.com/ystia/yorc/v4/tosca"
	"github.com/ystia/yorc/v4/vault"
)

func testResolver(t *testing.T) {
	log.SetDebug(true)

	t.Run("deployments/resolver/testGetOperationOutput", func(t *testing.T) {
		testGetOperationOutput(t)
	})
	t.Run("deployments/resolver/testGetOperationOutputReal", func(t *testing.T) {
		testGetOperationOutputReal(t)
	})
	t.Run("deployments/resolver/testResolveComplex", func(t *testing.T) {
		testResolveComplex(t)
	})

	t.Run("TestResolveSecret", func(t *testing.T) {
		testResolveSecret(t)
	})

}

func generateToscaValueAssignmentFromString(t *testing.T, rawFunction string) *tosca.Function {
	f, err := tosca.ParseFunction(rawFunction)
	require.Nil(t, err)
	require.NotNil(t, f)
	return f
}

func testGetOperationOutput(t *testing.T) {
	// t.Parallel()
	ctx := context.Background()
	deploymentID := testutil.BuildDeploymentID(t)
	err := StoreDeploymentDefinition(context.Background(), deploymentID, "testdata/get_op_output.yaml")
	require.Nil(t, err, "Failed to parse testdata/get_op_output.yaml definition")

	err = consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/GetOPOutputsNode/0/outputs/standard/configure/MY_OUTPUT"), "MY_RESULT")
	require.Nil(t, err)
	r := resolver(deploymentID)

	result, err := r.context(withNodeName("GetOPOutputsNode"), withInstanceName("0"), withRequirementIndex("")).resolveFunction(ctx, generateToscaValueAssignmentFromString(t, `{ get_operation_output: [ SELF, Standard, configure, MY_OUTPUT ] }`))
	require.Nil(t, err)
	require.NotNil(t, result)
	require.Equal(t, "MY_RESULT", result.RawString())

	err = consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances/GetOPOutputsNode/0/0/outputs/configure/pre_configure_source/PARTITION_NAME"), "part1")
	require.Nil(t, err)
	result, err = r.context(withNodeName("GetOPOutputsNode"), withInstanceName("0"), withRequirementIndex("0")).resolveFunction(ctx, generateToscaValueAssignmentFromString(t, `{ get_operation_output: [ SELF, Configure, pre_configure_source, PARTITION_NAME ] }`))
	require.Nil(t, err, "%+v", err)
	require.NotNil(t, result)
	require.Equal(t, "part1", result.RawString())

	result, err = r.context(withNodeName("GetOPOutputsNode"), withInstanceName("0"), withRequirementIndex("")).resolveFunction(ctx, generateToscaValueAssignmentFromString(t, `{ get_attribute: [ SELF, partition_name ] }`))
	require.Nil(t, err, "%+v", err)
	require.NotNil(t, result)
	require.Equal(t, "part1", result.RawString())

	err = consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances/GetOPOutputsNodeFirstReq/0/0/outputs/configure/pre_configure_source/PARTITION_NAME"), "part2")
	require.Nil(t, err)
	result, err = r.context(withNodeName("GetOPOutputsNodeFirstReq"), withInstanceName("0"), withRequirementIndex("")).resolveFunction(ctx, generateToscaValueAssignmentFromString(t, `{ get_attribute: [ SELF, partition_name ] }`))
	require.Nil(t, err, "%+v", err)
	require.NotNil(t, result)
	require.Equal(t, "part2", result.RawString())

	err = consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances/GetOPOutputsNodeSecondReq/1/0/outputs/configure/pre_configure_source/PARTITION_NAME"), "part3")
	require.Nil(t, err)
	result, err = r.context(withNodeName("GetOPOutputsNodeSecondReq"), withInstanceName("0"), withRequirementIndex("")).resolveFunction(ctx, generateToscaValueAssignmentFromString(t, `{ get_attribute: [ SELF, partition_name ] }`))
	require.Nil(t, err, "%+v", err)
	require.NotNil(t, result)
	require.Equal(t, "part3", result.RawString())

	outputs, err := GetOperationOutputs(ctx, deploymentID, "", "yorc.tests.nodes.GetOPOutputs", "Standard.start")
	require.Nil(t, err, "%+v", err)
	require.NotNil(t, outputs)
	require.NotNil(t, outputs["ANOTHER_OUTPUT"].AttributeMapping)
	require.Equal(t, []string{"SELF", "my_output"}, outputs["ANOTHER_OUTPUT"].AttributeMapping.Parameters)
}

func testGetOperationOutputReal(t *testing.T) {
	// t.Parallel()
	ctx := context.Background()
	deploymentID := testutil.BuildDeploymentID(t)
	err := StoreDeploymentDefinition(context.Background(), deploymentID, "testdata/get_op_output_real.yaml")
	require.Nil(t, err, "Failed to parse testdata/get_op_output_real.yaml definition: %+v", err)

	r := resolver(deploymentID)

	err = consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances/PublisherFromDockerVolume/0/0/outputs/configure/post_configure_target/HOST_PATH"), "/mypath")
	require.Nil(t, err)

	result, err := r.context(withNodeName("PublisherFromDockerVolume"), withInstanceName("0"), withRequirementIndex("")).resolveFunction(ctx, generateToscaValueAssignmentFromString(t, `{ get_attribute: [ SELF, host_path ] }`))
	require.Nil(t, err)
	require.NotNil(t, result)
	require.Equal(t, "/mypath", result.RawString())

}

func testResolveComplex(t *testing.T) {
	ctx := context.Background()
	// t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	err := StoreDeploymentDefinition(context.Background(), deploymentID, "testdata/value_assignments.yaml")
	require.Nil(t, err, "Failed to parse testdata/value_assignments.yaml definition: %+v", err)
	r := resolver(deploymentID)

	type data struct {
		nodeName         string
		instanceName     string
		requirementIndex string
	}
	type args struct {
		functionAsString string
	}
	resolverTests := []struct {
		name      string
		data      data
		args      args
		wantErr   bool
		wantFound bool
		want      string
	}{
		{"ResolveGetPropertyListAll", data{"VANode1", "", ""}, args{`{get_property: [SELF, list]}`}, false, true, `["http://","yorc",".io"]`},
		{"ResolveGetPropertyListIndex0", data{"VANode1", "", ""}, args{`{get_property: [SELF, list, 0]}`}, false, true, `http://`},
		{"ResolveGetPropertyListIndex0Alien", data{"VANode1", "", ""}, args{`{get_property: [SELF, "list[0]"]}`}, false, true, `http://`},
		{"ResolveGetPropertyMapAll", data{"VANode1", "", ""}, args{`{get_property: [SELF, map]}`}, false, true, `{"one":"1","two":"2"}`},
		{"ResolveGetPropertyMapSubKey", data{"VANode1", "", ""}, args{`{get_property: [SELF, map, one]}`}, false, true, `1`},
		{"ResolveGetPropertyMapSubKeyAlien", data{"VANode1", "", ""}, args{`{get_property: [SELF, "map.one"]}`}, false, true, `1`},
		{"ResolveGetPropertyLiteralRelationship", data{"VANode2", "0", "0"}, args{`{get_property: [SELF, "literal"]}`}, false, true, `user rel literal`},
		{"ResolveEmpty", data{"VANode1", "", ""}, args{`{get_property: [SELF, empty]}`}, false, true, ``},
		// Attribute are resolvable even if absent - returns an empty string
		{"ResolveGetAttributeWithAbsent", data{"VANode1", "0", ""}, args{`{get_attribute: [SELF, absentAttr]}`}, false, false, ``},
		{"ResolveGetRequirementAttributeWithAbsent", data{"VANode1", "0", "0"}, args{`{get_attribute: [SELF, host, absentAttr]}`}, false, false, ``},
		{"ResolveGetPropertyWithAbsent", data{"VANode1", "", ""}, args{`{get_property: [SELF, absentAttr]}`}, true, false, ``},
		{"ResolveGetRequirementPropertyWithAbsent", data{"VANode1", "", "0"}, args{`{get_property: [SELF, host, absentAttr]}`}, true, false, ``},
	}
	for _, tt := range resolverTests {
		t.Run(tt.name, func(t *testing.T) {
			f := generateToscaValueAssignmentFromString(t, tt.args.functionAsString)
			got, err := r.context(withNodeName(tt.data.nodeName), withInstanceName(tt.data.instanceName), withRequirementIndex(tt.data.requirementIndex)).resolveFunction(ctx, f)
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

func testResolveSecret(t *testing.T) {
	ctx := context.Background()
	// t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	err := StoreDeploymentDefinition(context.Background(), deploymentID, "testdata/get_secrets.yaml")
	require.Nil(t, err, "Failed to parse testdata/value_assignments.yaml definition: %+v", err)
	_, grp, store := consulutil.WithContext(context.Background())
	createNodeInstance(store, deploymentID, "Tomcat", "0")
	require.NoError(t, grp.Wait(), "Failed to create node instances")

	type data struct {
		nodeName         string
		instanceName     string
		requirementIndex string
	}
	type resolveFunc func(data data) (*TOSCAValue, error)

	opInputResolveFn := func(opName, implementedInType, requirementIndex, input string) resolveFunc {

		return func(data data) (*TOSCAValue, error) {
			op := prov.Operation{Name: opName, ImplementedInType: implementedInType}
			if requirementIndex != "" {
				op.RelOp.IsRelationshipOperation = true
				op.RelOp.RequirementIndex = requirementIndex
			}

			inputs, err := GetOperationInput(ctx, deploymentID, data.nodeName, op, input)
			if err != nil {
				return nil, err
			}
			for i := range inputs {
				if inputs[i].InstanceName == data.instanceName {
					return &TOSCAValue{Value: inputs[i].Value, IsSecret: inputs[i].IsSecret}, nil
				}
			}
			return nil, nil
		}
	}

	resolverTests := []struct {
		name        string
		data        data
		vaultClient vault.Client
		resolveFn   resolveFunc
		wantErr     bool
		wantFound   bool
		want        string
	}{
		{"ResolvePropWithoutVault", data{"JDK", "", ""}, nil, func(data data) (*TOSCAValue, error) {
			return GetNodePropertyValue(ctx, deploymentID, data.nodeName, "java_home")
		}, true, false, ""},
		{"ResolveCapabilityPropWithoutVault", data{"Tomcat", "", ""}, nil, func(data data) (*TOSCAValue, error) {
			return GetCapabilityPropertyValue(ctx, deploymentID, data.nodeName, "data_endpoint", "port")
		}, true, false, ""},
		{"ResolveAttributeWithoutVault", data{"JDK", "0", ""}, nil, func(data data) (*TOSCAValue, error) {
			return GetInstanceAttributeValue(ctx, deploymentID, data.nodeName, data.instanceName, "java_secret")
		}, true, false, ""},
		{"ResolveOperationInputWithoutVault", data{"Tomcat", "0", ""}, nil, opInputResolveFn("standard.create", "org.alien4cloud.lang.java.jdk.linux.nodes.OracleJDK", "", "JAVA_INPUT_SEC"), true, false, ""},
		{"ResolveOperationInputRelWithoutVault", data{"Tomcat", "0", "0"}, nil, opInputResolveFn("configure.post_configure_source", "org.alien4cloud.lang.java.pub.relationships.JavaSoftwareHostedOnJDK", "0", "TOMCAT_SEC"), true, false, ""},
		{"ResolvePropWithVault", data{"JDK", "", ""}, &vaultClientMock{"/secrets/myapp/javahome", "mysupersecret", []string{"java_opt1=1", "java_opt2=2"}}, func(data data) (*TOSCAValue, error) {
			return GetNodePropertyValue(ctx, deploymentID, data.nodeName, "java_home")
		}, false, true, "mysupersecret"},
		{"ResolvePropWithVaultAndNestedFn1", data{"JDK", "", ""}, &vaultClientMock{"/secrets/myapp/javahome", "mysupersecret", []string{"java_opt1=1", "java_opt2=2"}}, func(data data) (*TOSCAValue, error) {
			return GetNodePropertyValue(ctx, deploymentID, data.nodeName, "java_home2")
		}, false, true, "mysupersecret"},
		{"ResolvePropWithVaultAndNestedFn2", data{"JDK", "", ""}, &vaultClientMock{"/secrets/myapp/javahome", "mysupersecret", []string{"java_opt1=1", "java_opt2=2"}}, func(data data) (*TOSCAValue, error) {
			return GetNodePropertyValue(ctx, deploymentID, data.nodeName, "java_home3")
		}, false, true, "mysupersecret"},
		{"ResolvePropWithVaultAndNestedFn3", data{"JDK", "", ""}, &vaultClientMock{"/secrets/myapp/javahome", "mysupersecret", []string{"java_opt1=1", "java_opt2=2"}}, func(data data) (*TOSCAValue, error) {
			return GetNodePropertyValue(ctx, deploymentID, data.nodeName, "java_home4")
		}, false, true, "mysupersecret"},
		{"ResolveCapabilityPropWithoutVault", data{"Tomcat", "", ""}, &vaultClientMock{"/secrets/myapp/tomcatport", "443", []string{"tom_opt1=1", "tom_opt2=2"}}, func(data data) (*TOSCAValue, error) {
			return GetCapabilityPropertyValue(ctx, deploymentID, data.nodeName, "data_endpoint", "port")
		}, false, true, "443"},
		{"ResolveAttributeWithoutVault", data{"JDK", "0", ""}, &vaultClientMock{id: "/secrets/app/javatype", result: "java_supersecret"}, func(data data) (*TOSCAValue, error) {
			return GetInstanceAttributeValue(ctx, deploymentID, data.nodeName, data.instanceName, "java_secret")
		}, false, true, "java_supersecret"},
		{"ResolveOperationInputWithoutVault", data{"Tomcat", "0", ""}, &vaultClientMock{"/secrets/app/javatype", "tomcat_supersecret_input", []string{"ji_o"}}, opInputResolveFn("standard.create", "org.alien4cloud.lang.java.jdk.linux.nodes.OracleJDK", "", "JAVA_INPUT_SEC"), false, true, "tomcat_supersecret_input"},
		{"ResolveOperationInputRelWithoutVault", data{"Tomcat", "0", "0"}, &vaultClientMock{id: "/secrets/app/tominput", result: "java_supersecret_rel_input"}, opInputResolveFn("configure.post_configure_source", "org.alien4cloud.lang.java.pub.relationships.JavaSoftwareHostedOnJDK", "0", "TOMCAT_SEC"), false, true, "java_supersecret_rel_input"},
	}
	for _, tt := range resolverTests {
		t.Run(tt.name, func(t *testing.T) {
			DefaultVaultClient = tt.vaultClient
			got, err := tt.resolveFn(tt.data)
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
