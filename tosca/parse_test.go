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
	"io/ioutil"
	_ "log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestGroupedParsingParallel(t *testing.T) {
	t.Run("groupParsing", func(t *testing.T) {
		t.Run("TestParsing", parsing)
		t.Run("TestSubsitutionMappingsParsing", parsingSubstitutionMappings)
	})
}

func parsing(t *testing.T) {
	t.Parallel()
	definition, err := os.Open(filepath.Join("testdata", "dep.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defBytes, err := ioutil.ReadAll(definition)
	if err != nil {
		t.Fatal(err)
	}
	topology := Topology{}
	err = yaml.Unmarshal(defBytes, &topology)
	if err != nil {
		t.Fatal(err)
	}
	//log.Printf("%+v\n\n%#v\n", topology, topology)

	// Check Topology types
	assert.Equal(t, "tosca_simple_yaml_1_0_0_wd03", topology.TOSCAVersion)
	assert.Equal(t, "Alien4Cloud generated service template", topology.Description)
	assert.Equal(t, "Test", topology.Metadata[TemplateName])
	assert.Equal(t, "0.1.0-SNAPSHOT", topology.Metadata[TemplateVersion])
	assert.Equal(t, "admin", topology.Metadata[TemplateAuthor])

	assert.Len(t, topology.Imports, 1)
	importDef := topology.Imports[0]
	assert.Equal(t, "1.0.0-ALIEN11", importDef.File)

	//Check topology template
	topologyTemplate := topology.TopologyTemplate
	assert.NotNil(t, topologyTemplate)

	// Check node type
	assert.Len(t, topologyTemplate.NodeTemplates, 1)
	assert.Contains(t, topologyTemplate.NodeTemplates, "Compute")

	compute := topologyTemplate.NodeTemplates["Compute"]
	assert.NotNil(t, compute)

	assert.Equal(t, "yorc.nodes.openstack.Compute", compute.Type)

	// Check node's properties
	assert.Len(t, compute.Properties, 4)
	assert.Contains(t, compute.Properties, "user")
	assert.Equal(t, "cloud-user", fmt.Sprint(compute.Properties["user"]))
	assert.Contains(t, compute.Properties, "image")
	assert.Equal(t, "89ec515c-3251-4c2f-8402-bda280c31650", fmt.Sprint(compute.Properties["image"]))
	assert.Contains(t, compute.Properties, "flavor")
	assert.Equal(t, "2", fmt.Sprint(compute.Properties["flavor"]))
	assert.Contains(t, compute.Properties, "availability_zone")
	assert.Equal(t, "nova", fmt.Sprint(compute.Properties["availability_zone"]))

	// Check node's capabilities
	assert.Len(t, compute.Capabilities, 2)
	assert.Contains(t, compute.Capabilities, "endpoint")
	assert.Contains(t, compute.Capabilities, "scalable")
	endpointCap := compute.Capabilities["endpoint"]
	assert.Len(t, endpointCap.Properties, 4)
	assert.Contains(t, endpointCap.Properties, "protocol")
	assert.Contains(t, endpointCap.Properties, "initiator")
	assert.Contains(t, endpointCap.Properties, "secure")
	assert.Contains(t, endpointCap.Properties, "network_name")
	assert.Equal(t, "tcp", fmt.Sprint(endpointCap.Properties["protocol"]))
	assert.Equal(t, "source", fmt.Sprint(endpointCap.Properties["initiator"]))
	assert.Equal(t, "true", fmt.Sprint(endpointCap.Properties["secure"]))
	assert.Equal(t, "private_starlings", fmt.Sprint(endpointCap.Properties["network_name"]))

	scalableCap := compute.Capabilities["scalable"]
	assert.Len(t, scalableCap.Properties, 3)
	assert.Contains(t, scalableCap.Properties, "max_instances")
	assert.Contains(t, scalableCap.Properties, "min_instances")
	assert.Contains(t, scalableCap.Properties, "default_instances")
	assert.Equal(t, "1", fmt.Sprint(scalableCap.Properties["max_instances"]))
	assert.Equal(t, "1", fmt.Sprint(scalableCap.Properties["min_instances"]))
	assert.Equal(t, "1", fmt.Sprint(scalableCap.Properties["default_instances"]))

	// check workflow
	assert.NotNil(t, topologyTemplate.Workflows)
	assert.Len(t, topologyTemplate.Workflows, 2)
	assert.Contains(t, topologyTemplate.Workflows, "install")
	assert.Contains(t, topologyTemplate.Workflows, "uninstall")

	installWF := topologyTemplate.Workflows["install"]
	assert.Len(t, installWF.Steps, 1)
	assert.Contains(t, installWF.Steps, "Compute_install")
	computeInstallStep := installWF.Steps["Compute_install"]
	assert.Equal(t, "Compute", computeInstallStep.Target)
	assert.Nil(t, computeInstallStep.OnSuccess)
	assert.Equal(t, "", computeInstallStep.Activities[0].CallOperation)
	assert.Equal(t, "", computeInstallStep.Activities[0].SetState)
	assert.Equal(t, "install", computeInstallStep.Activities[0].Delegate)

	uninstallWF := topologyTemplate.Workflows["uninstall"]
	assert.Len(t, installWF.Steps, 1)
	assert.Contains(t, uninstallWF.Steps, "Compute_uninstall")
	computeUninstallStep := uninstallWF.Steps["Compute_uninstall"]
	assert.Equal(t, "Compute", computeUninstallStep.Target)
	assert.Nil(t, computeUninstallStep.OnSuccess)
	assert.Equal(t, "", computeUninstallStep.Activities[0].CallOperation)
	assert.Equal(t, "", computeUninstallStep.Activities[0].SetState)
	assert.Equal(t, "uninstall", computeUninstallStep.Activities[0].Delegate)
}

func parsingSubstitutionMappings(t *testing.T) {
	t.Parallel()
	definition, err := os.Open(filepath.Join("testdata", "test_substitution.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defBytes, err := ioutil.ReadAll(definition)
	if err != nil {
		t.Fatal(err)
	}
	topology := Topology{}
	err = yaml.Unmarshal(defBytes, &topology)
	require.NoError(t, err, "Unexpected failure unmarshalling topology with substitution mappings")

	//log.Printf("%+v\n\n%#v\n", topology, topology)

	// Check Topology types
	assert.Equal(t, "TestService", topology.Metadata[TemplateName])

	mappings := topology.TopologyTemplate.SubstitionMappings
	assert.Equal(t, "org.test.SoftwareAType", mappings.NodeType)
	assert.Equal(t, 4, len(mappings.Properties),
		"Wrong number of properties substitutions")

	// Check property substitutions
	assert.Equal(t, 0, len(mappings.Properties["propertyA"].Mapping),
		"Unexpected mapping found for propertyA")
	assert.Equal(t, ValueAssignmentLiteral, mappings.Properties["propertyA"].Value.Type,
		"Unexpected value type for propertyA")
	require.Equal(t, 2, len(mappings.Properties["propertyB"].Mapping),
		"Wrong number of propertyB mapping")
	assert.Equal(t, "ServerAInstance", mappings.Properties["propertyB"].Mapping[0],
		"Wrong node template in propertyB mapping")
	assert.Equal(t, "propertyB", mappings.Properties["propertyB"].Mapping[1],
		"Wrong property in propertyB mapping")
	assert.Equal(t, 0, len(mappings.Properties["propertyC"].Mapping),
		"Unexpected mapping found for propertyC")
	assert.Equal(t, ValueAssignmentLiteral, mappings.Properties["propertyC"].Value.Type,
		"Unexpected value type for propertyC %+v", mappings.Properties["propertyC"].Value)
	require.Equal(t, 2, len(mappings.Properties["propertyD"].Mapping),
		"Wrong number of propertyD mapping")
	assert.Equal(t, "ServerAInstance", mappings.Properties["propertyD"].Mapping[0],
		"Wrong node template in propertyD mapping")
	assert.Equal(t, "propertyD", mappings.Properties["propertyD"].Mapping[1],
		"Wrong property in propertyD mapping")

	// Check attributes substitutions
	assert.Equal(t, 0, len(mappings.Attributes["attributesA"].Mapping),
		"Unexpected mapping found for attributeA")
	assert.Equal(t, ValueAssignmentLiteral, mappings.Attributes["attributeA"].Value.Type,
		"Unexpected mapping found for attributeA")
	require.Equal(t, 2, len(mappings.Attributes["attributeB"].Mapping),
		"Wrong number of attributeB mapping")
	assert.Equal(t, "ServerAInstance", mappings.Attributes["attributeB"].Mapping[0],
		"Wrong node template in attributeB mapping")
	assert.Equal(t, "attributeB", mappings.Attributes["attributeB"].Mapping[1],
		"Wrong attribute in attributeB mapping")
	assert.Equal(t, 0, len(mappings.Attributes["attributeC"].Mapping),
		"Unexpected mapping found for attributeC")
	assert.Equal(t, ValueAssignmentLiteral, mappings.Attributes["attributeC"].Value.Type,
		"Unexpected value type for attributeC")
	require.Equal(t, 2, len(mappings.Attributes["attributeD"].Mapping),
		"Wrong number of attributeD mapping")
	assert.Equal(t, "ServerAInstance", mappings.Attributes["attributeD"].Mapping[0],
		"Wrong node template in attributeD mapping")
	assert.Equal(t, "attributeD", mappings.Attributes["attributeD"].Mapping[1],
		"Wrong attribute in attributeD mapping")

	// Check capabilities substitutions
	require.Equal(t, 3, len(mappings.Capabilities),
		"Unexpected number of capabilities mapping")
	require.Equal(t, 2, len(mappings.Capabilities["capabilityA"].Mapping),
		"Unexpected size of Mapping for capabilityA")
	assert.Equal(t, "ServerAInstance", mappings.Capabilities["capabilityA"].Mapping[0],
		"Wrong node template in capabilityA mapping")
	assert.Equal(t, "capabilityA", mappings.Capabilities["capabilityA"].Mapping[1],
		"Wrong capability in capabilityA mapping")
	assert.Equal(t, 0, len(mappings.Capabilities["capabilityA"].Properties),
		"Unexpected size of Properties Mapping for capabilityA")
	assert.Equal(t, 0, len(mappings.Capabilities["capabilityA"].Attributes),
		"Unexpected size of Attributes Mapping for capabilityA")
	require.Equal(t, 2, len(mappings.Capabilities["capabilityB"].Mapping),
		"Unexpected size of Mapping for capabilityB")
	assert.Equal(t, "ServerAInstance", mappings.Capabilities["capabilityB"].Mapping[0],
		"Wrong node template in capabilityB mapping")
	assert.Equal(t, "capabilityB", mappings.Capabilities["capabilityB"].Mapping[1],
		"Wrong capability in capabilityB mapping")
	require.Equal(t, 1, len(mappings.Capabilities["capabilityC"].Properties),
		"Unexpected size of Properties Mapping for capabilityC")
	assert.Equal(t, ValueAssignmentLiteral, mappings.Capabilities["capabilityC"].Properties["propertyA"].Type,
		"Unexpected value type for capabilityC propertyA mapping")
	require.Equal(t, 1, len(mappings.Capabilities["capabilityC"].Attributes),
		"Unexpected size of Attributes Mapping for capabilityC")
	assert.Equal(t, ValueAssignmentLiteral, mappings.Capabilities["capabilityC"].Attributes["attributeA"].Type,
		"Unexpected value type for capabilityC attributeA mapping")

	// Check requirements substitutions
	require.Equal(t, 2, len(mappings.Requirements),
		"Unexpected number of requirements mapping")
	require.Equal(t, 2, len(mappings.Requirements["requirementA"].Mapping),
		"Unexpected size of Mapping for requirementA")
	assert.Equal(t, "ServerAInstance", mappings.Requirements["requirementA"].Mapping[0],
		"Wrong node template in requirementA mapping")
	assert.Equal(t, "requirementA", mappings.Requirements["requirementA"].Mapping[1],
		"Wrong requirement in requirementA mapping")
	assert.Equal(t, 0, len(mappings.Requirements["requirementA"].Properties),
		"Unexpected size of Properties Mapping for requirementA")
	assert.Equal(t, 0, len(mappings.Requirements["requirementA"].Attributes),
		"Unexpected size of Attributes Mapping for requirementA")
	require.Equal(t, 2, len(mappings.Requirements["requirementB"].Mapping),
		"Unexpected size of Mapping for requirementB")
	assert.Equal(t, "ServerAInstance", mappings.Requirements["requirementB"].Mapping[0],
		"Wrong node template in requirementB mapping")
	assert.Equal(t, "requirementB", mappings.Requirements["requirementB"].Mapping[1],
		"Wrong requirement in requirementB mapping")

	// Check interfaces mappings
	require.Equal(t, 2, len(mappings.Interfaces),
		"Unexpected number of interfaces mapping")
	assert.Equal(t, "workflowA", mappings.Interfaces["operationA"],
		"Wrong workflow name in operationA interface mapping")
}
