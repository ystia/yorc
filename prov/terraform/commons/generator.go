package commons

import (
	"context"

	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/events"
)

// A Generator is used to generate the Terraform infrastructure for a given TOSCA node
type Generator interface {
	// GenerateTerraformInfraForNode generates the Terraform infrastructure file for the given node.
	// It returns 'true' if a file was generated and 'false' otherwise (in case of a infrastructure component
	// already exists for this node and should just be reused).
	// GenerateTerraformInfraForNode can also return a map of outputs names indexed by consul keys into which the outputs results should be stored.
	// And a list of environment variables in form "key=value" to be added to the current process environment when running terraform commands.
	// This is particularly useful for adding secrets that should not be in tf files.
	GenerateTerraformInfraForNode(ctx context.Context, cfg config.Configuration, deploymentID, nodeName string) (bool, map[string]string, []string, error)
}

// PreDestroyInfraCallback is a function that is call before destroying an infrastructure. If it returns false the node will not be destroyed.
type PreDestroyInfraCallback func(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, logOptFields events.LogOptionalFields) (bool, error)
