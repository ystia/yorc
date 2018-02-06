package hostspool

import (
	"context"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

type defaultExecutor struct{}

func (e *defaultExecutor) ExecDelegate(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	cc, err := cfg.GetConsulClient()
	if err != nil {
		return err
	}
	instances, err := tasks.GetInstances(cc.KV(), taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}
	switch strings.ToLower(delegateOperation) {
	case "install":
		for _, instance := range instances {
			deployments.SetInstanceState(cc.KV(), deploymentID, nodeName, instance, tosca.NodeStateCreating)
		}
		err = e.hostsPoolCreate(ctx, cc, cfg, taskID, deploymentID, nodeName)
		if err != nil {
			return err
		}
		for _, instance := range instances {
			deployments.SetInstanceState(cc.KV(), deploymentID, nodeName, instance, tosca.NodeStateCreated)
		}
		return nil
	case "uninstall":
		for _, instance := range instances {
			deployments.SetInstanceState(cc.KV(), deploymentID, nodeName, instance, tosca.NodeStateDeleting)
		}
		err = e.hostsPoolDelete(ctx, cc, cfg, taskID, deploymentID, nodeName)
		if err != nil {
			return err
		}
		for _, instance := range instances {
			deployments.SetInstanceState(cc.KV(), deploymentID, nodeName, instance, tosca.NodeStateDeleting)
		}
		return nil
	}
	return errors.Errorf("operation %q not supported", delegateOperation)
}

func (e *defaultExecutor) hostsPoolCreate(ctx context.Context, cc *api.Client, cfg config.Configuration, taskID, deploymentID, nodeName string) error {
	return errors.New("not implemented")
}

func (e *defaultExecutor) hostsPoolDelete(ctx context.Context, cc *api.Client, cfg config.Configuration, taskID, deploymentID, nodeName string) error {
	return errors.New("not implemented")
}
