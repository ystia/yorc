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
	ifDef := InterfaceDefinition{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDef)
	require.Nil(t, err, "Expecting no error when unmarshaling Interface with simple grammar")
	require.Len(t, ifDef.Operations, 1, "Expecting one interface")
	require.Contains(t, ifDef.Operations, "start")
	opDef := ifDef.Operations["start"]
	require.Len(t, opDef.Inputs, 0, "Expecting no inputs")
	require.Equal(t, "scripts/start_server.sh", opDef.Implementation.Primary)
}

func TestInterfaceComplexGrammar(t *testing.T) {
	t.Parallel()
	var inputYaml = `
start:
  inputs:
    X: "Y"
  implementation: scripts/start_server.sh`
	ifDef := InterfaceDefinition{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDef)
	require.Nil(t, err, "Expecting no error when unmarshaling Interface with complex grammar")
	require.Len(t, ifDef.Operations, 1, "Expecting one interface")
	require.Contains(t, ifDef.Operations, "start")
	opDef := ifDef.Operations["start"]
	require.Len(t, opDef.Inputs, 1, "Expecting 1 input")
	require.Contains(t, opDef.Inputs, "X")
	require.Equal(t, "Y", fmt.Sprint(opDef.Inputs["X"].ValueAssign.String()))
	require.Equal(t, "scripts/start_server.sh", opDef.Implementation.Primary)
}

func TestInterfaceExpressionInputs(t *testing.T) {
	t.Parallel()
	var inputYaml = `
start:
  inputs:
    X: { get_property: [SELF, prop]}
  implementation: scripts/start_server.sh`
	ifDef := InterfaceDefinition{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDef)
	require.Nil(t, err, "Expecting no error when unmarshaling Interface with complex grammar")
	require.Len(t, ifDef.Operations, 1, "Expecting one interface")
	require.Contains(t, ifDef.Operations, "start")
	opDef := ifDef.Operations["start"]
	require.Len(t, opDef.Inputs, 1, "Expecting 1 input")
	require.Contains(t, opDef.Inputs, "X")
	require.Nil(t, opDef.Inputs["X"].PropDef)
	require.NotNil(t, opDef.Inputs["X"].ValueAssign)
	require.Equal(t, ValueAssignmentFunction, opDef.Inputs["X"].ValueAssign.Type)
	require.EqualValues(t, "get_property", opDef.Inputs["X"].ValueAssign.GetFunction().Operator)

	var testData = []struct {
		index int
		value string
	}{
		{0, "SELF"},
		{1, "prop"},
	}
	for _, tt := range testData {
		require.Equal(t, tt.value, opDef.Inputs["X"].ValueAssign.GetFunction().Operands[tt.index].String())
		require.True(t, opDef.Inputs["X"].ValueAssign.GetFunction().Operands[tt.index].IsLiteral())
	}

	require.Equal(t, "scripts/start_server.sh", opDef.Implementation.Primary)
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
	ifDef := InterfaceDefinition{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDef)
	require.Nil(t, err, "Expecting no error when unmarshaling Interface with complex grammar")
	require.Len(t, ifDef.Operations, 1, "Expecting one interface")
	require.Contains(t, ifDef.Operations, "update_replicas")
	opDef := ifDef.Operations["update_replicas"]
	require.Len(t, opDef.Inputs, 2, "Expecting 2 inputs")
	require.Contains(t, opDef.Inputs, "nb_replicas")
	require.NotNil(t, opDef.Inputs["nb_replicas"].PropDef)
	require.Nil(t, opDef.Inputs["nb_replicas"].ValueAssign)
	require.Equal(t, "integer", opDef.Inputs["nb_replicas"].PropDef.Type)
	require.Equal(t, "Number of replicas for indexes", opDef.Inputs["nb_replicas"].PropDef.Description)
	require.Equal(t, true, *opDef.Inputs["nb_replicas"].PropDef.Required)
	require.Contains(t, opDef.Inputs, "index")
	require.NotNil(t, opDef.Inputs["index"].PropDef)
	require.Nil(t, opDef.Inputs["index"].ValueAssign)
	require.Equal(t, "string", opDef.Inputs["index"].PropDef.Type)
	require.Equal(t, "The name of the index to be updated (specify no value for all indexes)", opDef.Inputs["index"].PropDef.Description)
	require.Equal(t, false, *opDef.Inputs["index"].PropDef.Required)

	require.Equal(t, "scripts/elasticsearch_updateReplicas.sh", opDef.Implementation.Primary)
}

func TestInterfaceExpressionInputsComplexExpression(t *testing.T) {
	t.Parallel()
	var inputYaml = `
start:
  inputs:
    X: { concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port] ] }
  implementation: scripts/start_server.sh`
	ifDef := InterfaceDefinition{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDef)
	require.Nil(t, err, "Expecting no error when unmarshaling Interface with complex grammar")
	require.Len(t, ifDef.Operations, 1, "Expecting one interface")
	require.Contains(t, ifDef.Operations, "start")
	opDef := ifDef.Operations["start"]
	require.Len(t, opDef.Inputs, 1, "Expecting 1 input")
	require.Contains(t, opDef.Inputs, "X")

	require.Nil(t, opDef.Inputs["X"].PropDef)
	require.NotNil(t, opDef.Inputs["X"].ValueAssign)
	require.Equal(t, ValueAssignmentFunction, opDef.Inputs["X"].ValueAssign.Type)
	require.EqualValues(t, "concat", opDef.Inputs["X"].ValueAssign.GetFunction().Operator)

	concatChildren := opDef.Inputs["X"].ValueAssign.GetFunction().Operands

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

	require.Equal(t, "scripts/start_server.sh", opDef.Implementation.Primary)
}

func TestInterfaceMixed(t *testing.T) {
	t.Parallel()
	var inputYaml = `
start:
  inputs:
    X: "Y"
  implementation: scripts/start_server.sh
stop: scripts/stop_server.sh`
	ifDef := InterfaceDefinition{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDef)
	require.Nil(t, err, "Expecting no error when unmarshaling Interface with mixed grammar")
	require.Len(t, ifDef.Operations, 2, "Expecting one interface")
	require.Contains(t, ifDef.Operations, "start")
	opDef := ifDef.Operations["start"]
	require.Len(t, opDef.Inputs, 1, "Expecting 1 input")
	require.Contains(t, opDef.Inputs, "X")
	require.Equal(t, "Y", fmt.Sprint(opDef.Inputs["X"].ValueAssign))
	require.Equal(t, "scripts/start_server.sh", opDef.Implementation.Primary)
	require.Contains(t, ifDef.Operations, "stop")
	ifDefConcreteStop := ifDef.Operations["stop"]
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
	opDef := InterfaceDefinition{}

	err := yaml.Unmarshal([]byte(inputYaml), &opDef)
	require.NotNil(t, err, "Expecting an error when unmarshaling Interface with an array as inputs")
}

func TestComplexInterfaceWithGobalInputs(t *testing.T) {
	t.Parallel()
	inputYaml := `
type: yorc.test.interfaces.MyCustomInterface
inputs:
  Global1: {get_property: [SELF, p1]}
  Global2: "This is a string"
  Global3:
    type: integer
    description: "An int"
    required: true
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

	ifDef := InterfaceDefinition{}

	err := yaml.Unmarshal([]byte(inputYaml), &ifDef)
	require.Nil(t, err, "Expecting no error when unmarshaling Interface with complex grammar")
	require.Len(t, ifDef.Operations, 1, "Expecting one interface")
	require.Contains(t, ifDef.Operations, "update_replicas")
	opDef := ifDef.Operations["update_replicas"]
	require.Len(t, opDef.Inputs, 2, "Expecting 2 inputs")
	require.Contains(t, opDef.Inputs, "nb_replicas")
	require.NotNil(t, opDef.Inputs["nb_replicas"].PropDef)
	require.Nil(t, opDef.Inputs["nb_replicas"].ValueAssign)
	require.Equal(t, "integer", opDef.Inputs["nb_replicas"].PropDef.Type)
	require.Equal(t, "Number of replicas for indexes", opDef.Inputs["nb_replicas"].PropDef.Description)
	require.Equal(t, true, *opDef.Inputs["nb_replicas"].PropDef.Required)
	require.Contains(t, opDef.Inputs, "index")
	require.NotNil(t, opDef.Inputs["index"].PropDef)
	require.Nil(t, opDef.Inputs["index"].ValueAssign)
	require.Equal(t, "string", opDef.Inputs["index"].PropDef.Type)
	require.Equal(t, "The name of the index to be updated (specify no value for all indexes)", opDef.Inputs["index"].PropDef.Description)
	require.Equal(t, false, *opDef.Inputs["index"].PropDef.Required)

	require.Equal(t, "scripts/elasticsearch_updateReplicas.sh", opDef.Implementation.Primary)

	require.Equal(t, "yorc.test.interfaces.MyCustomInterface", ifDef.Type)
	require.Len(t, ifDef.Inputs, 3)
	require.Contains(t, ifDef.Inputs, "Global1")
	require.NotNil(t, ifDef.Inputs["Global1"].ValueAssign)
	require.Nil(t, ifDef.Inputs["Global1"].PropDef)
	require.Equal(t, ValueAssignmentFunction, ifDef.Inputs["Global1"].ValueAssign.Type)
	require.Equal(t, "get_property: [SELF, p1]", ifDef.Inputs["Global1"].ValueAssign.String())
	require.Contains(t, ifDef.Inputs, "Global2")
	require.NotNil(t, ifDef.Inputs["Global2"].ValueAssign)
	require.Nil(t, ifDef.Inputs["Global2"].PropDef)
	require.Equal(t, ValueAssignmentLiteral, ifDef.Inputs["Global2"].ValueAssign.Type)
	require.Equal(t, "This is a string", ifDef.Inputs["Global2"].ValueAssign.String())
	require.Contains(t, ifDef.Inputs, "Global3")
	require.NotNil(t, ifDef.Inputs["Global3"].PropDef)
	require.Nil(t, ifDef.Inputs["Global3"].ValueAssign)
	require.Equal(t, true, *ifDef.Inputs["Global3"].PropDef.Required)
	require.Equal(t, "integer", ifDef.Inputs["Global3"].PropDef.Type)
	require.Equal(t, "An int", ifDef.Inputs["Global3"].PropDef.Description)

}
