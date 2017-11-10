package ansible

import (
	"bytes"
	"context"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"text/template"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/prov"
	"novaforge.bull.com/starlings-janus/janus/prov/operations"
	janus_testutil "novaforge.bull.com/starlings-janus/janus/testutil"
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
		NodeName:            "Welcome",
		operation:           prov.Operation{Name: "tosca.interfaces.node.lifecycle.standard.start"},
		Artifacts:           map[string]string{"scripts": "my_scripts"},
		OverlayPath:         "/some/local/path",
		VarInputsNames:      []string{"INSTANCE", "PORT"},
		OperationRemotePath: ".janus/path/on/remote",
	}

	e := &executionScript{
		executionCommon: ec,
	}

	funcMap := template.FuncMap{
		// The name "path" is what the function will be called in the template text.
		"path": filepath.Dir,
	}

	tmpl := template.New("execTest")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl = tmpl.Funcs(funcMap)
	tmpl, err := tmpl.Parse(shellAnsiblePlaybook)
	require.Nil(t, err)
	err = tmpl.Execute(os.Stdout, e)
	t.Log(err)
	require.Nil(t, err)
}

func testExecution(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	deploymentID := janus_testutil.BuildDeploymentID(t)
	err := deployments.StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/execTemplate.yml")
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
	})

	t.Run("testExecutionResolveInputsOnNode", func(t *testing.T) {
		testExecutionResolveInputsOnNode(t, kv, deploymentID, nodeAName, "janus.types.A", "tosca.interfaces.node.lifecycle.standard.create")
	})
	t.Run("testExecutionGenerateOnNode", func(t *testing.T) {
		testExecutionGenerateOnNode(t, kv, deploymentID, nodeAName, "tosca.interfaces.node.lifecycle.standard.create")
	})

	var operationTestCases = []string{
		"tosca.interfaces.node.lifecycle.configure.pre_configure_source/1",
		"tosca.interfaces.node.lifecycle.configure.pre_configure_source/connect/" + nodeBName,
	}
	for i, operation := range operationTestCases {
		t.Run("testExecutionResolveInputsOnRelationshipSource-"+strconv.Itoa(i), func(t *testing.T) {
			testExecutionResolveInputsOnRelationshipSource(t, kv, deploymentID, nodeAName, nodeBName, operation, "janus.types.Rel")
		})
		t.Run("testExecutionGenerateOnRelationshipSource-"+strconv.Itoa(i), func(t *testing.T) {
			testExecutionGenerateOnRelationshipSource(t, kv, deploymentID, nodeAName, operation)
		})
	}

	operationTestCases = []string{
		"tosca.interfaces.node.lifecycle.configure.add_source/1",
		"tosca.interfaces.node.lifecycle.configure.add_source/connect/" + nodeBName,
	}

	for i, operation := range operationTestCases {
		t.Run("testExecutionResolveInputOnRelationshipTarget-"+strconv.Itoa(i), func(t *testing.T) {
			testExecutionResolveInputOnRelationshipTarget(t, kv, deploymentID, nodeAName, nodeBName, operation, "janus.types.Rel")
		})

		t.Run("testExecutionGenerateOnRelationshipTarget-"+strconv.Itoa(i), func(t *testing.T) {
			testExecutionGenerateOnRelationshipTarget(t, kv, deploymentID, nodeAName, operation)
		})
	}
}

func testExecutionResolveInputsOnNode(t *testing.T, kv *api.KV, deploymentID, nodeName, nodeTypeName, operation string) {
	op, err := operations.GetOperation(kv, deploymentID, nodeName, operation)
	require.Nil(t, err)
	execution := &executionCommon{kv: kv,
		deploymentID:           deploymentID,
		NodeName:               nodeName,
		operation:              op,
		OperationPath:          path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create"),
		isPerInstanceOperation: false,
		VarInputsNames:         make([]string, 0),
		EnvInputs:              make([]*operations.EnvInput, 0)}

	err = execution.resolveOperation()
	require.Nil(t, err)

	err = execution.resolveInputs()
	require.Nil(t, err)
	require.Len(t, execution.EnvInputs, 9)
	instanceNames := make(map[string]struct{})
	for _, envInput := range execution.EnvInputs {
		instanceNames[envInput.InstanceName+"_"+envInput.Name] = struct{}{}
		switch envInput.Name {
		case "A1":
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
		case "A3":
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
		default:
			require.Fail(t, "Unexpected input name: ", envInput.Name)
		}
	}
	require.Len(t, instanceNames, 9)
}

func compareStringsIgnoreWhitespace(t *testing.T, expected, actual string) {
	reLeadcloseWhtsp := regexp.MustCompile(`^[\s\p{Zs}]+|[\s\p{Zs}]+$`)
	reInsideWhtsp := regexp.MustCompile(`[\s\p{Zs}]{2,}`)
	expected = reLeadcloseWhtsp.ReplaceAllString(expected, "")
	expected = reInsideWhtsp.ReplaceAllString(expected, " ")
	actual = reLeadcloseWhtsp.ReplaceAllString(actual, "")
	actual = reInsideWhtsp.ReplaceAllString(actual, " ")
	require.Equal(t, expected, actual)
}

func testExecutionGenerateOnNode(t *testing.T, kv *api.KV, deploymentID, nodeName, operation string) {
	op, err := operations.GetOperation(kv, deploymentID, nodeName, operation)
	require.Nil(t, err)
	execution, err := newExecution(kv, GetConfig(), "taskIDNotUsedForNow", deploymentID, nodeName, op)
	require.Nil(t, err)

	// This is bad.... Hopefully it will be temporary
	execution.(*executionScript).OperationRemotePath = "tmp"

	expectedResult := `- name: Executing script {{ script_to_run }}
  hosts: all
  strategy: free
  tasks:
    - file: path="{{ ansible_env.HOME}}/tmp" state=directory mode=0755

    - copy: src="{{ script_to_run }}" dest="{{ ansible_env.HOME}}/tmp" mode=0744


    - shell: "{{ ansible_env.HOME}}/tmp/create.sh"
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
	    DEPLOYMENT_ID: "` + deploymentID + `"
        HOST: "ComputeA"
        INSTANCES: "NodeA_0,NodeA_1,NodeA_2"
        NODE: "NodeA"
        A1: "{{A1}}"
        A2: "{{A2}}"
        A3: "{{A3}}"
        INSTANCE: "{{INSTANCE}}"

     - file: path="{{ ansible_env.HOME}}/` + execution.(*executionScript).OperationRemotePath + `" state=absent


`

	funcMap := template.FuncMap{
		// The name "path" is what the function will be called in the template text.
		"path": filepath.Dir,
	}
	var writer bytes.Buffer
	tmpl := template.New("execTest")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl = tmpl.Funcs(funcMap)
	tmpl, err = tmpl.Parse(shellAnsiblePlaybook)
	require.Nil(t, err)
	err = tmpl.Execute(&writer, execution)
	t.Log(err)
	require.Nil(t, err)
	t.Log(writer.String())
	compareStringsIgnoreWhitespace(t, expectedResult, writer.String())
}

func testExecutionResolveInputsOnRelationshipSource(t *testing.T, kv *api.KV, deploymentID, nodeAName, nodeBName, operation, relationshipTypeName string) {
	op, err := operations.GetOperation(kv, deploymentID, nodeAName, operation)
	require.Nil(t, err)
	execution := &executionCommon{kv: kv,
		deploymentID:           deploymentID,
		NodeName:               nodeAName,
		operation:              op,
		OperationPath:          path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/configure/pre_configure_source"),
		isPerInstanceOperation: false,
		relationshipType:       relationshipTypeName,
		VarInputsNames:         make([]string, 0),
		EnvInputs:              make([]*operations.EnvInput, 0),
		sourceNodeInstances:    []string{"0", "1", "2"},
		targetNodeInstances:    []string{"0", "1"},
	}

	err = execution.resolveInputs()
	require.Nil(t, err, "%+v", err)
	require.Len(t, execution.EnvInputs, 5)
	instanceNames := make(map[string]struct{})
	for _, envInput := range execution.EnvInputs {
		instanceNames[envInput.InstanceName+"_"+envInput.Name] = struct{}{}
		switch envInput.Name {
		case "A1":
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
			case "NodeB_0":
				require.Equal(t, "10.10.10.10", envInput.Value)
			case "NodeB_1":
				require.Equal(t, "10.10.10.11", envInput.Value)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		default:
			require.Fail(t, "Unexpected input name: ", envInput.Name)
		}
	}
	require.Len(t, instanceNames, 5)
}

func testExecutionGenerateOnRelationshipSource(t *testing.T, kv *api.KV, deploymentID, nodeName, operation string) {
	op, err := operations.GetOperation(kv, deploymentID, nodeName, operation)
	require.Nil(t, err)
	execution, err := newExecution(kv, GetConfig(), "taskIDNotUsedForNow", deploymentID, nodeName, op)
	require.Nil(t, err)

	// This is bad.... Hopefully it will be temporary
	execution.(*executionScript).OperationRemotePath = "tmp"

	expectedResult := `- name: Executing script {{ script_to_run }}
  hosts: all
  strategy: free
  tasks:
    - file: path="{{ ansible_env.HOME}}/tmp" state=directory mode=0755

    - copy: src="{{ script_to_run }}" dest="{{ ansible_env.HOME}}/tmp" mode=0744


    - shell: "{{ ansible_env.HOME}}/tmp/pre_configure_source.sh"
      environment:
        NodeA_0_A1: "/var/www"
        NodeA_1_A1: "/var/www"
        NodeA_2_A1: "/var/www"
        NodeB_0_A2: "10.10.10.10"
        NodeB_1_A2: "10.10.10.11"
	    DEPLOYMENT_ID: "` + deploymentID + `"
        SOURCE_HOST: "ComputeA"
        SOURCE_INSTANCES: "NodeA_0,NodeA_1,NodeA_2"
        SOURCE_NODE: "NodeA"
        TARGET_HOST: "ComputeB"
        TARGET_INSTANCE: "NodeB_0"
        TARGET_INSTANCES: "NodeB_0,NodeB_1"
        TARGET_NODE: "NodeB"
        A1: "{{A1}}"
        A2: "{{A2}}"
        SOURCE_INSTANCE: "{{SOURCE_INSTANCE}}"

     - file: path="{{ ansible_env.HOME}}/` + execution.(*executionScript).OperationRemotePath + `" state=absent


`

	funcMap := template.FuncMap{
		// The name "path" is what the function will be called in the template text.
		"path": filepath.Dir,
	}
	var writer bytes.Buffer
	tmpl := template.New("execTest")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl = tmpl.Funcs(funcMap)
	tmpl, err = tmpl.Parse(shellAnsiblePlaybook)
	require.Nil(t, err)
	err = tmpl.Execute(&writer, execution)
	t.Log(err)
	require.Nil(t, err)
	t.Log(writer.String())
	compareStringsIgnoreWhitespace(t, expectedResult, writer.String())
}

func testExecutionResolveInputOnRelationshipTarget(t *testing.T, kv *api.KV, deploymentID, nodeAName, nodeBName, operation, relationshipTypeName string) {
	op, err := operations.GetOperation(kv, deploymentID, nodeAName, operation)
	require.Nil(t, err)
	execution := &executionCommon{kv: kv,
		deploymentID:             deploymentID,
		NodeName:                 nodeAName,
		operation:                op,
		OperationPath:            path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/configure/add_source"),
		isRelationshipTargetNode: true,
		isPerInstanceOperation:   false,
		relationshipType:         relationshipTypeName,
		VarInputsNames:           make([]string, 0),
		EnvInputs:                make([]*operations.EnvInput, 0)}

	err = execution.resolveOperation()
	require.Nil(t, err)

	err = execution.resolveInputs()
	require.Nil(t, err)
	require.Len(t, execution.EnvInputs, 5)
	instanceNames := make(map[string]struct{})
	for _, envInput := range execution.EnvInputs {
		instanceNames[envInput.InstanceName+"_"+envInput.Name] = struct{}{}
		switch envInput.Name {
		case "A1":
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
			case "NodeB_0":
				require.Equal(t, "10.10.10.10", envInput.Value)
			case "NodeB_1":
				require.Equal(t, "10.10.10.11", envInput.Value)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		default:
			require.Fail(t, "Unexpected input name: ", envInput.Name)
		}
	}
	require.Len(t, instanceNames, 5)
}

func testExecutionGenerateOnRelationshipTarget(t *testing.T, kv *api.KV, deploymentID, nodeName, operation string) {
	op, err := operations.GetOperation(kv, deploymentID, nodeName, operation)
	require.Nil(t, err)
	execution, err := newExecution(kv, GetConfig(), "taskIDNotUsedForNow", deploymentID, nodeName, op)
	require.Nil(t, err)
	// This is bad.... Hopefully it will be temporary
	execution.(*executionScript).OperationRemotePath = "tmp"

	expectedResult := `- name: Executing script {{ script_to_run }}
  hosts: all
  strategy: free
  tasks:
    - file: path="{{ ansible_env.HOME}}/tmp" state=directory mode=0755

    - copy: src="{{ script_to_run }}" dest="{{ ansible_env.HOME}}/tmp" mode=0744


    - shell: "{{ ansible_env.HOME}}/tmp/add_source.sh"
      environment:
        NodeA_0_A1: "/var/www"
        NodeA_1_A1: "/var/www"
        NodeA_2_A1: "/var/www"
        NodeB_0_A2: "10.10.10.10"
        NodeB_1_A2: "10.10.10.11"
	    DEPLOYMENT_ID: "` + deploymentID + `"
        SOURCE_HOST: "ComputeA"
        SOURCE_INSTANCES: "NodeA_0,NodeA_1,NodeA_2"
        SOURCE_NODE: "NodeA"
        TARGET_HOST: "ComputeB"
        TARGET_INSTANCES: "NodeB_0,NodeB_1"
        TARGET_NODE: "NodeB"
        A1: "{{A1}}"
        A2: "{{A2}}"
        SOURCE_INSTANCE: "{{SOURCE_INSTANCE}}"
        TARGET_INSTANCE: "{{TARGET_INSTANCE}}"

     - file: path="{{ ansible_env.HOME}}/` + execution.(*executionScript).OperationRemotePath + `" state=absent


`

	funcMap := template.FuncMap{
		// The name "path" is what the function will be called in the template text.
		"path": filepath.Dir,
	}
	var writer bytes.Buffer
	tmpl := template.New("execTest")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl = tmpl.Funcs(funcMap)
	tmpl, err = tmpl.Parse(shellAnsiblePlaybook)
	require.Nil(t, err)
	err = tmpl.Execute(&writer, execution)
	t.Log(err)
	require.Nil(t, err)
	t.Log(writer.String())
	compareStringsIgnoreWhitespace(t, expectedResult, writer.String())
}
