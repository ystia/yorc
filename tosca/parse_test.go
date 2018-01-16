package tosca

import (
	"fmt"
	"io/ioutil"
	_ "log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestGroupedParsingParallel(t *testing.T) {
	t.Run("groupParsing", func(t *testing.T) {
		t.Run("TestParsing", parsing)
	})
}

func parsing(t *testing.T) {
	t.Parallel()
	definition, err := os.Open(filepath.Join("..", "testdata", "deployment", "dep.yaml"))
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
	assert.Equal(t, "Test", topology.Name)
	assert.Equal(t, "0.1.0-SNAPSHOT", topology.Version)
	assert.Equal(t, "admin", topology.Author)

	assert.Len(t, topology.Imports, 1)
	importDefMap := topology.Imports[0]
	assert.Equal(t, "1.0.0-ALIEN11", importDefMap["tosca-normative-types"].File)

	//Check topology template
	topologyTemplate := topology.TopologyTemplate
	assert.NotNil(t, topologyTemplate)

	// Check node type
	assert.Len(t, topologyTemplate.NodeTemplates, 1)
	assert.Contains(t, topologyTemplate.NodeTemplates, "Compute")

	compute := topologyTemplate.NodeTemplates["Compute"]
	assert.NotNil(t, compute)

	assert.Equal(t, "janus.nodes.openstack.Compute", compute.Type)

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
