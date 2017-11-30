package slurm

import (
	"context"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/helper/sshutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"novaforge.bull.com/starlings-janus/janus/tosca"
	"strings"
)

type defaultExecutor struct {
	generator defaultGenerator
}

func newExecutor(generator defaultGenerator) prov.DelegateExecutor {
	return &defaultExecutor{generator: generator}
}

func (e *defaultExecutor) ExecDelegate(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	consulClient, err := cfg.GetConsulClient()
	if err != nil {
		return err
	}
	kv := consulClient.KV()
	instances, err := tasks.GetInstances(kv, taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}

	// Fill log optional fields for log registration
	wfName, _ := tasks.GetTaskData(kv, taskID, "workflowName")
	logOptFields := events.LogOptionalFields{
		events.NodeID:        nodeName,
		events.WorkFlowID:    wfName,
		events.InterfaceName: "delegate",
		events.OperationName: delegateOperation,
	}

	operation := strings.ToLower(delegateOperation)
	switch {
	case operation == "install":
		err = e.installNode(ctx, kv, cfg, deploymentID, nodeName, instances, logOptFields, operation)
	case operation == "uninstall":
		err = e.uninstallNode(ctx, kv, cfg, deploymentID, nodeName, instances, logOptFields, operation)
	default:
		return errors.Errorf("Unsupported operation %q", delegateOperation)
	}
	return err
}

func (e *defaultExecutor) installNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, instances []string, logOptFields events.LogOptionalFields, operation string) error {
	for _, instance := range instances {
		err := deployments.SetInstanceState(kv, deploymentID, nodeName, instance, tosca.NodeStateCreating)
		if err != nil {
			return err
		}
	}
	infra, err := e.generator.generateInfrastructure(ctx, kv, cfg, deploymentID, nodeName, operation)
	if err != nil {
		return err
	}
	if err = e.createInfrastructure(ctx, kv, cfg, deploymentID, nodeName, infra, logOptFields); err != nil {
		return err
	}
	for _, instance := range instances {
		err := deployments.SetInstanceState(kv, deploymentID, nodeName, instance, tosca.NodeStateStarted)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *defaultExecutor) uninstallNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, instances []string, logOptFields events.LogOptionalFields, operation string) error {
	for _, instance := range instances {
		err := deployments.SetInstanceState(kv, deploymentID, nodeName, instance, tosca.NodeStateDeleting)
		if err != nil {
			return err
		}
	}
	infra, err := e.generator.generateInfrastructure(ctx, kv, cfg, deploymentID, nodeName, operation)
	if err != nil {
		return err
	}

	log.Debugf("infrastructure =%+v", infra)
	if err = e.destroyInfrastructure(ctx, kv, cfg, deploymentID, nodeName, infra, logOptFields); err != nil {
		return err
	}
	for _, instance := range instances {
		err := deployments.SetInstanceState(kv, deploymentID, nodeName, instance, tosca.NodeStateDeleted)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *defaultExecutor) createInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, infra *infrastructure, logOptFields events.LogOptionalFields) error {
	events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString("Creating the slurm infrastructure")
	var g errgroup.Group
	for _, compute := range infra.nodes {
		func(comp *nodeAllocation) {
			g.Go(func() error {
				return e.createNodeAllocation(ctx, comp, infra.provider.session, deploymentID, nodeName, logOptFields)
			})
		}(&compute)
	}

	if err := g.Wait(); err != nil {
		err = errors.Wrapf(err, "Failed to create slurm infrastructure for deploymentID:%q, node name:%s", deploymentID, nodeName)
		log.Debugf("%+v", err)
		events.WithOptionalFields(logOptFields).NewLogEntry(events.ERROR, deploymentID).RegisterAsString(err.Error())
		return err
	}

	events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString("Successfully creating the slurm infrastructure")
	return nil
}

func (e *defaultExecutor) destroyInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, infra *infrastructure, logOptFields events.LogOptionalFields) error {
	events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString("Destroying the slurm infrastructure")
	var g errgroup.Group
	for _, compute := range infra.nodes {
		func(comp *nodeAllocation) {
			g.Go(func() error {
				return e.destroyNodeAllocation(ctx, kv, comp, infra.provider.session, deploymentID, nodeName, logOptFields)
			})
		}(&compute)
	}

	if err := g.Wait(); err != nil {
		err = errors.Wrapf(err, "Failed to destroy slurm infrastructure for deploymentID:%q, node name:%s", deploymentID, nodeName)
		log.Debugf("%+v", err)
		events.WithOptionalFields(logOptFields).NewLogEntry(events.ERROR, deploymentID).RegisterAsString(err.Error())
		return err
	}

	events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString("Successfully destroying the slurm infrastructure")
	return nil
}

func (e *defaultExecutor) createNodeAllocation(ctx context.Context, nodeAlloc *nodeAllocation, s sshutil.Session, deploymentID, nodeName string, logOptFields events.LogOptionalFields) error {
	events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString(fmt.Sprintf("Creating node allocation for: deploymentID:%q, node name:%q", deploymentID, nodeName))
	// salloc cmd
	var sallocCPUFlag, sallocMemFlag, sallocPartitionFlag, sallocGresFlag string
	if nodeAlloc.cpu != "" {
		sallocCPUFlag = fmt.Sprintf("-c %s", nodeAlloc.cpu)
	}
	if nodeAlloc.memory != "" {
		sallocMemFlag = fmt.Sprintf("--mem=%s", nodeAlloc.memory)
	}
	if nodeAlloc.partition != "" {
		sallocMemFlag = fmt.Sprintf("-p %s", nodeAlloc.partition)
	}
	if nodeAlloc.gres != "" {
		sallocMemFlag = fmt.Sprintf("--gres=%s", nodeAlloc.gres)
	}

	sallocCmd := fmt.Sprintf("salloc --no-shell -J %s %s %s %s %s", nodeAlloc.jobName, sallocCPUFlag, sallocMemFlag, sallocPartitionFlag, sallocGresFlag)
	sallocOutput, err := s.RunCommand(sallocCmd)
	if err != nil {
		return errors.Wrapf(err, "Failed to allocate Slurm resource: %q:", sallocOutput)
	}
	// set the jobID
	split := strings.Split(sallocOutput, " ")
	jobID := strings.TrimSpace(split[len(split)-1])
	err = deployments.SetInstanceCapabilityAttribute(deploymentID, nodeName, nodeAlloc.instanceName, "endpoint", "job_id", jobID)
	if err != nil {
		return errors.Wrapf(err, "Failed to set capability attribute (job_id) for node name:%q, instance name:%q", nodeName, nodeAlloc.instanceName)
	}
	events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString(fmt.Sprintf("Allocating Job ID:%q", jobID))

	// run squeue cmd to get slurm node name
	squeueCmd := fmt.Sprintf("squeue -n %s -j %s --noheader -o \"%%N\"", nodeAlloc.jobName, jobID)
	slurmNodeName, err := s.RunCommand(squeueCmd)
	if err != nil {
		return errors.Wrapf(err, "Failed to retrieve Slurm node name: %q:", slurmNodeName)
	}
	slurmNodeName = strings.Trim(slurmNodeName, "\" \t\n")
	err = deployments.SetInstanceCapabilityAttribute(deploymentID, nodeName, nodeAlloc.instanceName, "endpoint", "ip_address", slurmNodeName)
	if err != nil {
		return errors.Wrapf(err, "Failed to set capability attribute (ip_address) for node name:%s, instance name:%q", nodeName, nodeAlloc.instanceName)
	}
	err = deployments.SetInstanceAttribute(deploymentID, nodeName, nodeAlloc.instanceName, "ip_address", slurmNodeName)
	if err != nil {
		return errors.Wrapf(err, "Failed to set attribute (ip_address) for node name:%q, instance name:%q", nodeName, nodeAlloc.instanceName)
	}
	err = deployments.SetInstanceAttribute(deploymentID, nodeName, nodeAlloc.instanceName, "node_name", slurmNodeName)
	if err != nil {
		return errors.Wrapf(err, "Failed to set attribute (node_name) for node name:%q, instance name:%q", nodeName, nodeAlloc.instanceName)
	}

	// Get cuda_visible_device attribute
	var cudaVisibleDevice string
	if cudaVisibleDevice, err = getAttribute(s, "cuda_visible_devices", jobID, nodeName); err != nil {
		// cuda_visible_device attribute is not mandatory : just log the error
		log.Println("[Warning]: " + err.Error())
	}
	err = deployments.SetInstanceAttribute(deploymentID, nodeName, nodeAlloc.instanceName, "cuda_visible_devices", cudaVisibleDevice)
	if err != nil {
		return errors.Wrapf(err, "Failed to set attribute (cuda_visible_devices) for node name:%q, instance name:%q", nodeName, nodeAlloc.instanceName)
	}

	return nil
}

func (e *defaultExecutor) destroyNodeAllocation(ctx context.Context, kv *api.KV, nodeAlloc *nodeAllocation, s sshutil.Session, deploymentID, nodeName string, logOptFields events.LogOptionalFields) error {
	events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString(fmt.Sprintf("Destroying node allocation for: deploymentID:%q, node name:%q", deploymentID, nodeName))
	// scancel cmd
	found, jobID, err := deployments.GetInstanceCapabilityAttribute(kv, deploymentID, nodeName, nodeAlloc.instanceName, "endpoint", "job_id")
	if err != nil {
		return errors.Wrapf(err, "Failed to retrieve Slurm job ID for node name:%s, instance name:%q: %q:", nodeName, nodeAlloc.instanceName)
	}
	if !found {
		return errors.Errorf("Failed to retrieve Slurm job ID for node name:%s, instance name:%q:", nodeName, nodeAlloc.instanceName)
	}
	scancelCmd := fmt.Sprintf("scancel %s", jobID)
	sCancelOutput, err := s.RunCommand(scancelCmd)
	if err != nil {
		return errors.Wrapf(err, "Failed to cancel Slurm job: %s:", sCancelOutput)
	}
	events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString(fmt.Sprintf("Cancelling Job ID:%q", jobID))
	return nil
}
