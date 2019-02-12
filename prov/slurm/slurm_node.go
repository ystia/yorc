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

	// Check if user credentials provided in node definition
	user, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "user_account", "user")
	if err != nil {
		return err
	}
	if user != nil {
		node.userName = user.RawString()
		log.Debugf("Got user name from user_account property : %s", node.userName)
	}

	// Check for token-type
	token_type, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "user_account", "token_type")
	if err != nil {
		return err
	}
	if token_type != nil {
		switch token_type.RawString() {
		case "password":
			pwd, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "user_account", "token")
			if err != nil {
				return err
			}
			if pwd != nil {
				node.password = pwd.RawString()
				log.Debugf("Got password from user_account property : %s", node.password)
			}
		case "private_key":
			privateKey, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "user_account", "keys", "0")
			if err != nil {
				return err
			}
			if privateKey != nil {
				node.privateKey = privateKey.RawString()
				log.Debugf("Got private key from user_account property : %s", node.privateKey)
			}
		default:
			// password or private_key expected as token_type
			return errors.Errorf("Unsupported token_type in compute endpoint credentials %s. One of password or private_key extected", token_type.RawString())
		}
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
