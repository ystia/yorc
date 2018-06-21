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
	"path"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"strconv"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov/terraform/commons"
)

func (g *googleGenerator) generateComputeInstance(ctx context.Context, kv *api.KV,
	cfg config.Configuration, deploymentID, nodeName, instanceName string,
	infrastructure *commons.Infrastructure,
	outputs map[string]string) error {

	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if nodeType != "yorc.nodes.google.Compute" {
		return errors.Errorf("Unsupported node type for %q: %s", nodeName, nodeType)
	}

	instancesPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID,
		"topology", "instances")
	instancesKey := path.Join(instancesPrefix, nodeName)

	instance := ComputeInstance{}

	// Must be a match of regex '(?:[a-z](?:[-a-z0-9]{0,61}[a-z0-9])?)'
	instance.Name = strings.ToLower(cfg.ResourcesPrefix + nodeName + "-" + instanceName)

	// Getting string parameters
	var imageProject, imageFamily, image, serviceAccount string

	stringParams := []struct {
		pAttr        *string
		propertyName string
		mandatory    bool
	}{
		{&instance.MachineType, "machine_type", true},
		{&instance.Zone, "zone", true},
		{&imageProject, "image_project", false},
		{&imageFamily, "image_family", false},
		{&image, "image", false},
		{&instance.Description, "description", false},
		{&serviceAccount, "service_acount", false},
	}

	for _, stringParam := range stringParams {
		if *stringParam.pAttr, err = getStringProperty(kv, deploymentID, nodeName,
			stringParam.propertyName, stringParam.mandatory); err != nil {
			return err
		}
	}

	// Define the boot disk from image settings
	var bootImage string
	if imageProject != "" {
		bootImage = imageProject
		if image != "" {
			bootImage = bootImage + "/" + image
		} else if imageFamily != "" {
			bootImage = bootImage + "/" + imageFamily
		} else {
			// Unexpected image project without a family or image
			return errors.Errorf("Exepected an image or family for image project %s on %s", imageProject, nodeName)
		}
	} else if image != "" {
		bootImage = image
	} else {
		bootImage = imageFamily
	}

	var bootDisk Disk
	if bootImage != "" {
		bootDisk.Image = bootImage
	}
	instance.Disks = []Disk{bootDisk}

	// Get boolean parameters

	if instance.NoAddress, err = getBooleanProperty(kv, deploymentID, nodeName, "no_address"); err != nil {
		return err
	}

	if instance.Preemptible, err = getBooleanProperty(kv, deploymentID, nodeName, "preemptible"); err != nil {
		return err
	}

	// Network interface definition
	networkInterface := NetworkInterface{Network: "default"}
	// Define an external access if there will be an external IP address
	if !instance.NoAddress {
		// keeping all default values
		accessConfig := AccessConfig{}
		networkInterface.AccessConfigs = []AccessConfig{accessConfig}
	}
	instance.NetworkInterfaces = []NetworkInterface{networkInterface}

	// Get list of strings parameters
	var scopes []string
	if scopes, err = getStringArray(kv, deploymentID, nodeName, "scopes"); err != nil {
		return err
	}

	if serviceAccount != "" || len(scopes) > 0 {
		// Adding a service account section, where scopes can't be empty
		if len(scopes) == 0 {
			scopes = []string{"cloud-platform"}
		}
		configuredAccount := ServiceAccount{serviceAccount, scopes}
		instance.ServiceAccounts = []ServiceAccount{configuredAccount}
	}

	if instance.Tags, err = getStringArray(kv, deploymentID, nodeName, "tags"); err != nil {
		return err
	}

	// Get list of key/value pairs parameters

	if instance.Labels, err = getKeyValuePairs(kv, deploymentID, nodeName, "labels"); err != nil {
		return err
	}

	if instance.Metadata, err = getKeyValuePairs(kv, deploymentID, nodeName, "metadata"); err != nil {
		return err
	}

	// user is mandatory
	var user string
	if _, user, err = deployments.GetCapabilityProperty(kv, deploymentID, nodeName,
		"endpoint", "credentials", "user"); err != nil {
		return err
	} else if user == "" {
		return errors.Errorf("Missing mandatory parameter 'user' node type for %s", nodeName)
	}

	var privateKeyFilePath string
	if _, privateKeyFilePath, err = deployments.GetCapabilityProperty(kv, deploymentID,
		nodeName, "endpoint", "credentials", "keys", "0"); err != nil {
		return err
	} else if privateKeyFilePath == "" {
		// Using default value
		privateKeyFilePath = commons.DefaultSSHPrivateKeyFilePath
		log.Printf("No private key defined for user %s, using default %s", user, privateKeyFilePath)
	}

	// Add the compute instance
	commons.AddResource(infrastructure, "google_compute_instance", instance.Name, &instance)

	// Provide Consul Keys
	consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{}}

	// Define the private IP address using the value exported by Terraform
	privateIP := fmt.Sprintf("${google_compute_instance.%s.network_interface.0.address}",
		instance.Name)

	consulKeyPrivateAddr := commons.ConsulKey{
		Path:  path.Join(instancesKey, instanceName, "/attributes/private_address"),
		Value: privateIP}

	consulKeys.Keys = append(consulKeys.Keys, consulKeyPrivateAddr)

	// Define the public IP using the value exported by Terraform
	// except if it was specified the instance shouldn't have a public address
	var accessIP string
	if instance.NoAddress {
		accessIP = privateIP
	} else {
		accessIP = fmt.Sprintf("${google_compute_instance.%s.network_interface.0.access_config.0.assigned_nat_ip}",
			instance.Name)
		consulKeyPublicAddr := commons.ConsulKey{
			Path:  path.Join(instancesKey, instanceName, "/attributes/public_address"),
			Value: accessIP}
		// For backward compatibility...
		consulKeyPublicIPAddr := commons.ConsulKey{
			Path:  path.Join(instancesKey, instanceName, "/attributes/public_ip_address"),
			Value: accessIP}

		consulKeys.Keys = append(consulKeys.Keys, consulKeyPublicAddr,
			consulKeyPublicIPAddr)
	}

	// IP Address capability
	capabilityIPAddr := commons.ConsulKey{
		Path:  path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/ip_address"),
		Value: accessIP}
	// Default TOSCA Attributes
	consulKeyIPAddr := commons.ConsulKey{
		Path:  path.Join(instancesKey, instanceName, "/attributes/ip_address"),
		Value: accessIP}

	consulKeys.Keys = append(consulKeys.Keys, consulKeyIPAddr, capabilityIPAddr)

	commons.AddResource(infrastructure, "consul_keys", instance.Name, &consulKeys)

	// Check the connection in order to be sure that ansible will be able to log on the instance
	nullResource := commons.Resource{}
	re := commons.RemoteExec{Inline: []string{`echo "connected"`},
		Connection: &commons.Connection{User: user, Host: accessIP,
			PrivateKey: `${file("` + privateKeyFilePath + `")}`}}
	nullResource.Provisioners = make([]map[string]interface{}, 0)
	provMap := make(map[string]interface{})
	provMap["remote-exec"] = re
	nullResource.Provisioners = append(nullResource.Provisioners, provMap)

	commons.AddResource(infrastructure, "null_resource", instance.Name+"-ConnectionCheck", &nullResource)

	return nil
}

func getStringProperty(kv *api.KV, deploymentID, nodeName, propertyName string, mandatory bool) (string, error) {

	var result string
	var err error
	if _, result, err = deployments.GetNodeProperty(kv, deploymentID,
		nodeName, propertyName); err != nil {
		return result, err
	}

	if result == "" && mandatory {
		return result, errors.Errorf("Missing value for mandatory parameter %s of %s",
			propertyName, nodeName)
	}

	return result, nil
}

func getBooleanProperty(kv *api.KV, deploymentID, nodeName, propertyName string) (bool, error) {

	var result bool
	var strValue string
	var err error
	if _, strValue, err = deployments.GetNodeProperty(kv, deploymentID,
		nodeName, propertyName); err != nil {
		return result, err
	}

	if strValue == "" {
		result = false
	} else {
		result, err = strconv.ParseBool(strValue)
		if err != nil {
			log.Printf("Unexpected value for %s %s: '%s', considering it is set to 'false'",
				nodeName, propertyName, strValue)
			result = false
		}
	}

	return result, nil
}

func getStringArray(kv *api.KV, deploymentID, nodeName, propertyName string) ([]string, error) {

	var result []string
	var strValue string
	var found bool
	var err error
	if found, strValue, err = deployments.GetNodeProperty(kv, deploymentID,
		nodeName, propertyName); err != nil {
		return nil, err
	}

	if found && strValue != "" {
		values := strings.Split(strValue, ",")
		for _, val := range values {
			result = append(result, strings.TrimSpace(val))
		}
	}

	return result, nil
}

func getKeyValuePairs(kv *api.KV, deploymentID, nodeName, propertyName string) (map[string]string, error) {

	var result map[string]string
	var strValue string
	var found bool
	var err error
	if found, strValue, err = deployments.GetNodeProperty(kv, deploymentID,
		nodeName, propertyName); err != nil {
		return nil, err
	}

	if found && strValue != "" {
		result = make(map[string]string)
		values := strings.Split(strValue, ",")
		for _, val := range values {
			keyValuePair := strings.Split(val, "=")
			if len(keyValuePair) != 2 {

				return result, errors.Errorf("Expected KEY=VALUE format, got %s for property %s on %s",
					val, propertyName, nodeName)
			}
			result[strings.TrimSpace(keyValuePair[0])] =
				strings.TrimSpace(keyValuePair[1])
		}
	}
	return result, nil
}
