package commons

import "novaforge.bull.com/starlings-janus/janus/config"

// A Generator is used to generate the Terraform infrastructure for a given TOSCA node
type Generator interface {
	// GenerateTerraformInfraForNode generates the Terraform infrastructure file for the given node.
	// It returns 'true' if a file was generated and 'false' otherwise (in case of a infrastructure component
	// already exists for this node and should just be reused).
	GenerateTerraformInfraForNode(cfg config.Configuration, deploymentID, nodeName string) (bool, error)
}
