package hostspool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"novaforge.bull.com/starlings-janus/janus/helper/labelsutil"

	"github.com/hashicorp/go-multierror"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

type defaultExecutor struct{}

func (e *defaultExecutor) ExecDelegate(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	cc, err := cfg.GetConsulClient()
	if err != nil {
		return err
	}
	// Fill log optional fields for log registration
	wfName, _ := tasks.GetTaskData(cc.KV(), taskID, "workflowName")
	logOptFields := events.LogOptionalFields{
		events.NodeID:        nodeName,
		events.WorkFlowID:    wfName,
		events.InterfaceName: "delegate",
		events.OperationName: delegateOperation,
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
		err = e.hostsPoolCreate(ctx, cc, cfg, taskID, deploymentID, nodeName, logOptFields)
		if err != nil {
			return err
		}
		for _, instance := range instances {
			deployments.SetInstanceState(cc.KV(), deploymentID, nodeName, instance, tosca.NodeStateStarted)
		}
		return nil
	case "uninstall":
		for _, instance := range instances {
			deployments.SetInstanceState(cc.KV(), deploymentID, nodeName, instance, tosca.NodeStateDeleting)
		}
		err = e.hostsPoolDelete(ctx, cc, cfg, taskID, deploymentID, nodeName, logOptFields)
		if err != nil {
			return err
		}
		for _, instance := range instances {
			deployments.SetInstanceState(cc.KV(), deploymentID, nodeName, instance, tosca.NodeStateDeleted)
		}
		return nil
	}
	return errors.Errorf("operation %q not supported", delegateOperation)
}

func (e *defaultExecutor) hostsPoolCreate(ctx context.Context, cc *api.Client, cfg config.Configuration, taskID, deploymentID, nodeName string, logOptFields events.LogOptionalFields) error {
	hpManager := NewManager(cc)

	_, jsonProp, err := deployments.GetNodeProperty(cc.KV(), deploymentID, nodeName, "filters")
	if err != nil {
		return err
	}
	var filtersString []string
	if jsonProp != "" {
		err = json.Unmarshal([]byte(jsonProp), &filtersString)
		if err != nil {
			return errors.Wrapf(err, `failed to parse property "filter" for node %q as json %q`, nodeName, jsonProp)
		}
	}
	filters := make([]labelsutil.Filter, len(filtersString))
	for i := range filtersString {
		filters[i], err = labelsutil.CreateFilter(filtersString[i])
		if err != nil {
			return err
		}
	}

	instances, err := tasks.GetInstances(cc.KV(), taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}
	for _, instance := range instances {
		logOptFields[events.InstanceID] = instance
		hostname, err := hpManager.Allocate(fmt.Sprintf(`allocated for node instance "%s-%s" in deployment %q`, nodeName, instance, deploymentID), filters...)
		if err != nil {
			return err
		}
		err = deployments.SetInstanceAttribute(deploymentID, nodeName, instance, "hostname", hostname)
		if err != nil {
			return err
		}
		host, err := hpManager.GetHost(hostname)
		if err != nil {
			return err
		}
		err = deployments.SetInstanceCapabilityAttribute(deploymentID, nodeName, instance, "endpoint", "ip_address", host.Connection.Host)
		if err != nil {
			return err
		}

		privateAddress, ok := host.Labels["private_address"]
		if !ok {
			privateAddress = host.Connection.Host
			events.WithOptionalFields(logOptFields).
				NewLogEntry(events.WARN, deploymentID).Registerf(`no "private_address" label for host %q, we will use the address from the connection section`, hostname)
		}
		err = deployments.SetInstanceAttribute(deploymentID, nodeName, instance, "ip_address", privateAddress)
		if err != nil {
			return err
		}
		err = deployments.SetInstanceAttribute(deploymentID, nodeName, instance, "private_address", privateAddress)
		if err != nil {
			return err
		}

		for label, value := range host.Labels {
			err = setAttributeFromLabel(deploymentID, nodeName, instance, label, value, "networks", "network_name")
			if err != nil {
				return err
			}
			err = setAttributeFromLabel(deploymentID, nodeName, instance, label, value, "networks", "network_id")
			if err != nil {
				return err
			}
			// This is bad as we split value even if we are not sure that it matches
			err = setAttributeFromLabel(deploymentID, nodeName, instance, label, strings.Split(value, ","), "networks", "addresses")
			if err != nil {
				return err
			}
		}
	}
	delete(logOptFields, events.InstanceID)

	return nil
}

func (e *defaultExecutor) hostsPoolDelete(ctx context.Context, cc *api.Client, cfg config.Configuration, taskID, deploymentID, nodeName string, logOptFields events.LogOptionalFields) error {
	hpManager := NewManager(cc)
	instances, err := tasks.GetInstances(cc.KV(), taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}
	var errs error
	for _, instance := range instances {
		logOptFields[events.InstanceID] = instance
		found, hostname, err := deployments.GetInstanceAttribute(cc.KV(), deploymentID, nodeName, instance, "hostname")
		if err != nil {
			errs = multierror.Append(errs, err)
		}
		if !found {
			events.WithOptionalFields(logOptFields).NewLogEntry(events.WARN, deploymentID).Registerf("instance %q of node %q does not have a registered hostname. This may be due to an error at creation time. Should be checked.", instance, nodeName)
			continue
		}
		err = hpManager.Release(hostname)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	delete(logOptFields, events.InstanceID)
	return errors.Wrap(errs, "errors encountered during hosts pool node release. Some hosts maybe not properly released.")
}

func setAttributeFromLabel(deploymentID, nodeName, instance, label string, value interface{}, prefix, suffix string) error {
	if strings.HasPrefix(label, prefix+".") && strings.HasSuffix(label, "."+suffix) {
		attrName := strings.Replace(strings.Replace(label, prefix+".", prefix+"/", -1), "."+suffix, "/"+suffix, -1)
		err := deployments.SetInstanceAttributeComplex(deploymentID, nodeName, instance, attrName, value)
		if err != nil {
			return err
		}
	}
	return nil
}
