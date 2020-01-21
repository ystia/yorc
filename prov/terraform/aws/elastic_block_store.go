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

package aws

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"path"
	"strings"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/helper/sizeutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov/terraform/commons"
)

func (g *awsGenerator) generateEBS(ctx context.Context, nodeParams nodeParams, instanceName string, instanceID int, outputs map[string]string) error {
	err := verifyThatNodeIsTypeOf(ctx, nodeParams, "yorc.nodes.aws.EBSVolume")
	if err != nil {
		return err
	}

	// Check first if Volume already provided
	instancesPrefix := path.Join(consulutil.DeploymentKVPrefix, nodeParams.deploymentID, "topology", "instances", nodeParams.nodeName, instanceName)
	var volumeID string
	volumes, err := deployments.GetStringNodeProperty(ctx, nodeParams.deploymentID, nodeParams.nodeName, "volume_id", false)
	if err != nil {
		return err
	}
	if volumes != "" {
		tabVol := strings.Split(volumes, ",")
		if len(tabVol) > instanceID {
			volumeID = strings.TrimSpace(tabVol[instanceID])

			// as the volume already exists, just add output on volume id
			volumeIDKey := nodeParams.nodeName + "-" + instanceName + "-id" // ex : "BlockStorage-0-id"
			commons.AddOutput(nodeParams.infrastructure, volumeIDKey, &commons.Output{Value: volumeID})
			outputs[path.Join(instancesPrefix, "/attributes/volume_id")] = volumeIDKey
			return nil
		}
	}

	// As no volume is provided, we create one
	ebs := &EBSVolume{}

	// Get EBS props
	err = g.getEBSProperties(ctx, nodeParams, ebs)
	if err != nil {
		return err
	}

	// Create the name for the resource
	name := strings.ToLower(nodeParams.nodeName + "-" + instanceName)
	commons.AddResource(nodeParams.infrastructure, "aws_ebs_volume", name, ebs)

	// Terraform Outputs
	volumeIDKey := nodeParams.nodeName + "-" + instanceName + "-id" // ex : "BlockStorage-0-id"
	volumeIDValue := fmt.Sprintf("${aws_ebs_volume.%s.id}", name)   // ex : ${aws_ebs_volume.blockstorage-0.id}
	volumeARNKey := nodeParams.nodeName + "-" + instanceName + "-arn"
	volumeARNValue := fmt.Sprintf("${aws_ebs_volume.%s.arn}", name)
	commons.AddOutput(nodeParams.infrastructure, volumeIDKey, &commons.Output{Value: volumeIDValue})
	commons.AddOutput(nodeParams.infrastructure, volumeARNKey, &commons.Output{Value: volumeARNValue})

	// Yorc outputs
	outputs[path.Join(instancesPrefix, "/attributes/volume_id")] = volumeIDKey
	outputs[path.Join(instancesPrefix, "/attributes/volume_arn")] = volumeARNKey
	return nil
}

func (g *awsGenerator) getEBSProperties(ctx context.Context, nodeParams nodeParams, ebs *EBSVolume) error {
	var size string
	// Get string params
	stringParams := []struct {
		pAttr        *string
		propertyName string
		mandatory    bool
	}{
		{&ebs.AvailabilityZone, "availability_zone", true},
		{&ebs.SnapshotID, "snapshot_id", false},
		{&ebs.KMSKeyID, "kms_key_id", false},
		{&size, "size", false},
		{&ebs.Type, "volume_type", false},
		{&ebs.IOPS, "iops", false},
	}

	for _, stringParam := range stringParams {
		val, err := deployments.GetStringNodeProperty(ctx, nodeParams.deploymentID, nodeParams.nodeName, stringParam.propertyName, stringParam.mandatory)
		if err != nil {
			return err
		}
		*stringParam.pAttr = val
	}

	// Get bool properties
	val, err := deployments.GetBooleanNodeProperty(ctx, nodeParams.deploymentID, nodeParams.nodeName, "encrypted")
	if err != nil {
		return err
	}
	ebs.Encrypted = val

	// Convert human readable size into GB
	if size != "" {
		// Default size unit is MB
		log.Debugf("Initial size property value (default is MB): %q", size)
		ebs.Size, err = sizeutil.ConvertToGB(size)
		if err != nil {
			return err
		}
		log.Debugf("Computed size (in GB): %d", ebs.Size)
	}

	// If a an encryption key is given, considered the param "encrypted" to be true
	if ebs.KMSKeyID != "" && ebs.Encrypted == false {
		ebs.Encrypted = true
	}

	// Get tags map
	tagsVal, err := deployments.GetNodePropertyValue(ctx, nodeParams.deploymentID, nodeParams.nodeName, "tags")
	if tagsVal != nil && tagsVal.RawString() != "" {
		d, ok := tagsVal.Value.(map[string]interface{})
		if !ok {
			return errors.New("failed to retrieve tags map from Tosca Value: not expected type")
		}

		ebs.Tags = make(map[string]string, len(d))
		for k, v := range d {
			v, ok := v.(string)
			if !ok {
				return errors.Errorf("failed to retrieve string value from tags map from Tosca Value:%q not expected type", v)
			}
			ebs.Tags[k] = v
		}
	}

	return nil
}
