// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package bootstrap

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	expect "github.com/Netflix/go-expect"
	"github.com/hinshun/vt10x"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	survey "gopkg.in/AlecAivazis/survey.v1"
	terminal "gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/tosca"
)

const (
	propertyDescription         = "Property value"
	inputString                 = "test value"
	inputBooleanString          = "true"
	inputList                   = "item1,item2"
	datatypeName                = "myDatatype"
	datatypePropName            = "myDatatypeProperty"
	datatypeInputFmtQuestion    = "Do you want to define property %q"
	datatypePropertyDescription = "DatatypeProperty value"
	datatypeInputAnswer         = "no"
)

// TestGetPropertiesInput checks functions getting interactive inputs
func TestGetPropertiesInput(t *testing.T) {

	required := false

	tests := []struct {
		propertyName string
		definition   tosca.PropertyDefinition
		question     string // Expected output in question before sending input
		answer       string // Answer to question
		convertBool  bool
		want         interface{}
	}{
		{propertyName: "stringProperty",
			definition: tosca.PropertyDefinition{
				Type:        "string",
				Description: propertyDescription,
				Required:    &required},
			question: propertyDescription,
			answer:   inputString,
			want:     inputString,
		},
		{propertyName: "stringSecretProperty",
			definition: tosca.PropertyDefinition{
				Type:        "string",
				Description: propertyDescription,
				Required:    &required},
			question: propertyDescription,
			answer:   inputString,
			want:     inputString,
		},
		{propertyName: "booleanStringProperty",
			definition: tosca.PropertyDefinition{
				Type:        "boolean",
				Description: propertyDescription,
				Required:    &required,
				Default: &tosca.ValueAssignment{
					Type:  tosca.ValueAssignmentLiteral,
					Value: "false",
				}},
			question:    propertyDescription,
			answer:      inputBooleanString,
			convertBool: true,
			want:        inputBooleanString,
		},
		{propertyName: "booleanProperty",
			definition: tosca.PropertyDefinition{
				Type:        "boolean",
				Description: propertyDescription,
				Required:    &required,
				Default: &tosca.ValueAssignment{
					Type:  tosca.ValueAssignmentLiteral,
					Value: "false",
				}},
			question:    propertyDescription,
			answer:      inputBooleanString,
			convertBool: false,
			want:        (inputBooleanString == "true"),
		},
		{propertyName: "listProperty",
			definition: tosca.PropertyDefinition{
				Type:        "list",
				Description: propertyDescription,
				Required:    &required},
			question: propertyDescription,
			answer:   inputList,
			want:     strings.Split(inputList, ","),
		},
		{propertyName: "dataTypeProperty",
			definition: tosca.PropertyDefinition{
				Type:        datatypeName,
				Description: propertyDescription,
				Required:    &required},
			question: fmt.Sprintf(datatypeInputFmtQuestion, propertyDescription),
			answer:   datatypeInputAnswer,
			want:     nil,
		},
	}

	topology := tosca.Topology{
		DataTypes: map[string]tosca.DataType{
			datatypeName: tosca.DataType{
				Properties: map[string]tosca.PropertyDefinition{
					datatypePropName: tosca.PropertyDefinition{
						Type:        "string",
						Description: datatypePropertyDescription,
						Required:    &required},
				},
			},
		},
	}

	for _, tt := range tests {

		resultMap := make(config.DynamicMap)
		properties := map[string]tosca.PropertyDefinition{
			tt.propertyName: tt.definition,
		}

		runTest(t, tt.question, tt.answer, func(stdio terminal.Stdio) error {
			return getPropertiesInput(topology, properties, true, tt.convertBool, "", &resultMap,
				survey.WithStdio(stdio.In, stdio.Out, stdio.Err))
		})

		assert.Equal(t, tt.want, resultMap[tt.propertyName], "Unexpected property value for %s", tt.propertyName)
	}

}

func expectOutputsSendInputs(c *expect.Console, output, input string) {

	res, err := c.ExpectString(output)
	if err != nil {
		fmt.Printf("Failed to get expected output, got %s\n", res)
	}
	c.SendLine(input)
	c.ExpectEOF()
}

func runTest(t *testing.T, question, answer string, test func(terminal.Stdio) error) {

	buf := new(bytes.Buffer)
	c, _, err := vt10x.NewVT10XConsole(expect.WithStdout(buf), expect.WithDefaultTimeout(10*time.Second))
	require.NoError(t, err, "Failed to create a console")
	defer c.Close()

	donec := make(chan struct{})
	go func() {
		defer close(donec)
		expectOutputsSendInputs(c, question, answer)
	}()

	err = test(terminal.Stdio{c.Tty(), c.Tty(), c.Tty()})
	require.NoError(t, err)

	// Close the slave end of the pty, and read the remaining bytes from the master end.
	c.Tty().Close()
	<-donec

	t.Logf("Raw output: %q", buf.String())

}
