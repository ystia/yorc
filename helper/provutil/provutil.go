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

package provutil

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/sshutil"
	"github.com/ystia/yorc/v4/tosca/datatypes"
)

// SanitizeForShell allows to sanitize a string in order to be embedded in shell command
func SanitizeForShell(str string) string {
	return strings.Map(func(r rune) rune {
		// Replace hyphen by underscore
		if r == '-' {
			return '_'
		}
		// Keep underscores
		if r == '_' {
			return r
		}
		// Drop any other non-alphanum rune
		if r < '0' || r > 'z' || r > '9' && r < 'A' || r > 'Z' && r < 'a' {
			return rune(-1)
		}
		return r

	}, str)
}

// GetInstanceBastionHost looks for a DeploysThrough relationship to another host and returns
// a bastion host configuration for the relationship.
// If a node has multiple matching relationships, the first one is chosen. If no relationship
// is found, nil is returned.
func GetInstanceBastionHost(ctx context.Context, deploymentID, nodeName string) (*sshutil.BastionHostConfig, error) {
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

			credsJSON, err := deployments.GetInstanceAttributeValue(ctx, deploymentID, bastionNodeName, "0", "credentials")
			if err != nil {
				return nil, err
			}

			creds := datatypes.Credential{}
			if err := json.Unmarshal([]byte(credsJSON.RawString()), &creds); err != nil {
				return nil, err
			}

			b := &sshutil.BastionHostConfig{Host: ipAddr.String(), User: creds.User}

			if creds.TokenType == "password" {
				b.Password = creds.Token
			}

			keys, err := sshutil.GetKeysFromCredentialsDataType(&creds)
			if err != nil {
				return nil, err
			}
			b.PrivateKeys = keys

			return b, nil
		}
	}
	return nil, nil
}
