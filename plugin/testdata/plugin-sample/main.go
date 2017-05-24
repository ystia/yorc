package main

import (
	"context"
	"log"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/plugin"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

type myDelegateExecutor struct{}

func (d *myDelegateExecutor) ExecDelegate(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	log.Printf("Hello from myDelegateExecutor")
	return nil
}

type myOperationExecutor struct{}

func (d *myOperationExecutor) ExecOperation(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) error {
	log.Printf("Hello from myOperationExecutor")
	return nil
}

func main() {
	def := []byte(`tosca_definitions_version: janus_tosca_simple_yaml_1_0

template_name: janus-my-types
template_author: Janus
template_version: 1.0.0

imports:
  - janus: <janus-types.yml>

artifact_types:
  janus.artifacts.Implementation.MyImplementation:
    derived_from: tosca.artifacts.Implementation
    description: My dummy implementation artifact
    file_ext: [ "myext" ]

node_types:
  janus.my.types.Compute:
    derived_from: tosca.nodes.Compute

  janus.my.types.Soft:
    derived_from: tosca.nodes.SoftwareComponent
    interfaces:
      Standard:
        create: dothis.myext

`)

	plugin.Serve(&plugin.ServeOpts{
		DelegateFunc: func() prov.DelegateExecutor {
			return new(myDelegateExecutor)
		},
		DelegateSupportedTypes: []string{`janus\.my\.types\..*`},
		Definitions: map[string][]byte{
			"my-def.yaml": def,
		},
		OperationFunc: func() prov.OperationExecutor {
			return new(myOperationExecutor)
		},
		OperationSupportedArtifactTypes: []string{"janus.artifacts.Implementation.MyImplementation"},
	})
}
