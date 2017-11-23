package ansible

import (
	"bytes"
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
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
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
		NodeName:               "Welcome",
		operation:              prov.Operation{Name: "tosca.interfaces.node.lifecycle.standard.start"},
		Artifacts:              map[string]string{"scripts": "my_scripts"},
		OverlayPath:            "/some/local/path",
		VarInputsNames:         []string{"INSTANCE", "PORT"},
		OperationRemoteBaseDir: ".janus/path/on/remote",
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

func testExecutionOnNode(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()

	deploymentID := janus_testutil.BuildDeploymentID(t)
	nodeName := "NodeA"
	nodeTypeName := "janus.types.A"
	operation := "tosca.interfaces.node.lifecycle.standard.create"

	srv1.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/implementation_artifacts_extensions/sh"):         []byte("tosca.artifacts.Implementation.Bash"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types/tosca.artifacts.Implementation.Bash/name"): []byte("tosca.artifacts.Implementation.Bash"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "name"):                                                        []byte(nodeTypeName),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/name"):                             []byte("create"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A1/name"):                   []byte("A1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A1/expression"):             []byte("get_property: [SELF, document_root]"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A3/name"):                   []byte("A3"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A3/expression"):             []byte("get_property: [SELF, empty]"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A2/name"):                   []byte("A2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A2/expression"):             []byte("get_attribute: [HOST, ip_address]"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/implementation/primary"):           []byte("/tmp/.janus/create.sh"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/implementation/dependencies"):      []byte(""),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A1/is_property_definition"): []byte("false"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A3/is_property_definition"): []byte("false"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeTypeName, "interfaces/standard/create/inputs/A2/is_property_definition"): []byte("false"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types/tosca.nodes.Compute/name"): []byte("tosca.nodes.Compute"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "type"): []byte(nodeTypeName),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "properties/document_root"):    []byte("/var/www"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "properties/empty"):            []byte(""),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements/0/capability"):   []byte("tosca.capabilities.Container"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements/0/name"):         []byte("host"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements/0/node"):         []byte("Compute"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements/0/relationship"): []byte("tosca.relationships.HostedOn"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes/Compute/type"):                             []byte("tosca.nodes.Compute"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/Compute/0/attributes/ip_address"):      []byte("10.10.10.1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/Compute/1/attributes/ip_address"):      []byte("10.10.10.2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/Compute/2/attributes/ip_address"):      []byte("10.10.10.3"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/Compute/0/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/Compute/1/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/Compute/2/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.3"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, "0/attributes/state"): []byte("initial"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, "1/attributes/state"): []byte("initial"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, "2/attributes/state"): []byte("initial"),
	})

	t.Run("testExecutionResolveInputsOnNode", func(t *testing.T) {
		testExecutionResolveInputsOnNode(t, kv, deploymentID, nodeName, nodeTypeName, operation)
	})
	t.Run("testExecutionGenerateOnNode", func(t *testing.T) {
		testExecutionGenerateOnNode(t, kv, deploymentID, nodeName, operation)
	})

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
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_1":
				require.Equal(t, "/var/www", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_2":
				require.Equal(t, "/var/www", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		case "A2":

			switch envInput.InstanceName {
			case "NodeA_0":
				require.Equal(t, "10.10.10.1", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_1":
				require.Equal(t, "10.10.10.2", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_2":
				require.Equal(t, "10.10.10.3", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		case "A3":
			switch envInput.InstanceName {
			case "NodeA_0":
				require.Equal(t, "", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_1":
				require.Equal(t, "", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_2":
				require.Equal(t, "", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
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
	execution.(*executionScript).OperationRemoteBaseDir = "tmp"
	execution.(*executionScript).OperationRemotePath = path.Join(execution.(*executionScript).OperationRemoteBaseDir, ".janus")

	expectedResult := `- name: Executing script {{ script_to_run }}
  hosts: all
  strategy: free
  tasks:
    - file: path="{{ ansible_env.HOME}}/tmp/.janus" state=directory mode=0755

    - copy: src="{{ script_to_run }}" dest="{{ ansible_env.HOME}}/tmp/.janus" mode=0744


    - shell: "{{ ansible_env.HOME}}/tmp/.janus/create.sh"
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
        HOST: "Compute"
        INSTANCES: "NodeA_0,NodeA_1,NodeA_2"
        NODE: "NodeA"
        A1: "{{A1}}"
        A2: "{{A2}}"
        A3: "{{A3}}"
        INSTANCE: "{{INSTANCE}}"

     - file: path="{{ ansible_env.HOME}}/` + execution.(*executionScript).OperationRemoteBaseDir + `" state=absent


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

func testExecutionOnRelationshipSource(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()
	log.SetDebug(true)

	deploymentID := janus_testutil.BuildDeploymentID(t)
	nodeAName := "NodeA"
	relationshipTypeName := "janus.types.Rel"
	nodeBName := "NodeB"

	var operationTestCases = []string{
		"tosca.interfaces.node.lifecycle.Configure.pre_configure_source/1",
		"tosca.interfaces.node.lifecycle.Configure.pre_configure_source/connect/" + nodeBName,
	}

	srv1.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/implementation_artifacts_extensions/sh"):         []byte("tosca.artifacts.Implementation.Bash"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types/tosca.artifacts.Implementation.Bash/name"): []byte("tosca.artifacts.Implementation.Bash"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types/janus.types.A/name"):                                                                                  []byte("janus.types.A"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "name"):                                                                       []byte(relationshipTypeName),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source/name"):                             []byte("pre_configure_source"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source/inputs/A1/name"):                   []byte("A1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source/inputs/A1/expression"):             []byte("get_property: [SOURCE, document_root]"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source/inputs/A2/name"):                   []byte("A2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source/inputs/A2/expression"):             []byte("get_attribute: [TARGET, ip_address]"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source/implementation/primary"):           []byte("/tmp/.janus/pre_configure_source.sh"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source/implementation/dependencies"):      []byte(""),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source/inputs/A1/is_property_definition"): []byte("false"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source/inputs/A2/is_property_definition"): []byte("false"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types/tosca.nodes.Compute/name"): []byte("tosca.nodes.Compute"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types/janus.types.B/name"): []byte("janus.types.B"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeAName, "properties/document_root"): []byte("/var/www"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeAName, "type"):                     []byte("janus.types.A"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeAName, "requirements/0/capability"):   []byte("tosca.capabilities.Container"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeAName, "requirements/0/name"):         []byte("host"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeAName, "requirements/0/node"):         []byte("ComputeA"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeAName, "requirements/0/relationship"): []byte("tosca.relationships.HostedOn"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeAName, "requirements/1/name"):         []byte("connect"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeAName, "requirements/1/node"):         []byte("NodeB"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeAName, "requirements/1/relationship"): []byte(relationshipTypeName),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeBName, "type"): []byte("janus.types.B"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeBName, "requirements/0/capability"):   []byte("tosca.capabilities.Container"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeBName, "requirements/0/name"):         []byte("host"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeBName, "requirements/0/node"):         []byte("ComputeB"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeBName, "requirements/0/relationship"): []byte("tosca.relationships.HostedOn"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes/ComputeA/type"):                             []byte("tosca.nodes.Compute"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeA/0/attributes/ip_address"):      []byte("10.10.10.1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeA/1/attributes/ip_address"):      []byte("10.10.10.2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeA/2/attributes/ip_address"):      []byte("10.10.10.3"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeA/0/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeA/1/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeA/2/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.3"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes/ComputeB/type"):                        []byte("tosca.nodes.Compute"),
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
	for i, operation := range operationTestCases {
		t.Run("testExecutionResolveInputsOnRelationshipSource-"+strconv.Itoa(i), func(t *testing.T) {
			testExecutionResolveInputsOnRelationshipSource(t, kv, deploymentID, nodeAName, nodeBName, operation, relationshipTypeName)
		})
		t.Run("testExecutionGenerateOnRelationshipSource-"+strconv.Itoa(i), func(t *testing.T) {
			testExecutionGenerateOnRelationshipSource(t, kv, deploymentID, nodeAName, operation)
		})
	}
}

func testExecutionResolveInputsOnRelationshipSource(t *testing.T, kv *api.KV, deploymentID, nodeAName, nodeBName, operation, relationshipTypeName string) {
	op, err := operations.GetOperation(kv, deploymentID, nodeAName, operation)
	require.Nil(t, err)
	execution := &executionCommon{kv: kv,
		deploymentID:           deploymentID,
		NodeName:               nodeAName,
		operation:              op,
		OperationPath:          path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/pre_configure_source"),
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
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_1":
				require.Equal(t, "/var/www", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_2":
				require.Equal(t, "/var/www", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		case "A2":

			switch envInput.InstanceName {
			case "NodeB_0":
				require.Equal(t, "10.10.10.10", envInput.Value)
				require.True(t, envInput.IsTargetScoped)
			case "NodeB_1":
				require.Equal(t, "10.10.10.11", envInput.Value)
				require.True(t, envInput.IsTargetScoped)
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
	execution.(*executionScript).OperationRemoteBaseDir = "tmp"
	execution.(*executionScript).OperationRemotePath = path.Join(execution.(*executionScript).OperationRemoteBaseDir, ".janus")

	expectedResult := `- name: Executing script {{ script_to_run }}
  hosts: all
  strategy: free
  tasks:
    - file: path="{{ ansible_env.HOME}}/tmp/.janus" state=directory mode=0755

    - copy: src="{{ script_to_run }}" dest="{{ ansible_env.HOME}}/tmp/.janus" mode=0744


    - shell: "{{ ansible_env.HOME}}/tmp/.janus/pre_configure_source.sh"
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

     - file: path="{{ ansible_env.HOME}}/` + execution.(*executionScript).OperationRemoteBaseDir + `" state=absent


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

func testExecutionOnRelationshipTarget(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()
	log.SetDebug(true)

	deploymentID := janus_testutil.BuildDeploymentID(t)
	nodeAName := "NodeA"
	relationshipTypeName := "janus.types.Rel"
	nodeBName := "NodeB"

	var operationTestCases = []string{
		"tosca.interfaces.node.lifecycle.Configure.add_source/1",
		"tosca.interfaces.node.lifecycle.Configure.add_source/connect/" + nodeBName,
	}

	srv1.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/implementation_artifacts_extensions/sh"):         []byte("tosca.artifacts.Implementation.Bash"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types/tosca.artifacts.Implementation.Bash/name"): []byte("tosca.artifacts.Implementation.Bash"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types/janus.types.A/name"):                                                                        []byte("janus.types.A"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "name"):                                                             []byte(relationshipTypeName),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/add_source/name"):                             []byte("add_source"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/add_source/inputs/A1/name"):                   []byte("A1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/add_source/inputs/A1/expression"):             []byte("get_property: [SOURCE, document_root]"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/add_source/inputs/A2/name"):                   []byte("A2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/add_source/inputs/A2/expression"):             []byte("get_attribute: [TARGET, ip_address]"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/add_source/implementation/primary"):           []byte("/tmp/.janus/add_source.sh"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/add_source/implementation/dependencies"):      []byte(""),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/add_source/inputs/A1/is_property_definition"): []byte("false"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/add_source/inputs/A2/is_property_definition"): []byte("false"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types/tosca.nodes.Compute/name"): []byte("tosca.nodes.Compute"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types/janus.types.B/name"): []byte("janus.types.B"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeAName, "properties/document_root"): []byte("/var/www"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeAName, "type"):                     []byte("janus.types.A"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeAName, "requirements/0/capability"):   []byte("tosca.capabilities.Container"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeAName, "requirements/0/name"):         []byte("host"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeAName, "requirements/0/node"):         []byte("ComputeA"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeAName, "requirements/0/relationship"): []byte("tosca.relationships.HostedOn"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeBName, "type"): []byte("janus.types.B"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeBName, "requirements/0/capability"):   []byte("tosca.capabilities.Container"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeBName, "requirements/0/name"):         []byte("host"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeBName, "requirements/0/node"):         []byte("ComputeB"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeBName, "requirements/0/relationship"): []byte("tosca.relationships.HostedOn"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeAName, "requirements/1/name"):         []byte("connect"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeAName, "requirements/1/node"):         []byte("NodeB"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeAName, "requirements/1/relationship"): []byte(relationshipTypeName),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes/ComputeA/type"):                        []byte("tosca.nodes.Compute"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeA/0/attributes/ip_address"): []byte("10.10.10.1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeA/1/attributes/ip_address"): []byte("10.10.10.2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeA/2/attributes/ip_address"): []byte("10.10.10.3"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeA/0/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.1"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeA/1/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.2"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/ComputeA/2/capabilities/endpoint/attributes/ip_address"): []byte("10.10.10.3"),

		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes/ComputeB/type"):                        []byte("tosca.nodes.Compute"),
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
	for i, operation := range operationTestCases {
		t.Run("testExecutionResolveInputOnRelationshipTarget-"+strconv.Itoa(i), func(t *testing.T) {
			testExecutionResolveInputOnRelationshipTarget(t, kv, deploymentID, nodeAName, nodeBName, operation, relationshipTypeName)
		})

		t.Run("testExecutionGenerateOnRelationshipTarget-"+strconv.Itoa(i), func(t *testing.T) {
			testExecutionGenerateOnRelationshipTarget(t, kv, deploymentID, nodeAName, operation)
		})
	}
}
func testExecutionResolveInputOnRelationshipTarget(t *testing.T, kv *api.KV, deploymentID, nodeAName, nodeBName, operation, relationshipTypeName string) {
	op, err := operations.GetOperation(kv, deploymentID, nodeAName, operation)
	require.Nil(t, err)
	execution := &executionCommon{kv: kv,
		deploymentID:             deploymentID,
		NodeName:                 nodeAName,
		operation:                op,
		OperationPath:            path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", relationshipTypeName, "interfaces/Configure/add_source"),
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
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_1":
				require.Equal(t, "/var/www", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			case "NodeA_2":
				require.Equal(t, "/var/www", envInput.Value)
				require.False(t, envInput.IsTargetScoped)
			default:
				require.Fail(t, "Unexpected instance name: ", envInput.Name)
			}
		case "A2":

			switch envInput.InstanceName {
			case "NodeB_0":
				require.Equal(t, "10.10.10.10", envInput.Value)
				require.True(t, envInput.IsTargetScoped)
			case "NodeB_1":
				require.Equal(t, "10.10.10.11", envInput.Value)
				require.True(t, envInput.IsTargetScoped)
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
	execution.(*executionScript).OperationRemoteBaseDir = "tmp"
	execution.(*executionScript).OperationRemotePath = path.Join(execution.(*executionScript).OperationRemoteBaseDir, ".janus")

	expectedResult := `- name: Executing script {{ script_to_run }}
  hosts: all
  strategy: free
  tasks:
    - file: path="{{ ansible_env.HOME}}/tmp/.janus" state=directory mode=0755

    - copy: src="{{ script_to_run }}" dest="{{ ansible_env.HOME}}/tmp/.janus" mode=0744


    - shell: "{{ ansible_env.HOME}}/tmp/.janus/add_source.sh"
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

     - file: path="{{ ansible_env.HOME}}/` + execution.(*executionScript).OperationRemoteBaseDir + `" state=absent


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
