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

	"github.com/mitchellh/mapstructure"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/sshutil"
	"github.com/ystia/yorc/v4/tosca/types"
)

const bastionCapabilityName = "bastion"

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

func getInstanceBastionHostFromRelationship(ctx context.Context, deploymentID, nodeName, reqIdx string) (b *sshutil.BastionHostConfig, err error) {
	bastNode, err := deployments.GetTargetNodeForRequirement(ctx, deploymentID, nodeName, reqIdx)
	if err != nil {
		return
	}
	bastHostNode, err := deployments.GetHostedOnNode(ctx, deploymentID, bastNode)
	if err != nil {
		return
	}
	ip, err := deployments.GetInstanceAttributeValue(ctx, deploymentID, bastHostNode, "0", "ip_address")
	if err != nil {
		return
	}
	port, err := deployments.GetCapabilityPropertyValue(ctx, deploymentID, bastNode, bastionCapabilityName, "port")
	if err != nil {
		return
	}
	creds, err := getBastionHostInstanceCredentials(ctx, deploymentID, bastNode, bastHostNode)
	if err != nil {
		return
	}

	b = &sshutil.BastionHostConfig{Host: ip.String(), Port: port.String(), User: creds.User}
	if creds.TokenType == "password" {
		b.Password = creds.Token
	}
	keys, err := sshutil.GetKeysFromCredentialsDataType(&creds)
	if err != nil {
		return
	}
	b.PrivateKeys = keys
	if b.Port == "" {
		b.Port = "22"
	}
	return
}

func getBastionHostInstanceCredentials(ctx context.Context, deploymentID, bastNode, bastHostNode string) (creds types.Credential, err error) {
	useHostCreds, err := deployments.GetCapabilityPropertyValue(ctx, deploymentID, bastNode, "bastion", "use_host_credentials")
	if err != nil {
		return
	}
	n := bastNode
	c := bastionCapabilityName
	if useHostCreds.String() == "true" {
		n = bastHostNode
		c = "endpoint"
	}
	credsRaw, err := deployments.GetCapabilityPropertyValue(ctx, deploymentID, n, c, "credentials")
	if err != nil {
		return
	}
	if credsRaw.RawString() == "" {
		return
	}
	err = json.Unmarshal([]byte(credsRaw.RawString()), &creds)
	return
}
