package openstack

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform"
	"novaforge.bull.com/starlings-janus/janus/registry"
)

func init() {
	reg := registry.GetRegistry()
	reg.RegisterDelegates([]string{`janus\.nodes\.openstack\..*`}, terraform.NewExecutor(&osGenerator{}, preDestroyInfraCallback), registry.BuiltinOrigin)
}

func preDestroyInfraCallback(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, logOptFields events.LogOptionalFields) (bool, error) {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return false, err
	}
	// TODO  consider making this generic: references to OpenStack should not be found here.
	if nodeType == "janus.nodes.openstack.BlockStorage" {
		var deletable string
		var found bool
		found, deletable, err = deployments.GetNodeProperty(kv, deploymentID, nodeName, "deletable")
		if err != nil {
			return false, err
		}
		if !found || strings.ToLower(deletable) != "true" {
			// False by default
			msg := fmt.Sprintf("Node %q is a BlockStorage without the property 'deletable' do not destroy it...", nodeName)
			log.Debug(msg)
			events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString(msg)
			return false, nil
		}
	}
	return true, nil
}
