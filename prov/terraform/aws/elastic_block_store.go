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

func (g *awsGenerator) generateEBS(nodeParams nodeParams, instanceName string, instanceID int, outputs map[string]string) error {
	err := verifyThatNodeIsTypeOf(nodeParams, "yorc.nodes.aws.EBSVolume")
	if err != nil {
		return err
	}

	ebs := &EBSVolume{}

	// Get string params
	var size, deviceName, volumes string
	g.getEBSProperties(nodeParams, ebs, &size, &deviceName, &volumes)

	var volumeID string
	if volumes != "" {
		tabVol := strings.Split(volumes, ",")
		if len(tabVol) > instanceID {
			volumeID = strings.TrimSpace(tabVol[instanceID])
		}
	}

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

	// Create the name for the ressource
	name := strings.ToLower(nodeParams.nodeName + "-" + instanceName)
	// name = strings.Replace(name, "_", "-", -1)
	commons.AddResource(nodeParams.infrastructure, "aws_ebs_volume", name, ebs)

	// Terraform Outputs
	volumeID = nodeParams.nodeName + "-" + instanceName + "-id"   // ex : "BlockStorage-0-id"
	volumeIDValue := fmt.Sprintf("${aws_ebs_volume.%s.id}", name) // ex : ${aws_ebs_volume.blockstorage-0.id}
	volumeARN := nodeParams.nodeName + "-" + instanceName + "-arn"
	volumeARNValue := fmt.Sprintf("${aws_ebs_volume.%s.arn}", name)
	commons.AddOutput(nodeParams.infrastructure, volumeID, &commons.Output{Value: volumeIDValue})
	commons.AddOutput(nodeParams.infrastructure, volumeARN, &commons.Output{Value: volumeARNValue})

	// Yorc outputs
	instancesPrefix := path.Join(consulutil.DeploymentKVPrefix, nodeParams.deploymentID, "topology", "instances", nodeParams.nodeName, instanceName)
	outputs[path.Join(instancesPrefix, "/attributes/volume_id")] = volumeID

	return nil
}

func (g *awsGenerator) getEBSProperties(nodeParams nodeParams, ebs *EBSVolume, size *string, deviceName *string, volumes *string) error {
	// Get string params
	stringParams := []struct {
		pAttr        *string
		propertyName string
		mandatory    bool
	}{
		{volumes, "volume_id", false},
		{&ebs.AvailabilityZone, "availability_zone", true},
		{&ebs.SnapshotID, "snapshot_id", false},
		{&ebs.KMSKeyID, "kms_key_id", false},
		{size, "size", false},
		{deviceName, "device", false},
		{&ebs.Type, "volume_type", false},
		{&ebs.IOPS, "iops", false},
	}

	for _, stringParam := range stringParams {
		val, err := deployments.GetStringNodeProperty(*nodeParams.ctx, nodeParams.deploymentID, nodeParams.nodeName, stringParam.propertyName, stringParam.mandatory)
		if err != nil {
			return err
		}
		*stringParam.pAttr = val
	}

	// Get bool properties
	val, err := deployments.GetBooleanNodeProperty(*nodeParams.ctx, nodeParams.deploymentID, nodeParams.nodeName, "encrypted")
	if err != nil {
		return err
	}
	ebs.Encrypted = val

	// Get tags map
	tagsVal, err := deployments.GetNodePropertyValue(*nodeParams.ctx, nodeParams.deploymentID, nodeParams.nodeName, "tags")
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
