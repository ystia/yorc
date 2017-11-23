package tosca

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestInterfaceSimpleGrammar(t *testing.T) {
	t.Parallel()
	var inputYaml = `start: scripts/start_server.sh`
	ifDefMap := InterfaceDefinitionMap{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDefMap)
	require.Nil(t, err, "Expecting no error when unmarshaling Interface with simple grammar")
	require.Len(t, ifDefMap, 1, "Expecting one interface")
	require.Contains(t, ifDefMap, "start")
	ifDef := ifDefMap["start"]
	require.Len(t, ifDef.Inputs, 0, "Expecting no inputs")
	require.Equal(t, "scripts/start_server.sh", ifDef.Implementation.Primary)
}

func TestInterfaceComplexGrammar(t *testing.T) {
	t.Parallel()
	var inputYaml = `
start:
  inputs:
    X: "Y"
  implementation: scripts/start_server.sh`
	ifDefMap := InterfaceDefinitionMap{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDefMap)
	require.Nil(t, err, "Expecting no error when unmarshaling Interface with complex grammar")
	require.Len(t, ifDefMap, 1, "Expecting one interface")
	require.Contains(t, ifDefMap, "start")
	ifDef := ifDefMap["start"]
	require.Len(t, ifDef.Inputs, 1, "Expecting 1 input")
	require.Contains(t, ifDef.Inputs, "X")
	require.Equal(t, "Y", fmt.Sprint(ifDef.Inputs["X"].ValueAssign.String()))
	require.Equal(t, "scripts/start_server.sh", ifDef.Implementation.Primary)
}

func TestInterfaceExpressionInputs(t *testing.T) {
	t.Parallel()
	var inputYaml = `
start:
  inputs:
    X: { get_property: [SELF, prop]}
  implementation: scripts/start_server.sh`
	ifDefMap := InterfaceDefinitionMap{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDefMap)
	require.Nil(t, err, "Expecting no error when unmarshaling Interface with complex grammar")
	require.Len(t, ifDefMap, 1, "Expecting one interface")
	require.Contains(t, ifDefMap, "start")
	ifDef := ifDefMap["start"]
	require.Len(t, ifDef.Inputs, 1, "Expecting 1 input")
	require.Contains(t, ifDef.Inputs, "X")
	require.Nil(t, ifDef.Inputs["X"].PropDef)
	require.NotNil(t, ifDef.Inputs["X"].ValueAssign)
	require.Equal(t, ValueAssignmentFunction, ifDef.Inputs["X"].ValueAssign.Type)
	require.EqualValues(t, "get_property", ifDef.Inputs["X"].ValueAssign.GetFunction().Operator)

	var testData = []struct {
		index int
		value string
	}{
		{0, "SELF"},
		{1, "prop"},
	}
	for _, tt := range testData {
		require.Equal(t, tt.value, ifDef.Inputs["X"].ValueAssign.GetFunction().Operands[tt.index].String())
		require.True(t, ifDef.Inputs["X"].ValueAssign.GetFunction().Operands[tt.index].IsLiteral())
	}

	require.Equal(t, "scripts/start_server.sh", ifDef.Implementation.Primary)
}

func TestInterfaceExpressionInputsAsPropDef(t *testing.T) {
	t.Parallel()
	var inputYaml = `
update_replicas:
  inputs:
    nb_replicas:
      type: integer
      description: Number of replicas for indexes
      required: true
    index:
      type: string
      description: The name of the index to be updated (specify no value for all indexes)
      required: false
  implementation: scripts/elasticsearch_updateReplicas.sh`
	ifDefMap := InterfaceDefinitionMap{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDefMap)
	require.Nil(t, err, "Expecting no error when unmarshaling Interface with complex grammar")
	require.Len(t, ifDefMap, 1, "Expecting one interface")
	require.Contains(t, ifDefMap, "update_replicas")
	ifDef := ifDefMap["update_replicas"]
	require.Len(t, ifDef.Inputs, 2, "Expecting 2 inputs")
	require.Contains(t, ifDef.Inputs, "nb_replicas")
	require.NotNil(t, ifDef.Inputs["nb_replicas"].PropDef)
	require.Nil(t, ifDef.Inputs["nb_replicas"].ValueAssign)
	require.Equal(t, "integer", ifDef.Inputs["nb_replicas"].PropDef.Type)
	require.Equal(t, "Number of replicas for indexes", ifDef.Inputs["nb_replicas"].PropDef.Description)
	require.Equal(t, true, *ifDef.Inputs["nb_replicas"].PropDef.Required)
	require.Contains(t, ifDef.Inputs, "index")
	require.NotNil(t, ifDef.Inputs["index"].PropDef)
	require.Nil(t, ifDef.Inputs["index"].ValueAssign)
	require.Equal(t, "string", ifDef.Inputs["index"].PropDef.Type)
	require.Equal(t, "The name of the index to be updated (specify no value for all indexes)", ifDef.Inputs["index"].PropDef.Description)
	require.Equal(t, false, *ifDef.Inputs["index"].PropDef.Required)

	require.Equal(t, "scripts/elasticsearch_updateReplicas.sh", ifDef.Implementation.Primary)
}

func TestInterfaceExpressionInputsComplexExpression(t *testing.T) {
	t.Parallel()
	var inputYaml = `
start:
  inputs:
    X: { concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port] ] }
  implementation: scripts/start_server.sh`
	ifDefMap := InterfaceDefinitionMap{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDefMap)
	require.Nil(t, err, "Expecting no error when unmarshaling Interface with complex grammar")
	require.Len(t, ifDefMap, 1, "Expecting one interface")
	require.Contains(t, ifDefMap, "start")
	ifDef := ifDefMap["start"]
	require.Len(t, ifDef.Inputs, 1, "Expecting 1 input")
	require.Contains(t, ifDef.Inputs, "X")

	require.Nil(t, ifDef.Inputs["X"].PropDef)
	require.NotNil(t, ifDef.Inputs["X"].ValueAssign)
	require.Equal(t, ValueAssignmentFunction, ifDef.Inputs["X"].ValueAssign.Type)
	require.EqualValues(t, "concat", ifDef.Inputs["X"].ValueAssign.GetFunction().Operator)

	concatChildren := ifDef.Inputs["X"].ValueAssign.GetFunction().Operands

	require.Equal(t, 4, len(concatChildren))

	var testData = []struct {
		index     int
		value     string
		isLiteral bool
	}{
		{0, `"http://"`, true},
		{1, "get_attribute: [HOST, public_ip_address]", false},
		{2, `":"`, true},
		{3, "get_property: [SELF, port]", false},
	}
	for _, tt := range testData {
		require.Equal(t, tt.value, concatChildren[tt.index].String())
		require.Equal(t, tt.isLiteral, concatChildren[tt.index].IsLiteral())
	}

	getAttr := concatChildren[1]
	require.IsType(t, &Function{}, getAttr)
	getAttrChildren := getAttr.(*Function).Operands

	require.Equal(t, 2, len(getAttrChildren))

	testData = []struct {
		index     int
		value     string
		isLiteral bool
	}{
		{0, "HOST", true},
		{1, "public_ip_address", true},
	}
	for _, tt := range testData {
		require.Equal(t, tt.value, getAttrChildren[tt.index].String())
		require.Equal(t, tt.isLiteral, getAttrChildren[tt.index].IsLiteral())
	}

	getProp := concatChildren[3]
	require.IsType(t, &Function{}, getProp)
	getPropChildren := getProp.(*Function).Operands

	require.Equal(t, 2, len(getPropChildren))

	testData = []struct {
		index     int
		value     string
		isLiteral bool
	}{
		{0, "SELF", true},
		{1, "port", true},
	}
	for _, tt := range testData {
		require.Equal(t, tt.value, getPropChildren[tt.index].String())
		require.Equal(t, tt.isLiteral, getPropChildren[tt.index].IsLiteral())
	}

	require.Equal(t, "scripts/start_server.sh", ifDef.Implementation.Primary)
}

func TestInterfaceMixed(t *testing.T) {
	t.Parallel()
	var inputYaml = `
start:
  inputs:
    X: "Y"
  implementation: scripts/start_server.sh
stop: scripts/stop_server.sh`
	ifDefMap := InterfaceDefinitionMap{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDefMap)
	require.Nil(t, err, "Expecting no error when unmarshaling Interface with mixed grammar")
	require.Len(t, ifDefMap, 2, "Expecting one interface")
	require.Contains(t, ifDefMap, "start")
	ifDef := ifDefMap["start"]
	require.Len(t, ifDef.Inputs, 1, "Expecting 1 input")
	require.Contains(t, ifDef.Inputs, "X")
	require.Equal(t, "Y", fmt.Sprint(ifDef.Inputs["X"].ValueAssign))
	require.Equal(t, "scripts/start_server.sh", ifDef.Implementation.Primary)
	require.Contains(t, ifDefMap, "stop")
	ifDefConcreteStop := ifDefMap["stop"]
	require.Len(t, ifDefConcreteStop.Inputs, 0, "Expecting no inputs")
	require.Equal(t, "scripts/stop_server.sh", ifDefConcreteStop.Implementation.Primary)
}

func TestInterfaceFailing(t *testing.T) {
	t.Parallel()
	var inputYaml = `
start:
  inputs: ["Y" , "Z" ]
  implementation: scripts/start_server.sh
stop: scripts/stop_server.sh`
	ifDef := InterfaceDefinitionMap{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDef)
	require.NotNil(t, err, "Expecting an error when unmarshaling Interface with an array as inputs")
}
