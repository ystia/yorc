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

package ansible

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"text/template"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/prov/operations"
	yorc_testutil "github.com/ystia/yorc/v4/testutil"
)

// From now only WorkingDirectory is necessary for those tests
func GetConfig() config.Configuration {
	config := config.Configuration{
		WorkingDirectory: "work",
	}
	return config
}

func TestTemplates(t *testing.T) {
	t.Parallel()
	ec := &executionCommon{
		NodeName:               "Welcome",
		operation:              prov.Operation{Name: "standard.start"},
		Artifacts:              map[string]string{"scripts": "my_scripts"},
		OverlayPath:            "/some/local/path",
		VarInputsNames:         []string{"INSTANCE", "PORT"},
		OperationRemoteBaseDir: ".yorc/path/on/remote",
	}

	e := &executionScript{
		executionCommon: ec,
	}

	tmpl := template.New("execTest")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl = tmpl.Funcs(getExecutionScriptTemplateFnMap(ec, "", func() string { return "" }))
	tmpl, err := tmpl.Parse(shellAnsiblePlaybook)
	require.Nil(t, err)
	err = tmpl.Execute(os.Stdout, e)
	t.Log(err)
	require.Nil(t, err)
}

func testExecution(t *testing.T, srv1 *testutil.TestServer) {
	deploymentID := yorc_testutil.BuildDeploymentID(t)
	err := deployments.StoreDeploymentDefinition(context.Background(), deploymentID, "testdata/execTemplate.yml")
	require.NoError(t, err, "Can't store deployment definition")
	nodeAName := "NodeA"
	nodeBName := "NodeB"

	srv1.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeA/0/attributes/ip_address"): []byte("10.10.10.1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeA/1/attributes/ip_address"): []byte("10.10.10.2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeA/2/attributes/ip_address"): []byte("10.10.10.3"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeA/0/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeA/1/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeA/2/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.3"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeB/0/attributes/ip_address"): []byte("10.10.10.10"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeB/1/attributes/ip_address"): []byte("10.10.10.11"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeB/0/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.10"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeB/1/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.11"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeAName, "0/attributes/state"): []byte("initial"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeAName, "1/attributes/state"): []byte("initial"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeAName, "2/attributes/state"): []byte("initial"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeBName, "0/attributes/state"): []byte("initial"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeBName, "1/attributes/state"): []byte("initial"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeBName, "0/capabilities/cap/attributes/myattr"): []byte("attr-0"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeBName, "1/capabilities/cap/attributes/myattr"): []byte("attr-1"),

		path.Join(consulutil.TasksPrefix, "taskIDNotUsedForNow", "type"): []byte("0"),
	})
	t.Run("testExecutionResolveInputsOnNode", func(t *testing.T) {
		testExecutionResolveInputsOnNode(t, deploymentID, nodeAName, "yorc.types.A", "standard.create")
	})
	t.Run("testExecutionGenerateOnNode", func(t *testing.T) {
		testExecutionGenerateOnNode(t, deploymentID, nodeAName, "standard.create")
	})
	t.Run("testExecutionGenerateAnsibleConfig", func(t *testing.T) {
		testExecutionGenerateAnsibleConfig(t)
	})
	t.Run("testConcurrentExecutionGenerateAnsibleConfig", func(t *testing.T) {
		testConcurrentExecutionGenerateAnsibleConfig(t)
	})
	t.Run("testExecutionCleanup", func(t *testing.T) {
		testExecutionCleanup(t)
	})
	var operationTestCases = []string{
		"configure.pre_configure_source",
	}
	for i, operation := range operationTestCases {
		t.Run("testExecutionResolveInputsOnRelationshipSource-"+strconv.Itoa(i), func(t *testing.T) {
			testExecutionResolveInputsOnRelationshipSource(t, deploymentID, nodeAName, nodeBName, operation, "yorc.types.Rel", "connect", "SOURCE")
		})
		t.Run("testExecutionGenerateOnRelationshipSource-"+strconv.Itoa(i), func(t *testing.T) {
			testExecutionGenerateOnRelationshipSource(t, deploymentID, nodeAName, operation, "connect", "SOURCE")
		})
	}

	operationTestCases = []string{
		"configure.add_source",
	}

	for i, operation := range operationTestCases {
		t.Run("testExecutionResolveInputOnRelationshipTarget-"+strconv.Itoa(i), func(t *testing.T) {
			testExecutionResolveInputOnRelationshipTarget(t, deploymentID, nodeAName, nodeBName, operation, "yorc.types.Rel", "connect", "TARGET")
		})

		t.Run("testExecutionGenerateOnRelationshipTarget-"+strconv.Itoa(i), func(t *testing.T) {
			testExecutionGenerateOnRelationshipTarget(t, deploymentID, nodeAName, operation, "connect", "TARGET")
		})
	}
}

func testExecutionResolveInputsOnNode(t *testing.T, deploymentID, nodeName, nodeTypeName, operation string) {
	ctx := context.Background()
	op, err := operations.GetOperation(ctx, deploymentID, nodeName, operation, "", "", nil)
	require.Nil(t, err)
	execution := &executionCommon{
		deploymentID:           deploymentID,
		NodeName:               nodeName,
		operation:              op,
		isPerInstanceOperation: false,
		VarInputsNames:         make([]string, 0),
		EnvInputs:              make([]*operations.EnvInput, 0)}

	err = execution.resolveOperation(ctx)
	require.Nil(t, err)

	err = execution.resolveInputs(ctx)
	require.Nil(t, err)
	require.Len(t, execution.EnvInputs, 24)
	instanceNames := make(map[string]struct{})
	for _, envInput := range execution.EnvInputs {
		instanceNames[envInput.InstanceName+"_"+envInput.Name] = struct{}{}
		switch envInput.Name {
		case "A1", "G2":
			switch envInput.InstanceName {
			case "NodeA_0":
				require.Equal(t, "/var/www", envInput.Value)
			case "NodeA_1":
				require.Equal(t, "/var/www", envInput.Value)
			case "NodeA_2":
				require.Equal(t, "/var/www", envInput.Value)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		case "A2":

			switch envInput.InstanceName {
			case "NodeA_0":
				require.Equal(t, "10.10.10.1", envInput.Value)
			case "NodeA_1":
				require.Equal(t, "10.10.10.2", envInput.Value)
			case "NodeA_2":
				require.Equal(t, "10.10.10.3", envInput.Value)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		case "A3", "G3":
			switch envInput.InstanceName {
			case "NodeA_0":
				require.Equal(t, "", envInput.Value)
			case "NodeA_1":
				require.Equal(t, "", envInput.Value)
			case "NodeA_2":
				require.Equal(t, "", envInput.Value)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		case "A4", "G4":
			switch envInput.InstanceName {
			case "NodeA_0":
				require.Equal(t, "", envInput.Value)
			case "NodeA_1":
				require.Equal(t, "", envInput.Value)
			case "NodeA_2":
				require.Equal(t, "", envInput.Value)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		case "G1":
			switch envInput.InstanceName {
			case "NodeA_0":
				require.Equal(t, "G1", envInput.Value)
			case "NodeA_1":
				require.Equal(t, "G1", envInput.Value)
			case "NodeA_2":
				require.Equal(t, "G1", envInput.Value)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		default:
			require.Fail(t, "Unexpected input name: ", envInput.Name)
		}
	}
	require.Len(t, instanceNames, 24)
}

func compareStringsIgnoreWhitespace(t *testing.T, expected, actual string) {
	reLeadcloseWhtsp := regexp.MustCompile(`^[\s\p{Zs}]+|[\s\p{Zs}]+$`)
	reInsideWhtsp := regexp.MustCompile(`[\s\p{Zs}]{2,}`)
	expected = reLeadcloseWhtsp.ReplaceAllString(expected, "")
	expected = reInsideWhtsp.ReplaceAllString(expected, " ")
	actual = reLeadcloseWhtsp.ReplaceAllString(actual, "")
	actual = reInsideWhtsp.ReplaceAllString(actual, " ")
	// As environments var aren't in the same order, just compare string length
	require.Equal(t, len(expected), len(actual))
}

func getWrappedCommandFunc(path string) func() string {
	return func() string {
		return fmt.Sprintf("{{ ansible_env.HOME}}/%s/wrapper", path)
	}
}

func testExecutionGenerateOnNode(t *testing.T, deploymentID, nodeName, operation string) {
	op, err := operations.GetOperation(context.Background(), deploymentID, nodeName, operation, "", "", nil)
	require.Nil(t, err)
	execution, err := newExecution(context.Background(), GetConfig(), "taskIDNotUsedForNow", deploymentID, nodeName, op, nil)
	require.Nil(t, err)

	// This is bad.... Hopefully it will be temporary
	execution.(*executionScript).OperationRemoteBaseDir = "tmp"
	execution.(*executionScript).OperationRemotePath = path.Join(execution.(*executionScript).OperationRemoteBaseDir, ".yorc")
	execution.(*executionScript).ScriptToRun = "/path/to/some.sh"
	execution.(*executionScript).WrapperLocation = "/path/to/wrapper.sh"

	expectedResult := `- name: Executing script /path/to/some.sh
  hosts: all
  strategy: free
  tasks:
  - file: path="{{ ansible_env.HOME}}/tmp/.yorc" state=directory mode=0755
  - copy: src="/path/to/wrapper.sh" dest="{{ ansible_env.HOME}}/tmp/.yorc/wrapper" mode=0744
  - copy: src="/path/to/some.sh" dest="{{ ansible_env.HOME}}/tmp/.yorc" mode=0744
	- replace:
			path: "{{ ansible_env.HOME}}/tmp/.yorc/wrapper"
			regexp: "#!/usr/bin/env python"
			replace: "#!{{ ansible_python.executable}}"
  - shell: "/bin/bash -l -c {{ ansible_env.HOME}}/tmp/.yorc/wrapper"

      environment:
        NodeA_0_A1: "/var/www"
        NodeA_1_A1: "/var/www"
        NodeA_2_A1: "/var/www"
        NodeA_0_A2: "10.10.10.1"
        NodeA_1_A2: "10.10.10.2"
        NodeA_2_A2: "10.10.10.3"
        NodeA_0_A3: ""
        NodeA_1_A3: ""
        NodeA_2_A3: ""
        NodeA_0_A4: ""
        NodeA_1_A4: ""
        NodeA_2_A4: ""
        NodeA_0_G1: "G1"
        NodeA_1_G1: "G1"
        NodeA_2_G1: "G1"
        NodeA_0_G2: "/var/www"
        NodeA_1_G2: "/var/www"
        NodeA_2_G2: "/var/www"
        NodeA_0_G3: ""
        NodeA_1_G3: ""
        NodeA_2_G3: ""
        NodeA_0_G4: ""
        NodeA_1_G4: ""
        NodeA_2_G4: ""
	    DEPLOYMENT_ID: "` + deploymentID + `"
        HOST: "ComputeA"
        INSTANCES: "NodeA_0,NodeA_1,NodeA_2"
        NODE: "NodeA"
        A1: " {{A1}}"
        A2: " {{A2}}"
        A3: " {{A3}}"
        A4: " {{A4}}"
        G1: " {{G1}}"
        G2: " {{G2}}"
        G3: " {{G3}}"
        G4: " {{G4}}"
        INSTANCE: " {{INSTANCE}}"

     - fetch: src={{ ansible_env.HOME}}/tmp/.yorc/out.csv dest=/{{ansible_host}}-out.csv flat=yes

     - file: path="{{ ansible_env.HOME}}/` + execution.(*executionScript).OperationRemoteBaseDir + `" state=absent


`

	var writer bytes.Buffer
	tmpl := template.New("execTest")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl = tmpl.Funcs(getExecutionScriptTemplateFnMap(execution.(*executionScript).executionCommon, "",
		getWrappedCommandFunc(execution.(*executionScript).OperationRemotePath)))
	tmpl, err = tmpl.Parse(shellAnsiblePlaybook)
	require.Nil(t, err)
	err = tmpl.Execute(&writer, execution)
	t.Log(err)
	require.Nil(t, err)
	t.Log(writer.String())
	compareStringsIgnoreWhitespace(t, expectedResult, writer.String())
}

func testExecutionResolveInputsOnRelationshipSource(t *testing.T, deploymentID, nodeAName, nodeBName, operation, relationshipTypeName, requirementName, operationHost string) {
	ctx := context.Background()
	op, err := operations.GetOperation(ctx, deploymentID, nodeAName, operation, requirementName, operationHost, nil)
	require.Nil(t, err)
	execution := &executionCommon{
		deploymentID:           deploymentID,
		NodeName:               nodeAName,
		operation:              op,
		isPerInstanceOperation: false,
		relationshipType:       relationshipTypeName,
		VarInputsNames:         make([]string, 0),
		EnvInputs:              make([]*operations.EnvInput, 0),
		sourceNodeInstances:    []string{"0", "1", "2"},
		targetNodeInstances:    []string{"0", "1"},
	}

	err = execution.resolveInputs(ctx)
	require.Nil(t, err, "%+v", err)
	require.Len(t, execution.EnvInputs, 16)
	instanceNames := make(map[string]struct{})
	for _, envInput := range execution.EnvInputs {
		instanceNames[envInput.InstanceName+"_"+envInput.Name] = struct{}{}
		switch envInput.Name {
		case "A1", "G2":
			switch envInput.InstanceName {
			case "NodeA_0":
				require.Equal(t, "/var/www", envInput.Value)
			case "NodeA_1":
				require.Equal(t, "/var/www", envInput.Value)
			case "NodeA_2":
				require.Equal(t, "/var/www", envInput.Value)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		case "A2", "G3":

			switch envInput.InstanceName {
			case "NodeB_0":
				require.Equal(t, "10.10.10.10", envInput.Value)
			case "NodeB_1":
				require.Equal(t, "10.10.10.11", envInput.Value)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		case "G1":
			switch envInput.InstanceName {
			case "NodeA_0":
				require.Equal(t, "G1", envInput.Value)
			case "NodeA_1":
				require.Equal(t, "G1", envInput.Value)
			case "NodeA_2":
				require.Equal(t, "G1", envInput.Value)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		case "OO":
			require.Equal(t, "", envInput.Value)
		default:
			require.Fail(t, "Unexpected input name: ", envInput.Name)
		}
	}
	require.Len(t, instanceNames, 16)
}

func testExecutionGenerateOnRelationshipSource(t *testing.T, deploymentID, nodeName, operation, requirementName, operationHost string) {
	op, err := operations.GetOperation(context.Background(), deploymentID, nodeName, operation, requirementName, operationHost, nil)
	require.Nil(t, err)
	execution, err := newExecution(context.Background(), GetConfig(), "taskIDNotUsedForNow", deploymentID, nodeName, op, nil)
	require.Nil(t, err)

	// This is bad.... Hopefully it will be temporary
	execution.(*executionScript).OperationRemoteBaseDir = "tmp"
	execution.(*executionScript).OperationRemotePath = path.Join(execution.(*executionScript).OperationRemoteBaseDir, ".yorc")
	execution.(*executionScript).ScriptToRun = "/path/to/some.sh"
	execution.(*executionScript).WrapperLocation = "/path/to/wrapper.sh"

	expectedResult := `- name: Executing script /path/to/some.sh
  hosts: all
  strategy: free
  tasks:
  - file: path="{{ ansible_env.HOME}}/tmp/.yorc" state=directory mode=0755
  - copy: src="/path/to/wrapper.sh" dest="{{ ansible_env.HOME}}/tmp/.yorc/wrapper" mode=0744
  - copy: src="/path/to/some.sh" dest="{{ ansible_env.HOME}}/tmp/.yorc" mode=0744
	- replace:
			path: "{{ ansible_env.HOME}}/tmp/.yorc/wrapper"
			regexp: "#!/usr/bin/env python"
			replace: "#!{{ ansible_python.executable}}"
  - shell: "/bin/bash -l -c {{ ansible_env.HOME}}/tmp/.yorc/wrapper"
      environment:
        NodeA_0_A1: "/var/www"
        NodeA_1_A1: "/var/www"
        NodeA_2_A1: "/var/www"
        NodeB_0_A2: "10.10.10.10"
        NodeB_1_A2: "10.10.10.11"
        NodeA_0_G1: "G1"
        NodeA_1_G1: "G1"
        NodeA_2_G1: "G1"
        NodeA_0_G2: "/var/www"
        NodeA_1_G2: "/var/www"
        NodeA_2_G2: "/var/www"
        NodeB_0_G3: "10.10.10.10"
        NodeB_1_G3: "10.10.10.11"
		NodeA_0_OO: ""

		NodeA_1_OO: ""

		NodeA_2_OO: ""

        DEPLOYMENT_ID: "` + deploymentID + `"
        SOURCE_HOST: "ComputeA"
        SOURCE_INSTANCES: "NodeA_0,NodeA_1,NodeA_2"
        SOURCE_NODE: "NodeA"
        TARGET_HOST: "ComputeB"
        TARGET_INSTANCE: "NodeB_0"
        TARGET_INSTANCES: "NodeB_0,NodeB_1"
        TARGET_NODE: "NodeB"
        TARGET_CAPABILITY_NAMES: "cap"
        TARGET_CAPABILITY_NodeB_0_ATTRIBUTE_myattr: "attr-0"
        TARGET_CAPABILITY_NodeB_1_ATTRIBUTE_myattr: "attr-1"
        TARGET_CAPABILITY_PROPERTY_Cap1: "DCap1"
        TARGET_CAPABILITY_PROPERTY_Cap2: "DCap2"
        TARGET_CAPABILITY_TYPE: "yorc.types.Cap"
        TARGET_CAPABILITY_cap_NodeB_0_ATTRIBUTE_myattr: "attr-0"
        TARGET_CAPABILITY_cap_NodeB_1_ATTRIBUTE_myattr: "attr-1"
        TARGET_CAPABILITY_cap_PROPERTY_Cap1: "DCap1"
        TARGET_CAPABILITY_cap_PROPERTY_Cap2: "DCap2"
        TARGET_CAPABILITY_cap_TYPE: "yorc.types.Cap"
        A1: " {{A1}}"
        A2: " {{A2}}"
        G1: " {{G1}}"
        G2: " {{G2}}"
        G3: " {{G3}}"
    	OO: " {{OO}}"
        SOURCE_INSTANCE: " {{SOURCE_INSTANCE}}"

     - file: path="{{ ansible_env.HOME}}/` + execution.(*executionScript).OperationRemoteBaseDir + `" state=absent


`

	var writer bytes.Buffer
	tmpl := template.New("execTest")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl = tmpl.Funcs(getExecutionScriptTemplateFnMap(execution.(*executionScript).executionCommon, "",
		getWrappedCommandFunc(execution.(*executionScript).OperationRemotePath)))
	tmpl, err = tmpl.Parse(shellAnsiblePlaybook)
	require.Nil(t, err)
	err = tmpl.Execute(&writer, execution)
	t.Log(err)
	require.Nil(t, err)
	t.Log(writer.String())
	compareStringsIgnoreWhitespace(t, expectedResult, writer.String())
}

func testExecutionResolveInputOnRelationshipTarget(t *testing.T, deploymentID, nodeAName, nodeBName, operation, relationshipTypeName, requirementName, operationHost string) {
	ctx := context.Background()
	op, err := operations.GetOperation(ctx, deploymentID, nodeAName, operation, requirementName, operationHost, nil)
	require.Nil(t, err)
	execution := &executionCommon{
		deploymentID:             deploymentID,
		NodeName:                 nodeAName,
		operation:                op,
		isRelationshipTargetNode: true,
		isPerInstanceOperation:   false,
		relationshipType:         relationshipTypeName,
		VarInputsNames:           make([]string, 0),
		EnvInputs:                make([]*operations.EnvInput, 0)}

	err = execution.resolveOperation(ctx)
	require.Nil(t, err)

	err = execution.resolveInputs(ctx)
	require.Nil(t, err)
	require.Len(t, execution.EnvInputs, 12)
	instanceNames := make(map[string]struct{})
	for _, envInput := range execution.EnvInputs {
		instanceNames[envInput.InstanceName+"_"+envInput.Name] = struct{}{}
		switch envInput.Name {
		case "A1", "G2":
			switch envInput.InstanceName {
			case "NodeA_0":
				require.Equal(t, "/var/www", envInput.Value)
			case "NodeA_1":
				require.Equal(t, "/var/www", envInput.Value)
			case "NodeA_2":
				require.Equal(t, "/var/www", envInput.Value)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		case "A2", "G3":
			switch envInput.InstanceName {
			case "NodeB_0":
				require.Equal(t, "10.10.10.10", envInput.Value)
			case "NodeB_1":
				require.Equal(t, "10.10.10.11", envInput.Value)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		case "G1":
			switch envInput.InstanceName {
			case "NodeB_0":
				require.Equal(t, "G1", envInput.Value)
			case "NodeB_1":
				require.Equal(t, "G1", envInput.Value)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		default:
			require.Fail(t, "Unexpected input name: ", envInput.Name)
		}
	}
	require.Len(t, instanceNames, 12)
}

func testExecutionGenerateOnRelationshipTarget(t *testing.T, deploymentID, nodeName, operation, requirementName, operationHost string) {
	op, err := operations.GetOperation(context.Background(), deploymentID, nodeName, operation, requirementName, operationHost, nil)
	require.Nil(t, err)
	execution, err := newExecution(context.Background(), GetConfig(), "taskIDNotUsedForNow", deploymentID, nodeName, op, nil)
	require.Nil(t, err)
	// This is bad.... Hopefully it will be temporary
	execution.(*executionScript).OperationRemoteBaseDir = "tmp"
	execution.(*executionScript).OperationRemotePath = path.Join(execution.(*executionScript).OperationRemoteBaseDir, ".yorc")
	execution.(*executionScript).ScriptToRun = "/path/to/some.sh"
	execution.(*executionScript).WrapperLocation = "/path/to/wrapper.sh"

	expectedResult := `- name: Executing script /path/to/some.sh
  hosts: all
  strategy: free
  tasks:
  - file: path="{{ ansible_env.HOME}}/tmp/.yorc" state=directory mode=0755
  - copy: src="/path/to/wrapper.sh" dest="{{ ansible_env.HOME}}/tmp/.yorc/wrapper" mode=0744
	- copy: src="/path/to/some.sh" dest="{{ ansible_env.HOME}}/tmp/.yorc" mode=0744
	- replace:
			path: "{{ ansible_env.HOME}}/tmp/.yorc/wrapper"
			regexp: "#!/usr/bin/env python"
			replace: "#!{{ ansible_python.executable}}"
  - shell: "/bin/bash -l -c {{ ansible_env.HOME}}/tmp/.yorc/wrapper"
      environment:
        NodeA_0_A1: "/var/www"
        NodeA_1_A1: "/var/www"
        NodeA_2_A1: "/var/www"
        NodeB_0_A2: "10.10.10.10"
		NodeB_1_A2: "10.10.10.11"
		NodeB_0_G1: "G1"
        NodeB_1_G1: "G1"
        NodeA_0_G2: "/var/www"
        NodeA_1_G2: "/var/www"
        NodeA_2_G2: "/var/www"
        NodeB_0_G3: "10.10.10.10"
        NodeB_1_G3: "10.10.10.11"
	    DEPLOYMENT_ID: "` + deploymentID + `"
        SOURCE_HOST: "ComputeA"
        SOURCE_INSTANCES: "NodeA_0,NodeA_1,NodeA_2"
        SOURCE_NODE: "NodeA"
        TARGET_HOST: "ComputeB"
        TARGET_INSTANCES: "NodeB_0,NodeB_1"
		TARGET_NODE: "NodeB"
		TARGET_CAPABILITY_NAMES: "cap"
		TARGET_CAPABILITY_NodeB_0_ATTRIBUTE_myattr: "attr-0"
		TARGET_CAPABILITY_NodeB_1_ATTRIBUTE_myattr: "attr-1"
		TARGET_CAPABILITY_PROPERTY_Cap1: "DCap1"
		TARGET_CAPABILITY_PROPERTY_Cap2: "DCap2"
		TARGET_CAPABILITY_TYPE: "yorc.types.Cap"
		TARGET_CAPABILITY_cap_NodeB_0_ATTRIBUTE_myattr: "attr-0"
		TARGET_CAPABILITY_cap_NodeB_1_ATTRIBUTE_myattr: "attr-1"
		TARGET_CAPABILITY_cap_PROPERTY_Cap1: "DCap1"
		TARGET_CAPABILITY_cap_PROPERTY_Cap2: "DCap2"
		TARGET_CAPABILITY_cap_TYPE: "yorc.types.Cap"
        A1: " {{A1}}"
        A2: " {{A2}}"
        G1: " {{G1}}"
        G2: " {{G2}}"
        G3: " {{G3}}"
        SOURCE_INSTANCE: " {{SOURCE_INSTANCE}}"
        TARGET_INSTANCE: " {{TARGET_INSTANCE}}"
                
            
     - fetch: src={{ ansible_env.HOME}}/tmp/.yorc/out.csv dest=/{{ansible_host}}-out.csv flat=yes


     - file: path="{{ ansible_env.HOME}}/` + execution.(*executionScript).OperationRemoteBaseDir + `" state=absent


`

	var writer bytes.Buffer
	tmpl := template.New("execTest")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl = tmpl.Funcs(getExecutionScriptTemplateFnMap(execution.(*executionScript).executionCommon, "",
		getWrappedCommandFunc(execution.(*executionScript).OperationRemotePath)))
	tmpl, err = tmpl.Parse(shellAnsiblePlaybook)
	require.Nil(t, err)
	err = tmpl.Execute(&writer, execution)
	t.Log(err)
	require.Nil(t, err)
	t.Log(writer.String())
	compareStringsIgnoreWhitespace(t, expectedResult, writer.String())
}

// testConcurrentExecutionGenerateAnsibleConfig tests the concurrent execution
// of the function generating the ansible configuration file
func testConcurrentExecutionGenerateAnsibleConfig(t *testing.T) {
	nbConcurrents := 100
	var waitGroup sync.WaitGroup
	errors := make(chan error, nbConcurrents)

	// Additional config parameters to define in Ansible config
	tunables := make(map[string]string, 1000)
	for v := 0; v < 1000; v++ {
		str := strconv.Itoa(v)
		tunables[str] = str
	}

	for i := 0; i < nbConcurrents; i++ {
		waitGroup.Add(1)
		go routineGenerateAnsibleConfig(t, i, &waitGroup, tunables, errors)
	}
	waitGroup.Wait()

	// Check no error occured
	select {
	case err := <-errors:
		require.NoError(t, err, "Unexpected error generating ansible config concurrently")
	default:
	}

}

func routineGenerateAnsibleConfig(t *testing.T, id int, waitGroup *sync.WaitGroup,
	tunables map[string]string, errors chan<- error) {

	defer waitGroup.Done()

	tempdir, err := ioutil.TempDir("", path.Base(t.Name()+strconv.Itoa(id)))
	if err != nil {
		fmt.Printf("Concurrent %d failed to to create temporary directory\n", id)
		errors <- err
		return
	}
	defer os.RemoveAll(tempdir)

	ansibleAdditionalConfig := map[string]map[string]string{
		ansibleInventoryHostsHeader: tunables,
	}

	yorcConfig := config.Configuration{
		WorkingDirectory: tempdir,
		Ansible: config.Ansible{
			Config: ansibleAdditionalConfig,
		},
	}
	execution := &executionCommon{cfg: yorcConfig}
	err = execution.generateAnsibleConfigurationFile("ansiblePath", tempdir)

	if err != nil {
		fmt.Printf("Concurrent %d failed to generateAnsibleConfigurationFile\n", id)
		errors <- err
	}

}

func testExecutionGenerateAnsibleConfig(t *testing.T) {

	// Create a temporary directory where to create the ansible config file
	tempdir, err := ioutil.TempDir("", path.Base(t.Name()))
	require.NoError(t, err, "Failed to create temporary directory")
	defer os.RemoveAll(tempdir)

	// First test with no ansible config settings in Yorc server configuration
	yorcConfig := config.Configuration{
		WorkingDirectory: tempdir,
	}

	execution := &executionCommon{
		cfg: yorcConfig}

	err = execution.generateAnsibleConfigurationFile("ansiblePath", yorcConfig.WorkingDirectory)
	require.NoError(t, err, "Error generating ansible config file")

	cfgPath := path.Join(yorcConfig.WorkingDirectory, "ansible.cfg")
	resultMap, content := readAnsibleConfigSettings(t, cfgPath)

	// An additional entry for retry_files_save_path should be added to default
	// values
	assert.Equal(t,
		len(ansibleDefaultConfig[ansibleConfigDefaultsHeader])+1,
		len(resultMap[ansibleConfigDefaultsHeader]),
		"Missing entries in ansible config file, content: %q", content)

	for k, v := range ansibleDefaultConfig[ansibleConfigDefaultsHeader] {
		assert.Equal(t, resultMap[ansibleConfigDefaultsHeader][k], v,
			"Wrong ansible config value for %s", k)
	}
	v, ok := resultMap[ansibleConfigDefaultsHeader]["retry_files_save_path"]
	assert.True(t, ok, "Found no entry for retry_files_save_path in ansible config")
	assert.Equal(t, tempdir, v, "Unexpected value for retry_files_save_path value")

	// Test enabling fact caching, it should add configuration settings
	execution.CacheFacts = true
	initialConfigMapLength := len(ansibleDefaultConfig[ansibleConfigDefaultsHeader])
	err = execution.generateAnsibleConfigurationFile("ansiblePath", yorcConfig.WorkingDirectory)
	require.NoError(t, err, "Error generating ansible config file")
	resultMap, content = readAnsibleConfigSettings(t, cfgPath)
	assert.Equal(t,
		initialConfigMapLength+len(ansibleFactCaching)+1,
		len(resultMap[ansibleConfigDefaultsHeader]),
		"Missing entries in ansible config file with fact caching, content: %q", content)

	// Test with ansible config settings in Yorc server configuration
	// one of them overriding Yorc default ansible config setting
	newSettingName := "special_context_filesystems"
	newSettingValue := "nfs,vboxsf,fuse,ramfs,myspecialfs"
	overridenSettingName := "timeout"
	overridenSettingValue := "60"

	yorcConfig.Ansible.Config = map[string]map[string]string{
		ansibleConfigDefaultsHeader: map[string]string{
			newSettingName:       newSettingValue,
			overridenSettingName: overridenSettingValue,
		},
	}

	execution = &executionCommon{
		cfg: yorcConfig}

	err = execution.generateAnsibleConfigurationFile("ansiblePath", yorcConfig.WorkingDirectory)
	require.NoError(t, err, "Error generating ansible config file")

	resultMap, content = readAnsibleConfigSettings(t, cfgPath)

	assert.Equal(t,
		initialConfigMapLength+2,
		len(resultMap[ansibleConfigDefaultsHeader]),
		"Missing entries in ansible config file with user-defined values, content: %q", content)

	assert.Equal(t, newSettingValue, resultMap[ansibleConfigDefaultsHeader][newSettingName], "Wrong ansible config value for %s", newSettingName)
	assert.Equal(t, overridenSettingValue, resultMap[ansibleConfigDefaultsHeader][overridenSettingName], "Wrong ansible config value for %s", overridenSettingName)

}

type ansibleRunnerTest struct{}

func (a *ansibleRunnerTest) runAnsible(ctx context.Context, retry bool, currentInstance, ansibleRecipePath string) error {
	return nil
}

func testExecutionCleanup(t *testing.T) {

	// Create a temporary directory where to create the ansible config file
	tempdir, err := ioutil.TempDir("", path.Base(t.Name()))
	require.NoError(t, err, "Failed to create temporary directory")
	defer os.RemoveAll(tempdir)

	// First test with no ansible config settings in Yorc server configuration
	yorcConfig := config.Configuration{
		WorkingDirectory: tempdir,
	}

	e := &executionCommon{
		cfg:           yorcConfig,
		ansibleRunner: &ansibleRunnerTest{},
		deploymentID:  "testDeployment",
		taskID:        "testTaskID",
		NodeName:      "testNodeName",
		operation:     prov.Operation{Name: "standard.start"},
	}

	err = e.executeWithCurrentInstance(context.Background(), false, "0")
	require.NoError(t, err, "executeWithCurrentInstance failed")
	ansiblePath := filepath.Join(e.cfg.WorkingDirectory, "deployments", e.deploymentID, "ansible")
	var files []string
	err = filepath.Walk(ansiblePath, func(path string, info os.FileInfo, err error) error {
		if path != ansiblePath {
			files = append(files, path)
		}
		return nil
	})
	require.NoError(t, err, "Failed to check working directory content")
	require.Empty(t, files, "Expected ansible directory to be empty, found %v", files)

}

func readAnsibleConfigSettings(t *testing.T, filepath string) (map[string]map[string]string, string) {
	cfgFile, err := os.Open(filepath)
	require.NoError(t, err, "Failed to open ansible config file at %s", filepath)
	defer cfgFile.Close()

	scanner := bufio.NewScanner(cfgFile)
	resultMap := make(map[string]map[string]string)
	var fileContentBuider strings.Builder
	header := ""
	for scanner.Scan() {
		line := scanner.Text()
		fileContentBuider.WriteString(line)
		fileContentBuider.WriteString("\n")
		if squareBrackerIndex := strings.Index(line, "]"); squareBrackerIndex >= 0 {
			header = strings.TrimSpace(line[1:squareBrackerIndex])
			resultMap[header] = make(map[string]string)
		}

		if header == "" {
			continue
		}

		if equalIndex := strings.Index(line, "="); equalIndex >= 0 {
			if key := strings.TrimSpace(line[:equalIndex]); len(key) > 0 {
				fmt.Printf("key .%s.\n", key)
				value := ""
				if len(line) > equalIndex {
					value = strings.TrimSpace(line[equalIndex+1:])
				}
				fmt.Printf("value .%s.\n", value)
				resultMap[header][key] = value
			}
		}
	}

	err = scanner.Err()
	require.NoError(t, err, "Error reading ansible config file")

	return resultMap, fileContentBuider.String()
}
