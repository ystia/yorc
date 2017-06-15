package commons

import (
	"context"

	"novaforge.bull.com/starlings-janus/janus/config"
)

// A Generator is used to generate the Terraform infrastructure for a given TOSCA node
type Generator interface {
	// GenerateTerraformInfraForNode generates the Terraform infrastructure file for the given node.
	// It returns 'true' if a file was generated and 'false' otherwise (in case of a infrastructure component
	// already exists for this node and should just be reused).
	// GenerateTerraformInfraForNode can also return a map of outputs names indexed by consul keys into which the outputs results should be stored.
	GenerateTerraformInfraForNode(ctx context.Context, cfg config.Configuration, deploymentID, nodeName string) (bool, map[string]string, error)
}
