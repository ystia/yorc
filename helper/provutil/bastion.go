// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package provutil

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/mitchellh/mapstructure"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/sshutil"
	"github.com/ystia/yorc/v4/tosca/types"
)

// GetInstanceBastionHost returns a sshutil.BastionHostConfig for a given hosts or nil when no
// bastion host is specified.
//
// The function first looks for an explicit definition in a yorc.capabilities.Endpoint.ProvisioningAdmin
// capability. If there is not explicit definition, it looks for a DeploysThrough relationship to another
// host and returns a bastion host configuration for the first relationship that was found. If no relationship
// is found, nil is returned instead.
func GetInstanceBastionHost(ctx context.Context, deploymentID, nodeName string) (*sshutil.BastionHostConfig, error) {
	bastDefRaw, err := deployments.GetCapabilityPropertyValue(ctx, deploymentID, nodeName, "endpoint", "bastion")
	if err != nil {
		return nil, err
	}
	var bastDef types.ProvisioningBastion
	if bastDefRaw != nil && bastDefRaw.RawString() != "" {
		if err := mapstructure.Decode(bastDefRaw.Value, &bastDef); err != nil {
			return nil, err
		}
		if bastDef.Use == "true" {
			return getInstanceBastionHostFromEndpoint(ctx, deploymentID, nodeName, bastDef)
		}
	}

	deps, err := deployments.GetRequirementsKeysByTypeForNode(ctx, deploymentID, nodeName, "dependency")
	if err != nil {
		return nil, err
	}
	for _, depPrefix := range deps {
		depIdx := deployments.GetRequirementIndexFromRequirementKey(ctx, depPrefix)
		rel, err := deployments.GetRelationshipForRequirement(ctx, deploymentID, nodeName, depIdx)
		if err != nil {
			return nil, err
		}

		if rel == "yorc.relationships.DeploysThrough" {
			return getInstanceBastionHostFromRelationship(ctx, deploymentID, nodeName, depIdx)
		}
	}

	return nil, nil
}

func getInstanceBastionHostFromEndpoint(ctx context.Context, deploymentID, nodeName string, bas types.ProvisioningBastion) (*sshutil.BastionHostConfig, error) {
	if bas.Port == "" {
		bas.Port = "22"
	}

	credsRaw, err := deployments.GetCapabilityPropertyValue(ctx, deploymentID, nodeName, "endpoint", "bastion", "credentials")
	if err != nil {
		return nil, err
	}
	_ = credsRaw

	b := &sshutil.BastionHostConfig{Host: bas.Host, Port: bas.Port, User: bas.Credentials.User, PrivateKeys: make(map[string]*sshutil.PrivateKey)}

	b.PrivateKeys, err = sshutil.GetKeysFromCredentialsDataType(&bas.Credentials)
	if err != nil {
		return nil, err
	}

	if bas.Credentials.TokenType == "password" {
		b.Password = bas.Credentials.Token
	}
	return b, nil
}

func getInstanceBastionHostFromRelationship(ctx context.Context, deploymentID, nodeName, depIdx string) (*sshutil.BastionHostConfig, error) {
	bastionNodeName, err := deployments.GetTargetNodeForRequirement(ctx, deploymentID, nodeName, depIdx)
	if err != nil {
		return nil, err
	}

	bastionNodeType, err := deployments.GetNodeType(ctx, deploymentID, bastionNodeName)
	if err != nil {
		return nil, err
	}
	if bastionNodeType != "yorc.nodes.SSHBastionHost" {
		return nil, errors.New("unspported node type for DeploysThrough relationship")
	}

	ipAddr, err := deployments.GetInstanceAttributeValue(ctx, deploymentID, bastionNodeName, "0", "ip_address")
	if err != nil {
		return nil, err
	}

	port, err := deployments.GetInstanceAttributeValue(ctx, deploymentID, bastionNodeName, "0", "port")
	if err != nil {
		return nil, err
	}

	credsRaw, err := deployments.GetInstanceAttributeValue(ctx, deploymentID, bastionNodeName, "0", "credentials")
	if err != nil {
		return nil, err
	}

	creds := types.Credential{}
	if credsRaw.RawString() != "" {
		if err := json.Unmarshal([]byte(credsRaw.RawString()), &creds); err != nil {
			return nil, err
		}
	}

	b := &sshutil.BastionHostConfig{Host: ipAddr.String(), Port: port.String(), User: creds.User}

	if creds.TokenType == "password" {
		b.Password = creds.Token
	}

	keys, err := sshutil.GetKeysFromCredentialsDataType(&creds)
	if err != nil {
		return nil, err
	}
	b.PrivateKeys = keys

	if b.Port == "" {
		b.Port = "22"
	}

	return b, nil
}
