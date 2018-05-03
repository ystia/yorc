// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package slurm

import (
	"context"
	"fmt"
	"strings"

	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/sshutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/tosca"
	"golang.org/x/sync/errgroup"
)

type defaultExecutor struct {
	generator defaultGenerator
	client    *sshutil.SSHClient
}

type allocationResponse struct {
	jobID   string
	granted bool
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
	logOptFields, ok := events.FromContext(ctx)
	if !ok {
		return errors.New("Missing contextual log optionnal fields")
	}
	logOptFields[events.NodeID] = nodeName
	logOptFields[events.ExecutionID] = taskID
	logOptFields[events.OperationName] = delegateOperation
	logOptFields[events.InterfaceName] = "delegate"
	ctx = events.NewContext(ctx, logOptFields)

	instances, err := tasks.GetInstances(kv, taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}

	e.client, err = GetSSHClient(cfg)
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, deploymentID).RegisterAsString(err.Error())
		return err
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
	return e.createInfrastructure(ctx, kv, cfg, deploymentID, nodeName, infra)
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

	return e.destroyInfrastructure(ctx, kv, cfg, deploymentID, nodeName, infra)
}

func (e *defaultExecutor) createInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, infra *infrastructure) error {
	events.WithContextOptionalFields(ctx).NewLogEntry(events.INFO, deploymentID).RegisterAsString("Creating the slurm infrastructure")
	var g errgroup.Group
	for _, compute := range infra.nodes {
		func(comp *nodeAllocation) {
			g.Go(func() error {
				return e.createNodeAllocation(ctx, kv, comp, deploymentID, nodeName)
			})
		}(compute)
	}

	if err := g.Wait(); err != nil {
		err = errors.Wrapf(err, "Failed to create slurm infrastructure for deploymentID:%q, node name:%s", deploymentID, nodeName)
		log.Debugf("%+v", err)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, deploymentID).RegisterAsString(err.Error())
		return err
	}

	events.WithContextOptionalFields(ctx).NewLogEntry(events.INFO, deploymentID).RegisterAsString("Successfully creating the slurm infrastructure")
	return nil
}

func (e *defaultExecutor) destroyInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, infra *infrastructure) error {
	events.WithContextOptionalFields(ctx).NewLogEntry(events.INFO, deploymentID).RegisterAsString("Destroying the slurm infrastructure")
	var g errgroup.Group
	for _, compute := range infra.nodes {
		func(comp *nodeAllocation) {
			g.Go(func() error {
				return e.destroyNodeAllocation(ctx, kv, comp, deploymentID, nodeName)
			})
		}(compute)
	}

	if err := g.Wait(); err != nil {
		err = errors.Wrapf(err, "Failed to destroy slurm infrastructure for deploymentID:%q, node name:%s", deploymentID, nodeName)
		log.Debugf("%+v", err)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, deploymentID).RegisterAsString(err.Error())
		return err
	}

	events.WithContextOptionalFields(ctx).NewLogEntry(events.INFO, deploymentID).RegisterAsString("Successfully destroying the slurm infrastructure")
	return nil
}

func (e *defaultExecutor) createNodeAllocation(ctx context.Context, kv *api.KV, nodeAlloc *nodeAllocation, deploymentID, nodeName string) error {
	events.WithContextOptionalFields(ctx).NewLogEntry(events.INFO, deploymentID).RegisterAsString(fmt.Sprintf("Creating node allocation for: deploymentID:%q, node name:%q", deploymentID, nodeName))
	// salloc cmd
	var sallocCPUFlag, sallocMemFlag, sallocPartitionFlag, sallocGresFlag, sallocConstraintFlag string
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
	if nodeAlloc.constraint != "" {
		sallocConstraintFlag = fmt.Sprintf(" --constraint=%q", nodeAlloc.constraint)
	}

	// salloc command can potentially be a long synchronous command according to the slurm cluster state
	// so we run it with a session wrapper with stderr/stdout in order to allow job cancellation if user decides to give up the deployment
	var wg sync.WaitGroup
	sessionWrapper, err := e.client.GetSessionWrapper()
	if err != nil {
		return errors.Wrap(err, "Failed to get an SSH session wrapper")
	}

	// We keep these both two channels open as 2 routines are concurrently and potentially able to send messages on them and we only get the first sent message. They will be garbage collected.
	chErr := make(chan error)
	chAllocResp := make(chan allocationResponse)
	var allocResponse allocationResponse
	go parseSallocResponse(sessionWrapper.Stderr, chAllocResp, chErr)
	go parseSallocResponse(sessionWrapper.Stdout, chAllocResp, chErr)
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case allocResponse = <-chAllocResp:
			var mes string
			deployments.SetInstanceAttribute(deploymentID, nodeName, nodeAlloc.instanceName, "job_id", allocResponse.jobID)
			if allocResponse.granted {
				mes = fmt.Sprintf("salloc command returned a GRANTED job allocation notification with job ID:%q", allocResponse.jobID)
			} else {
				mes = fmt.Sprintf("salloc command returned a PENDING job allocation notification with job ID:%q", allocResponse.jobID)
			}
			events.WithContextOptionalFields(ctx).NewLogEntry(events.INFO, deploymentID).RegisterAsString(mes)
			return
		case err := <-chErr:
			log.Debug(err.Error())
			events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, deploymentID).RegisterAsString(err.Error())
			return
		case <-time.After(30 * time.Second):
			events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, deploymentID).RegisterAsString("timeout elapsed waiting for jobID parsing after slurm allocation request")
			return
		}
	}()

	// Listen to potential cancellation in case of pending allocation
	ctxAlloc, cancelAlloc := context.WithCancel(ctx)
	chEnd := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			if &allocResponse != nil && allocResponse.jobID != "" {
				log.Debug("Cancellation message has been sent: the pending job allocation has to be removed")
				if err := cancelJobID(allocResponse.jobID, e.client); err != nil {
					log.Printf("[Warning] an error occurred during cancelling jobID:%q", allocResponse.jobID)
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
	sallocCmd := strings.TrimSpace(fmt.Sprintf("salloc --no-shell -J %s%s%s%s%s%s", nodeAlloc.jobName, sallocCPUFlag, sallocMemFlag, sallocPartitionFlag, sallocGresFlag, sallocConstraintFlag))
	err = sessionWrapper.RunCommand(ctxAlloc, sallocCmd)
	if err != nil {
		return errors.Wrap(err, "Failed to allocate Slurm resource")
	}

	wg.Wait() // we wait until jobID has been set
	// retrieve nodename and partition
	var nodeAndPartitionAttrs []string
	if nodeAndPartitionAttrs, err = getAttributes(e.client, "node_partition", allocResponse.jobID); err != nil {
		return err
	}

	err = deployments.SetInstanceCapabilityAttribute(deploymentID, nodeName, nodeAlloc.instanceName, "endpoint", "ip_address", nodeAndPartitionAttrs[0])
	if err != nil {
		return errors.Wrapf(err, "Failed to set capability attribute (ip_address) for node name:%s, instance name:%q", nodeName, nodeAlloc.instanceName)
	}
	err = deployments.SetInstanceAttribute(deploymentID, nodeName, nodeAlloc.instanceName, "ip_address", nodeAndPartitionAttrs[0])
	if err != nil {
		return errors.Wrapf(err, "Failed to set attribute (ip_address) for node name:%q, instance name:%q", nodeName, nodeAlloc.instanceName)
	}
	err = deployments.SetInstanceAttribute(deploymentID, nodeName, nodeAlloc.instanceName, "node_name", nodeAndPartitionAttrs[0])
	if err != nil {
		return errors.Wrapf(err, "Failed to set attribute (node_name) for node name:%q, instance name:%q", nodeName, nodeAlloc.instanceName)
	}
	err = deployments.SetInstanceAttribute(deploymentID, nodeName, nodeAlloc.instanceName, "partition", nodeAndPartitionAttrs[1])
	if err != nil {
		return errors.Wrapf(err, "Failed to set attribute (partition) for node name:%q, instance name:%q", nodeName, nodeAlloc.instanceName)
	}

	// Get cuda_visible_device attribute
	var cudaVisibleDeviceAttrs []string
	if cudaVisibleDeviceAttrs, err = getAttributes(e.client, "cuda_visible_devices", allocResponse.jobID, nodeName); err != nil {
		// cuda_visible_device attribute is not mandatory : just log the error
		log.Println("[Warning]: " + err.Error())
	} else {
		err = deployments.SetInstanceAttribute(deploymentID, nodeName, nodeAlloc.instanceName, "cuda_visible_devices", cudaVisibleDeviceAttrs[0])
		if err != nil {
			return errors.Wrapf(err, "Failed to set attribute (cuda_visible_devices) for node name:%q, instance name:%q", nodeName, nodeAlloc.instanceName)
		}
	}

	// Update the instance state
	err = deployments.SetInstanceState(kv, deploymentID, nodeName, nodeAlloc.instanceName, tosca.NodeStateStarted)
	if err != nil {
		return err
	}

	close(chEnd)
	return nil
}

func (e *defaultExecutor) destroyNodeAllocation(ctx context.Context, kv *api.KV, nodeAlloc *nodeAllocation, deploymentID, nodeName string) error {
	events.WithContextOptionalFields(ctx).NewLogEntry(events.INFO, deploymentID).RegisterAsString(fmt.Sprintf("Destroying node allocation for: deploymentID:%q, node name:%q, instance name:%q", deploymentID, nodeName, nodeAlloc.instanceName))
	// scancel cmd
	found, jobID, err := deployments.GetInstanceAttribute(kv, deploymentID, nodeName, nodeAlloc.instanceName, "job_id")
	if jobID != "" {
		if err != nil {
			return errors.Wrapf(err, "Failed to retrieve Slurm job ID for node name:%q, instance name:%q", nodeName, nodeAlloc.instanceName)
		}
		if !found {
			log.Printf("[Warning]: No job ID found for node name:%q, instance name:%q. We assume it has already been deleted", nodeName, nodeAlloc.instanceName)
		} else {
			if err := cancelJobID(jobID, e.client); err != nil {
				return err
			}
			events.WithContextOptionalFields(ctx).NewLogEntry(events.INFO, deploymentID).RegisterAsString(fmt.Sprintf("Cancelling Job ID:%q", jobID))
		}
	}
	// Update the instance state
	err = deployments.SetInstanceState(kv, deploymentID, nodeName, nodeAlloc.instanceName, tosca.NodeStateDeleted)
	if err != nil {
		return err
	}
	return nil
}
