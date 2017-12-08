package slurm

import (
	"context"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/helper/sshutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"novaforge.bull.com/starlings-janus/janus/tosca"
	"strconv"
	"strings"
	"sync"
	"time"
)

type defaultExecutor struct {
	generator defaultGenerator
	client    *sshutil.SSHClient
}

const reSallocPending = `^salloc: Pending job allocation (\d+)`
const reSallocGranted = `^salloc: Granted job allocation (\d+)`

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

	// Get SSH client
	SSHConfig := &ssh.ClientConfig{
		User: cfg.Infrastructures[infrastructureName].GetString("user_name"),
		Auth: []ssh.AuthMethod{
			ssh.Password(cfg.Infrastructures[infrastructureName].GetString("password")),
		},
	}

	port, err := strconv.Atoi(cfg.Infrastructures[infrastructureName].GetString("port"))
	if err != nil {
		return errors.Errorf("Invalid Slurm port configuration:%d", port)
	}

	e.client = &sshutil.SSHClient{
		Config: SSHConfig,
		Host:   cfg.Infrastructures[infrastructureName].GetString("url"),
		Port:   port,
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

	if err = e.destroyInfrastructure(ctx, kv, cfg, deploymentID, nodeName, infra, logOptFields); err != nil {
		return err
	}
	return nil
}

func (e *defaultExecutor) createInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, infra *infrastructure, logOptFields events.LogOptionalFields) error {
	events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString("Creating the slurm infrastructure")
	g, ctx := errgroup.WithContext(ctx)
	chAllocationErr := make(chan error)
	for _, compute := range infra.nodes {
		func(comp *nodeAllocation) {
			g.Go(func() error {
				return e.createNodeAllocation(ctx, kv, comp, deploymentID, nodeName, chAllocationErr, logOptFields)
			})
		}(compute)
	}

	if err := g.Wait(); err != nil {
		err = errors.Wrapf(err, "Failed to create slurm infrastructure for deploymentID:%q, node name:%s", deploymentID, nodeName)
		log.Debugf("%+v", err)
		events.WithOptionalFields(logOptFields).NewLogEntry(events.ERROR, deploymentID).RegisterAsString(err.Error())
		return err
	}

	// Handle specific slurm job allocation errors
	select {
	case err := <-chAllocationErr:
		err = errors.Wrapf(err, "Failed to allocate slurm job for deploymentID:%q, node name:%s", deploymentID, nodeName)
		log.Debugf("%+v", err)
		events.WithOptionalFields(logOptFields).NewLogEntry(events.ERROR, deploymentID).RegisterAsString(err.Error())
		return err
	default:
		events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString("Successfully creating the slurm infrastructure")
		return nil
	}
}

func (e *defaultExecutor) destroyInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, infra *infrastructure, logOptFields events.LogOptionalFields) error {
	events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString("Destroying the slurm infrastructure")
	g, ctx := errgroup.WithContext(ctx)
	for _, compute := range infra.nodes {
		func(comp *nodeAllocation) {
			g.Go(func() error {
				return e.destroyNodeAllocation(ctx, kv, comp, deploymentID, nodeName, logOptFields)
			})
		}(compute)
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

func (e *defaultExecutor) createNodeAllocation(ctx context.Context, kv *api.KV, nodeAlloc *nodeAllocation, deploymentID, nodeName string, chAllocationErr chan error, logOptFields events.LogOptionalFields) error {
	events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString(fmt.Sprintf("Creating node allocation for: deploymentID:%q, node name:%q", deploymentID, nodeName))
	// salloc cmd
	var sallocCPUFlag, sallocMemFlag, sallocPartitionFlag, sallocGresFlag string
	if nodeAlloc.cpu != "" {
		sallocCPUFlag = fmt.Sprintf(" -c %s", nodeAlloc.cpu)
	}
	if nodeAlloc.memory != "" {
		sallocMemFlag = fmt.Sprintf(" --mem=%s", nodeAlloc.memory)
	}
	if nodeAlloc.partition != "" {
		sallocPartitionFlag = fmt.Sprintf(" -p %s", nodeAlloc.partition)
	}
	if nodeAlloc.gres != "" {
		sallocGresFlag = fmt.Sprintf(" --gres=%s", nodeAlloc.gres)
	}

	// salloc command can potentially be a long synchronous command according to the slurm cluster state
	// so we run it with a session wrapper with stderr/stdout in order to allow job cancellation if user decides to give up the deployment
	var wg sync.WaitGroup
	sessionWrapper, err := e.client.GetSessionWrapper()
	if err != nil {
		return errors.Wrap(err, "Failed to get an SSH session wrapper")
	}

	chResult := make(chan struct {
		jobID   string
		granted bool
	}, 1)
	var result struct {
		jobID   string
		granted bool
	}
	chOut := make(chan bool, 1)
	chErr := make(chan error)
	go parseSallocResponse(sessionWrapper.Stderr, chResult, chOut, chErr)
	go parseSallocResponse(sessionWrapper.Stdout, chResult, chOut, chErr)
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case result = <-chResult:
			var mes string
			deployments.SetInstanceAttribute(deploymentID, nodeName, nodeAlloc.instanceName, "job_id", result.jobID)
			if result.granted {
				mes = fmt.Sprintf("salloc command returned a GRANTED job allocation notification with job ID:%q", result.jobID)
			} else {
				mes = fmt.Sprintf("salloc command returned a PENDING job allocation notification with job ID:%q", result.jobID)
			}
			events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString(mes)
			return
		case err := <-chErr:
			log.Debug(err.Error())
			events.WithOptionalFields(logOptFields).NewLogEntry(events.ERROR, deploymentID).RegisterAsString(err.Error())
			chAllocationErr <- err
			return
		case <-time.After(5 * time.Second):
			log.Println("timeout elapsed waiting for jobID parsing after slurm allocation request")
			events.WithOptionalFields(logOptFields).NewLogEntry(events.ERROR, deploymentID).RegisterAsString("timeout elapsed waiting for jobID parsing after slurm allocation request")
			chAllocationErr <- err
			return
		}
	}()

	// Listen to potential cancellation in case of pending allocation
	ctxAlloc, cancelAlloc := context.WithCancel(ctx)
	chEnd := make(chan bool)
	go func() {
		select {
		case <-ctx.Done():
			if &result != nil && result.jobID != "" {
				log.Debug("Cancellation message has been sent: the pending job allocation has to be removed")
				if err := cancelJobID(result.jobID, e.client); err != nil {
					log.Printf("[Warning] an error occurred during cancelling jobID:%q", result.jobID)
					return
				}
				// Drain the related jobID compute attribute
				deployments.SetInstanceAttribute(deploymentID, nodeName, nodeAlloc.instanceName, "job_id", "")
				// Cancel salloc comand
				cancelAlloc()
			}
			return
		case <-chEnd:
			return
		}
	}()

	// Run the salloc command
	sallocCmd := strings.TrimSpace(fmt.Sprintf("salloc --no-shell -J %s%s%s%s%s", nodeAlloc.jobName, sallocCPUFlag, sallocMemFlag, sallocPartitionFlag, sallocGresFlag))
	err = sessionWrapper.RunCommand(ctxAlloc, sallocCmd)
	if err != nil {
		return errors.Wrap(err, "Failed to allocate Slurm resource")
	}

	wg.Wait() // we wait until jobID has been set
	// run squeue cmd to get slurm node name
	squeueCmd := fmt.Sprintf("squeue -n %s -j %s --noheader -o \"%%N\"", nodeAlloc.jobName, result.jobID)
	slurmNodeName, err := e.client.RunCommand(squeueCmd)
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
	if cudaVisibleDevice, err = getAttribute(e.client, "cuda_visible_devices", result.jobID, nodeName); err != nil {
		// cuda_visible_device attribute is not mandatory : just log the error
		log.Println("[Warning]: " + err.Error())
	}
	err = deployments.SetInstanceAttribute(deploymentID, nodeName, nodeAlloc.instanceName, "cuda_visible_devices", cudaVisibleDevice)
	if err != nil {
		return errors.Wrapf(err, "Failed to set attribute (cuda_visible_devices) for node name:%q, instance name:%q", nodeName, nodeAlloc.instanceName)
	}

	// Update the instance state
	err = deployments.SetInstanceState(kv, deploymentID, nodeName, nodeAlloc.instanceName, tosca.NodeStateStarted)
	if err != nil {
		return err
	}

	chEnd <- true
	return nil
}

func (e *defaultExecutor) destroyNodeAllocation(ctx context.Context, kv *api.KV, nodeAlloc *nodeAllocation, deploymentID, nodeName string, logOptFields events.LogOptionalFields) error {
	events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString(fmt.Sprintf("Destroying node allocation for: deploymentID:%q, node name:%q, instance name:%q", deploymentID, nodeName, nodeAlloc.instanceName))
	// scancel cmd
	found, jobID, err := deployments.GetInstanceAttribute(kv, deploymentID, nodeName, nodeAlloc.instanceName, "job_id")
	if jobID != "" {
		if err != nil {
			return errors.Wrapf(err, "Failed to retrieve Slurm job ID for node name:%s, instance name:%q: %q:", nodeName, nodeAlloc.instanceName)
		}
		if !found {
			return errors.Errorf("Failed to retrieve Slurm job ID for node name:%s, instance name:%q:", nodeName, nodeAlloc.instanceName)
		}
		if err := cancelJobID(jobID, e.client); err != nil {
			return err
		}
		events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString(fmt.Sprintf("Cancelling Job ID:%q", jobID))

		// Update the instance state
		err = deployments.SetInstanceState(kv, deploymentID, nodeName, nodeAlloc.instanceName, tosca.NodeStateDeleted)
		if err != nil {
			return err
		}
	}
	return nil
}
