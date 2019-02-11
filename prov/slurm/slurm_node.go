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
	"regexp"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/log"
)

func generateNodeAllocation(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID string, nodeName, instanceName string, infra *infrastructure) error {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if nodeType != "yorc.nodes.slurm.Compute" {
		return errors.Errorf("Unsupported node type for %q: %s", nodeName, nodeType)
	}
	node := &nodeAllocation{instanceName: instanceName}

	// Set the node CPU and memory property from Tosca Compute 'host' capability property
	cpu, err := deployments.GetCapabilityPropertyValue(kv, deploymentID, nodeName, "host", "num_cpus")
	if err != nil {
		return err
	}
	if cpu != nil {
		node.cpu = cpu.RawString()
	}

	memory, err := deployments.GetCapabilityPropertyValue(kv, deploymentID, nodeName, "host", "mem_size")
	if err != nil {
		return err
	}
	// Only take one letter for defining memory unit and remove the blank space between number and unit.
	if memory != nil && memory.RawString() != "" {
		re := regexp.MustCompile("[[:digit:]]*[[:upper:]]{1}")
		node.memory = re.FindString(strings.Replace(memory.RawString(), " ", "", -1))
	}

	user, err := deployments.GetCapabilityPropertyValue(kv, deploymentID, nodeName, "endpoint", "credentials", "user")
	if err != nil {
		return err
	}
	if user != nil && user.RawString() != "" {
		log.Debugf("Got user name from enpoint capability, property credentials : %s", user.RawString())
		node.userName = user.RawString()
	}

	token_type, err := deployments.GetCapabilityPropertyValue(kv, deploymentID, nodeName, "endpoint", "credentials", "token_type")
	if token_type != nil && token_type.RawString() != "" {
		log.Debugf("Got token_type from enpoint capability, property credentials : %s", token_type.RawString())
	}

	if token_type.RawString() == "password" {

		pwd, err := deployments.GetCapabilityPropertyValue(kv, deploymentID, nodeName, "endpoint", "credentials", "keys", "0")
		if err != nil {
			return err
		}
		if pwd != nil && pwd.RawString() != "" {
			log.Debugf("Got pwd from enpoint capability, property credentials : %s", pwd.RawString())
			node.password = pwd.RawString()
		}

	} else if token_type.RawString() == "private_key" {

		privateKey, err := deployments.GetCapabilityPropertyValue(kv, deploymentID, nodeName, "endpoint", "credentials", "keys", "0")
		if err != nil {
			return err
		}
		if privateKey != nil && privateKey.RawString() != "" {
			log.Debugf("Got private key from enpoint capability, property credentials : %s", privateKey.RawString())
			node.privateKey = privateKey.RawString()
		}
	} else if node.userName != "" {
		// If user name provided, then password or private_key expected as token_type
		return errors.Errorf("Unsupported token_type in compute endpoint credentials %s. One of password or private_key extected", token_type.RawString())
	}

	// Set the job name property
	// first: with the prop
	jobName, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "job_name")
	if err != nil {
		return err
	}
	if jobName == nil || jobName.RawString() == "" {
		// Second: with the config
		node.jobName = cfg.Infrastructures[infrastructureName].GetString("default_job_name")
		if node.jobName == "" {
			// Third: with the deploymentID
			node.jobName = deploymentID
		}
	} else {
		node.jobName = jobName.RawString()
	}

	// Set the node gres property from Tosca slurm.Compute property
	gres, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "gres")
	if err != nil {
		return err
	}
	if gres != nil {
		node.gres = gres.RawString()
	}
	// Set the node constraint property from Tosca slurm.Compute property
	constraint, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "constraint")
	if err != nil {
		return err
	}
	if constraint != nil {
		node.constraint = constraint.RawString()
	}
	// Set the node partition property from Tosca slurm.Compute property
	partition, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "partition")
	if err != nil {
		return err
	}
	if partition != nil {
		node.partition = partition.RawString()
	}
	// ToDo - #281
	// Check that is userName provided, then password or privateKey provided also
	// Otherwise, raise error, event ...

	infra.nodes = append(infra.nodes, node)
	return nil
}
