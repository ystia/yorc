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

package google

import (
	"context"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/helper/sizeutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov/terraform/commons"
	"path"
	"strings"
)

func (g *googleGenerator) generatePersistentDisk(ctx context.Context, kv *api.KV,
	cfg config.Configuration, deploymentID, nodeName, instanceName string, instanceID int,
	infrastructure *commons.Infrastructure,
	outputs map[string]string) error {

	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if nodeType != "yorc.nodes.google.PersistentDisk" {
		return errors.Errorf("Unsupported node type for %q: %s", nodeName, nodeType)
	}

	instancesPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID,
		"topology", "instances")
	instancesKey := path.Join(instancesPrefix, nodeName)

	persistentDisk := &PersistentDisk{}
	var size, volumes string
	stringParams := []struct {
		pAttr        *string
		propertyName string
		mandatory    bool
	}{
		{&volumes, "volume_id", false},
		{&persistentDisk.Description, "description", false},
		{&persistentDisk.SourceSnapshot, "snapshot_id", false},
		{&persistentDisk.Type, "type", false},
		{&persistentDisk.Zone, "zone", false},
		{&persistentDisk.SourceSnapshot, "source_snapshot", false},
		{&persistentDisk.SourceImage, "source_image", false},
		{&size, "size", false},
	}

	for _, stringParam := range stringParams {
		if *stringParam.pAttr, err = deployments.GetStringNodeProperty(kv, deploymentID, nodeName,
			stringParam.propertyName, stringParam.mandatory); err != nil {
			return err
		}
	}

	var volumeID string
	if volumes != "" {
		tabVol := strings.Split(volumes, ",")
		if len(tabVol) > instanceID {
			volumeID = strings.TrimSpace(tabVol[instanceID])
		}
	}

	persistentDisk.Labels, err = deployments.GetKeyValuePairsNodeProperty(kv, deploymentID, nodeName, "labels")
	if err != nil {
		return err
	}

	if size != "" {
		// Default size unit is MB
		log.Debugf("Initial size property value (default is MB): %q", size)
		persistentDisk.Size, err = sizeutil.ConvertToGB(size)
		if err != nil {
			return err
		}
		log.Debugf("Computed size (in GB): %d", persistentDisk.Size)
	}

	// Get Encryption key
	rawEncryptKeyValue, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "disk_encryption_key", "raw_key")
	if err != nil {
		return err
	}
	if rawEncryptKeyValue.RawString() != "" {
		hashValue, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "disk_encryption_key", "sha256")
		if err != nil {
			return err
		}
		persistentDisk.DiskEncryptionKey = &EncryptionKey{
			Raw:    rawEncryptKeyValue.RawString(),
			SHA256: hashValue.RawString()}
	}

	name := strings.ToLower(cfg.ResourcesPrefix + nodeName + "-" + instanceName)
	persistentDisk.Name = strings.Replace(name, "_", "-", -1)

	// Add google persistent disk resource if not any volume ID is provided
	if volumeID == "" {
		commons.AddResource(infrastructure, "google_compute_disk", persistentDisk.Name, persistentDisk)
		volumeID = fmt.Sprintf("${google_compute_disk.%s.name}", persistentDisk.Name)
	}

	// Provide Consul Key for attribute volume_id
	consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{}}
	consulKeyVolumeID := commons.ConsulKey{
		Path:  path.Join(instancesKey, instanceName, "/attributes/volume_id"),
		Value: volumeID}

	consulKeyUsers := commons.ConsulKey{
		Path: path.Join(instancesKey, instanceName, "/attributes/users"),
		Value: fmt.Sprintf("${google_compute_disk.%s.users}",
			persistentDisk.Name)}

	consulKeys.Keys = append(consulKeys.Keys, consulKeyVolumeID, consulKeyUsers)
	commons.AddResource(infrastructure, "consul_keys", persistentDisk.Name, &consulKeys)
	return nil
}
