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
	"path"
	"strings"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/helper/sizeutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov/terraform/commons"
)

func (g *awsGenerator) generateEBS(instanceName string, instanceID int, outputs map[string]string) error {
	err := verifyThatNodeIsTypeOf(g, "yorc.nodes.aws.EBSVolume")
	if err != nil {
		return err
	}

	ebs := &EBSVolume{}

	// Get string params
	var size, deviceName string
	g.getEBSProperties(ebs, &size, &deviceName)

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
	name := strings.ToLower(g.nodeName + "-" + instanceName)
	commons.AddResource(g.infrastructure, "aws_ebs_volume", name, ebs)

	// Terraform Outputs
	volumeID := g.nodeName + "-" + instanceName + "-id"           // ex : "BlockStorage-0-id"
	volumeIDValue := fmt.Sprintf("${aws_ebs_volume.%s.id}", name) // ex : ${aws_ebs_volume.blockstorage-0.id}
	volumeARN := g.nodeName + "-" + instanceName + "-arn"
	volumeARNValue := fmt.Sprintf("${aws_ebs_volume.%s.arn}", name)
	commons.AddOutput(g.infrastructure, volumeID, &commons.Output{Value: volumeIDValue})
	commons.AddOutput(g.infrastructure, volumeARN, &commons.Output{Value: volumeARNValue})

	// Yorc attributes
	instancesPrefix := path.Join(consulutil.DeploymentKVPrefix, g.deploymentID, "topology", "instances")
	instancesKey := path.Join(instancesPrefix, g.nodeName)
	outputs[path.Join(instancesKey, instanceName, "/attributes/volume_id")] = volumeID
	// outputs[path.Join(instancesKey, instanceName, "/attributes/device_name")] = deviceName
	// TODO : outputs ARN ?

	return nil
}

func (g *awsGenerator) getEBSProperties(ebs *EBSVolume, size *string, deviceName *string) error {
	// Get string params
	stringParams := []struct {
		pAttr        *string
		propertyName string
		mandatory    bool
	}{
		{&ebs.AvailabilityZone, "availability_zone", true},
		{&ebs.SnapshotID, "snapshot_id", false},
		{&ebs.KMSKeyID, "kms_key_id", false},
		{size, "size", false},
		{deviceName, "device", false},
	}

	for _, stringParam := range stringParams {
		val, err := deployments.GetStringNodeProperty(*g.ctx, g.deploymentID, g.nodeName, stringParam.propertyName, stringParam.mandatory)
		if err != nil {
			return err
		}
		*stringParam.pAttr = val
	}

	// Get bool properties
	val, err := deployments.GetBooleanNodeProperty(*g.ctx, g.deploymentID, g.nodeName, "encrypted")
	if err != nil {
		return err
	}
	ebs.Encrypted = val

	return nil
}
