package tosca

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestGroupedInterfacesParallel(t *testing.T) {
	t.Run("groupInterfaces", func(t *testing.T) {
		t.Run("TestInterfaceSimpleGrammar", interfaceSimpleGrammar)
		t.Run("TestInterfaceComplexGrammar", interfaceComplexGrammar)
		t.Run("TestInterfaceExpressionInputs", interfaceExpressionInputs)
		t.Run("TestInterfaceExpressionInputsComplexExpression", interfaceExpressionInputsComplexExpression)
		t.Run("TestInterfaceMixed", interfaceMixed)
		t.Run("TestInterfaceFailing", interfaceFailing)
		t.Run("TestInterfaceExpressionInputsAsPropDef", interfaceExpressionInputsAsPropDef)
	})
}

func interfaceSimpleGrammar(t *testing.T) {
	t.Parallel()
	var inputYaml = `start: scripts/start_server.sh`
	ifDefMap := InterfaceDefinitionMap{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDefMap)
	assert.Nil(t, err, "Expecting no error when unmarshaling Interface with simple grammar")
	assert.Len(t, ifDefMap, 1, "Expecting one interface")
	assert.Contains(t, ifDefMap, "start")
	ifDef := ifDefMap["start"]
	assert.Len(t, ifDef.Inputs, 0, "Expecting no inputs")
	assert.Equal(t, "scripts/start_server.sh", ifDef.Implementation.Primary)
}

func interfaceComplexGrammar(t *testing.T) {
	t.Parallel()
	var inputYaml = `
start:
  inputs:
    X: Y
  implementation: scripts/start_server.sh`
	ifDefMap := InterfaceDefinitionMap{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDefMap)
	assert.Nil(t, err, "Expecting no error when unmarshaling Interface with complex grammar")
	assert.Len(t, ifDefMap, 1, "Expecting one interface")
	assert.Contains(t, ifDefMap, "start")
	ifDef := ifDefMap["start"]
	assert.Len(t, ifDef.Inputs, 1, "Expecting 1 input")
	assert.Contains(t, ifDef.Inputs, "X")
	assert.Equal(t, "Y", fmt.Sprint(ifDef.Inputs["X"].ValueAssign))
	assert.Equal(t, "scripts/start_server.sh", ifDef.Implementation.Primary)
}

func interfaceExpressionInputs(t *testing.T) {
	t.Parallel()
	var inputYaml = `
start:
  inputs:
    X: { getProperty: [SELF, prop]}
  implementation: scripts/start_server.sh`
	ifDefMap := InterfaceDefinitionMap{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDefMap)
	assert.Nil(t, err, "Expecting no error when unmarshaling Interface with complex grammar")
	assert.Len(t, ifDefMap, 1, "Expecting one interface")
	assert.Contains(t, ifDefMap, "start")
	ifDef := ifDefMap["start"]
	assert.Len(t, ifDef.Inputs, 1, "Expecting 1 input")
	assert.Contains(t, ifDef.Inputs, "X")
	require.Nil(t, ifDef.Inputs["X"].PropDef)
	require.NotNil(t, ifDef.Inputs["X"].ValueAssign)
	assert.Equal(t, "getProperty", ifDef.Inputs["X"].ValueAssign.Expression.Value)

	var testData = []struct {
		index int
		value string
	}{
		{0, "SELF"},
		{1, "prop"},
	}
	for _, tt := range testData {
		assert.Equal(t, tt.value, ifDef.Inputs["X"].ValueAssign.Expression.Children()[tt.index].Value)
		assert.True(t, ifDef.Inputs["X"].ValueAssign.Expression.Children()[tt.index].IsLiteral())
	}

	assert.Equal(t, "scripts/start_server.sh", ifDef.Implementation.Primary)
}

func interfaceExpressionInputsAsPropDef(t *testing.T) {
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
	assert.Nil(t, err, "Expecting no error when unmarshaling Interface with complex grammar")
	assert.Len(t, ifDefMap, 1, "Expecting one interface")
	assert.Contains(t, ifDefMap, "update_replicas")
	ifDef := ifDefMap["update_replicas"]
	assert.Len(t, ifDef.Inputs, 2, "Expecting 2 inputs")
	assert.Contains(t, ifDef.Inputs, "nb_replicas")
	require.NotNil(t, ifDef.Inputs["nb_replicas"].PropDef)
	require.Nil(t, ifDef.Inputs["nb_replicas"].ValueAssign)
	assert.Equal(t, "integer", ifDef.Inputs["nb_replicas"].PropDef.Type)
	assert.Equal(t, "Number of replicas for indexes", ifDef.Inputs["nb_replicas"].PropDef.Description)
	assert.Equal(t, true, ifDef.Inputs["nb_replicas"].PropDef.Required)
	assert.Contains(t, ifDef.Inputs, "index")
	require.NotNil(t, ifDef.Inputs["index"].PropDef)
	require.Nil(t, ifDef.Inputs["index"].ValueAssign)
	assert.Equal(t, "string", ifDef.Inputs["index"].PropDef.Type)
	assert.Equal(t, "The name of the index to be updated (specify no value for all indexes)", ifDef.Inputs["index"].PropDef.Description)
	assert.Equal(t, false, ifDef.Inputs["index"].PropDef.Required)

	assert.Equal(t, "scripts/elasticsearch_updateReplicas.sh", ifDef.Implementation.Primary)
}

func interfaceExpressionInputsComplexExpression(t *testing.T) {
	t.Parallel()
	var inputYaml = `
start:
  inputs:
    X: { concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port] ] }
  implementation: scripts/start_server.sh`
	ifDefMap := InterfaceDefinitionMap{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDefMap)
	assert.Nil(t, err, "Expecting no error when unmarshaling Interface with complex grammar")
	assert.Len(t, ifDefMap, 1, "Expecting one interface")
	assert.Contains(t, ifDefMap, "start")
	ifDef := ifDefMap["start"]
	assert.Len(t, ifDef.Inputs, 1, "Expecting 1 input")
	assert.Contains(t, ifDef.Inputs, "X")

	require.Nil(t, ifDef.Inputs["X"].PropDef)
	require.NotNil(t, ifDef.Inputs["X"].ValueAssign)
	assert.Equal(t, "concat", ifDef.Inputs["X"].ValueAssign.Expression.Value)

	concatChildren := ifDef.Inputs["X"].ValueAssign.Expression.Children()

	assert.Equal(t, 4, len(concatChildren))

	var testData = []struct {
		index      int
		value      string
		isLitteral bool
	}{
		{0, "http://", true},
		{1, "get_attribute", false},
		{2, ":", true},
		{3, "get_property", false},
	}
	for _, tt := range testData {
		assert.Equal(t, tt.value, concatChildren[tt.index].Value)
		assert.Equal(t, tt.isLitteral, concatChildren[tt.index].IsLiteral())
		assert.Equal(t, ifDef.Inputs["X"].ValueAssign.Expression, concatChildren[tt.index].Parent())
	}

	getAttr := concatChildren[1]
	getAttrChildren := getAttr.Children()

	assert.Equal(t, 2, len(getAttrChildren))

	testData = []struct {
		index      int
		value      string
		isLitteral bool
	}{
		{0, "HOST", true},
		{1, "public_ip_address", true},
	}
	for _, tt := range testData {
		assert.Equal(t, tt.value, getAttrChildren[tt.index].Value)
		assert.Equal(t, tt.isLitteral, getAttrChildren[tt.index].IsLiteral())
		assert.Equal(t, getAttr, getAttrChildren[tt.index].Parent())
	}

	getProp := concatChildren[3]
	getPropChildren := getProp.Children()

	assert.Equal(t, 2, len(getPropChildren))

	testData = []struct {
		index      int
		value      string
		isLitteral bool
	}{
		{0, "SELF", true},
		{1, "port", true},
	}
	for _, tt := range testData {
		assert.Equal(t, tt.value, getPropChildren[tt.index].Value)
		assert.Equal(t, tt.isLitteral, getPropChildren[tt.index].IsLiteral())
		assert.Equal(t, getProp, getPropChildren[tt.index].Parent())
	}

	assert.Equal(t, "scripts/start_server.sh", ifDef.Implementation.Primary)
}

func interfaceMixed(t *testing.T) {
	t.Parallel()
	var inputYaml = `
start:
  inputs:
    X: Y
  implementation: scripts/start_server.sh
stop: scripts/stop_server.sh`
	ifDefMap := InterfaceDefinitionMap{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDefMap)
	assert.Nil(t, err, "Expecting no error when unmarshaling Interface with mixed grammar")
	assert.Len(t, ifDefMap, 2, "Expecting one interface")
	assert.Contains(t, ifDefMap, "start")
	ifDef := ifDefMap["start"]
	assert.Len(t, ifDef.Inputs, 1, "Expecting 1 input")
	assert.Contains(t, ifDef.Inputs, "X")
	assert.Equal(t, "Y", fmt.Sprint(ifDef.Inputs["X"].ValueAssign))
	assert.Equal(t, "scripts/start_server.sh", ifDef.Implementation.Primary)
	assert.Contains(t, ifDefMap, "stop")
	ifDefConcreteStop := ifDefMap["stop"]
	assert.Len(t, ifDefConcreteStop.Inputs, 0, "Expecting no inputs")
	assert.Equal(t, "scripts/stop_server.sh", ifDefConcreteStop.Implementation.Primary)
}

func interfaceFailing(t *testing.T) {
	t.Parallel()
	var inputYaml = `
start:
  inputs: ["Y" , "Z" ]
  implementation: scripts/start_server.sh
stop: scripts/stop_server.sh`
	ifDef := InterfaceDefinitionMap{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDef)
	assert.NotNil(t, err, "Expecting an error when unmarshaling Interface with an array as inputs")
}
