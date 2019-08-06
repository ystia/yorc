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
	propertyDescription = "Property value"
	expectedStringValue = "test value"
	expectedBoolValue   = "true"
)

// TestGetPropertiesInput checks functions getting interactive inputs
func TestGetPropertiesInput(t *testing.T) {

	required := false
	definition := tosca.PropertyDefinition{
		Type:        "string",
		Description: "Property value",
		Required:    &required,
	}

	tests := []struct {
		propertyName string
		definition   tosca.PropertyDefinition
		input        string
	}{
		{propertyName: "stringProperty",
			definition: tosca.PropertyDefinition{
				Type:        "string",
				Description: propertyDescription,
				Required:    &required},
			input: expectedStringValue,
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
			input: expectedBoolValue,
		},
	}
	var topology tosca.Topology
	for _, tt := range tests {

		resultMap := make(config.DynamicMap)
		properties := map[string]tosca.PropertyDefinition{
			tt.propertyName: definition,
		}

		runTest(t, tt.input, func(stdio terminal.Stdio) error {
			return getPropertiesInput(topology, properties, true, true, "", &resultMap, survey.WithStdio(stdio.In, stdio.Out, stdio.Err))
		})

		assert.Equal(t, tt.input, resultMap[tt.propertyName], "Unexpected property value for %s", tt.propertyName)
	}

}

func expectDescriptionSendInput(c *expect.Console, input string) {
	res, err := c.ExpectString(propertyDescription)
	if err != nil {
		fmt.Printf("Failed to get expected output, go %s\n", res)
	}
	c.SendLine(input)
	c.ExpectEOF()
}

func runTest(t *testing.T, input string, test func(terminal.Stdio) error) {

	// Multiplex output to a buffer as well for the raw bytes.
	buf := new(bytes.Buffer)
	c, _, err := vt10x.NewVT10XConsole(expect.WithStdout(buf), expect.WithDefaultTimeout(10*time.Second))
	require.NoError(t, err, "Failed to create a console")
	defer c.Close()

	donec := make(chan struct{})
	go func() {
		defer close(donec)
		expectDescriptionSendInput(c, input)
	}()

	err = test(terminal.Stdio{c.Tty(), c.Tty(), c.Tty()})
	require.NoError(t, err)

	// Close the slave end of the pty, and read the remaining bytes from the master end.
	c.Tty().Close()
	<-donec

	t.Logf("Raw output: %q", buf.String())

}
