package openstack

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

type osGenerator struct {
}

func (g *osGenerator) getStringFormConsul(kv *api.KV, baseURL, property string) (string, error) {
	getResult, _, err := kv.Get(baseURL+"/"+property, nil)
	if err != nil {
		return "", errors.Errorf("Can't get property %s for node %s: %v", property, baseURL, err)
	}
	if getResult == nil {
		log.Debugf("Can't get property %s for node %s (not found)", property, baseURL)
		return "", nil
	}
	return string(getResult.Value), nil
}

func (g *osGenerator) GenerateTerraformInfraForNode(ctx context.Context, cfg config.Configuration, deploymentID, nodeName string) (bool, map[string]string, error) {
	log.Debugf("Generating infrastructure for deployment with id %s", deploymentID)
	cClient, err := cfg.GetConsulClient()
	if err != nil {
		return false, nil, err
	}
	kv := cClient.KV()
	nodeKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	instancesKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName)
	terraformStateKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "terraform-state", nodeName)

	infrastructure := commons.Infrastructure{}

	consulAddress := "127.0.0.1:8500"
	if cfg.ConsulAddress != "" {
		consulAddress = cfg.ConsulAddress
	}
	consulScheme := "http"
	if cfg.ConsulSSL {
		consulScheme = "https"
	}
	consulCA := ""
	if cfg.ConsulCA != "" {
		consulCA = cfg.ConsulCA
	}
	consulKey := ""
	if cfg.ConsulKey != "" {
		consulKey = cfg.ConsulKey
	}
	consulCert := ""
	if cfg.ConsulCert != "" {
		consulCert = cfg.ConsulCert
	}

	log.Debugf("Generating infrastructure for deployment with node %s", nodeName)
	log.Debugf(">>> In GenerateTerraformInfraForNode use consulAddress %s", consulAddress)
	log.Debugf(">>> In GenerateTerraformInfraForNode use consulScheme %s", consulScheme)
	log.Debugf(">>> In GenerateTerraformInfraForNode use consulCA %s", consulCA)
	log.Debugf(">>> In GenerateTerraformInfraForNode use consulKey %s", consulKey)
	log.Debugf(">>> In GenerateTerraformInfraForNode use consulCert %s", consulCert)

	// Remote Configuration for Terraform State to store it in the Consul KV store
	infrastructure.Terraform = map[string]interface{}{
		"backend": map[string]interface{}{
			"consul": map[string]interface{}{
				"path":    terraformStateKey,
				"address": consulAddress,
				"scheme":  consulScheme,
			},
		},
	}

	// Management of variables for Terraform
	infrastructure.Provider = map[string]interface{}{
		"openstack": map[string]interface{}{
			"user_name":   cfg.Infrastructures[infrastructureName].GetString("user_name"),
			"tenant_name": cfg.Infrastructures[infrastructureName].GetString("tenant_name"),
			"password":    cfg.Infrastructures[infrastructureName].GetString("password"),
			"auth_url":    cfg.Infrastructures[infrastructureName].GetString("auth_url"),
			"insecure":    cfg.Infrastructures[infrastructureName].GetString("insecure"),
			"cacert_file": cfg.Infrastructures[infrastructureName].GetString("cacert_file"),
			"cert":        cfg.Infrastructures[infrastructureName].GetString("cert"),
			"key":         cfg.Infrastructures[infrastructureName].GetString("key"),
		},
		"consul": map[string]interface{}{
			"address":   consulAddress,
			"scheme":    consulScheme,
			"ca_file":   consulCA,
			"cert_file": consulCert,
			"key_file":  consulKey,
		},
	}

	log.Debugf("inspecting node %s", nodeKey)
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return false, nil, err
	}
	outputs := make(map[string]string)
	var instances []string
	switch nodeType {
	case "janus.nodes.openstack.Compute":
		instances, err = deployments.GetNodeInstancesIds(kv, deploymentID, nodeName)
		if err != nil {
			return false, nil, err
		}

		for _, instanceName := range instances {
			var instanceState tosca.NodeState
			instanceState, err = deployments.GetInstanceState(kv, deploymentID, nodeName, instanceName)
			if err != nil {
				return false, nil, err
			}
			if instanceState == tosca.NodeStateDeleting || instanceState == tosca.NodeStateDeleted {
				// Do not generate something for this node instance (will be deleted if exists)
				continue
			}
			err = g.generateOSInstance(ctx, kv, cfg, deploymentID, nodeName, instanceName, &infrastructure, outputs)
			if err != nil {
				return false, nil, err
			}
		}

	case "janus.nodes.openstack.BlockStorage":
		instances, err = deployments.GetNodeInstancesIds(kv, deploymentID, nodeName)
		if err != nil {
			return false, nil, err
		}

		var bsIds []string
		var volumeID string
		if volumeID, err = g.getStringFormConsul(kv, nodeKey, "properties/volume_id"); err != nil {
			return false, nil, err
		} else if volumeID != "" {
			log.Debugf("Reusing existing volume with id %q for node %q", volumeID, nodeName)
			bsIds = strings.Split(volumeID, ",")
		}

		for instNb, instanceName := range instances {
			var instanceState tosca.NodeState
			instanceState, err = deployments.GetInstanceState(kv, deploymentID, nodeName, instanceName)
			if err != nil {
				return false, nil, err
			}
			if instanceState == tosca.NodeStateDeleting || instanceState == tosca.NodeStateDeleted {
				// Do not generate something for this node instance (will be deleted if exists)
				continue
			}
			var bsVolume BlockStorageVolume
			bsVolume, err = g.generateOSBSVolume(kv, cfg, nodeKey, instanceName)
			if err != nil {
				return false, nil, err
			}

			if len(bsIds)-1 < instNb {
				commons.AddResource(&infrastructure, "openstack_blockstorage_volume_v1", bsVolume.Name, &bsVolume)
				consulKey := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/volume_id"), Value: fmt.Sprintf("${openstack_blockstorage_volume_v1.%s.id}", bsVolume.Name)}
				consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
				commons.AddResource(&infrastructure, "consul_keys", bsVolume.Name, &consulKeys)
			} else {
				name := cfg.ResourcesPrefix + nodeName + "-" + instanceName
				consulKey := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/properties/volume_id"), Value: strings.TrimSpace(bsIds[instNb])}
				consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
				commons.AddResource(&infrastructure, "consul_keys", name, &consulKeys)
			}

		}

	case "janus.nodes.openstack.FloatingIP":
		instances, err = deployments.GetNodeInstancesIds(kv, deploymentID, nodeName)
		if err != nil {
			return false, nil, err
		}

		for _, instanceName := range instances {
			var instanceState tosca.NodeState
			instanceState, err = deployments.GetInstanceState(kv, deploymentID, nodeName, instanceName)
			if err != nil {
				return false, nil, err
			}
			if instanceState == tosca.NodeStateDeleting || instanceState == tosca.NodeStateDeleted {
				// Do not generate something for this node instance (will be deleted if exists)
				continue
			}
			var ip IP
			ip, err = g.generateFloatingIP(kv, nodeKey, instanceName)

			if err != nil {
				return false, nil, err
			}

			var consulKey commons.ConsulKey
			if !ip.IsIP {
				floatingIP := FloatingIP{Pool: ip.Pool}
				commons.AddResource(&infrastructure, "openstack_compute_floatingip_v2", ip.Name, &floatingIP)
				consulKey = commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/floating_ip_address"), Value: fmt.Sprintf("${openstack_compute_floatingip_v2.%s.address}", ip.Name)}
			} else {
				ips := strings.Split(ip.Pool, ",")
				// TODO we should change this. instance name should not be considered as an int
				var instName int
				instName, err = strconv.Atoi(instanceName)
				if err != nil {
					return false, nil, err
				}
				if (len(ips) - 1) < instName {
					var networkName string
					networkName, err = g.getStringFormConsul(kv, nodeKey, "properties/floating_network_name")
					if err != nil {
						return false, nil, err
					} else if networkName == "" {
						return false, nil, errors.Errorf("You need to provide enough IP address or a Pool to generate missing IP address")
					}

					floatingIP := FloatingIP{Pool: networkName}
					commons.AddResource(&infrastructure, "openstack_compute_floatingip_v2", ip.Name, &floatingIP)
					consulKey = commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/floating_ip_address"), Value: fmt.Sprintf("${openstack_compute_floatingip_v2.%s.address}", ip.Name)}

				} else {
					// TODO we should change this. instance name should not be considered as an int
					instName, err = strconv.Atoi(instanceName)
					if err != nil {
						return false, nil, err
					}
					consulKey = commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/floating_ip_address"), Value: ips[instName]}
				}

			}
			consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
			commons.AddResource(&infrastructure, "consul_keys", ip.Name, &consulKeys)
		}

	case "janus.nodes.openstack.Network":
		var networkID string
		networkID, err = g.getStringFormConsul(kv, nodeKey, "properties/network_id")
		if err != nil {
			return false, nil, err
		} else if networkID != "" {
			log.Debugf("Reusing existing volume with id %q for node %q", networkID, nodeName)
			return false, nil, nil
		}
		var network Network
		network, err = g.generateNetwork(kv, cfg, nodeKey, deploymentID)

		if err != nil {
			return false, nil, err
		}
		var subnet Subnet
		subnet, err = g.generateSubnet(kv, cfg, nodeKey, deploymentID, nodeName)

		if err != nil {
			return false, nil, err
		}

		commons.AddResource(&infrastructure, "openstack_networking_network_v2", nodeName, &network)
		commons.AddResource(&infrastructure, "openstack_networking_subnet_v2", nodeName+"_subnet", &subnet)
		consulKey := commons.ConsulKey{Path: nodeKey + "/attributes/network_id", Value: fmt.Sprintf("${openstack_networking_network_v2.%s.id}", nodeName)}
		consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
		commons.AddResource(&infrastructure, "consul_keys", nodeName, &consulKeys)

	default:
		return false, nil, errors.Errorf("Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeName, deploymentID)
	}

	jsonInfra, err := json.MarshalIndent(infrastructure, "", "  ")
	if err != nil {
		return false, nil, errors.Wrap(err, "Failed to generate JSON of terraform Infrastructure description")
	}
	infraPath := filepath.Join(cfg.WorkingDirectory, "deployments", deploymentID, "infra", nodeName)
	if err = os.MkdirAll(infraPath, 0775); err != nil {
		return false, nil, errors.Wrapf(err, "Failed to create infrastructure working directory %q", infraPath)
	}

	if err = ioutil.WriteFile(filepath.Join(infraPath, "infra.tf.json"), jsonInfra, 0664); err != nil {
		return false, nil, errors.Wrapf(err, "Failed to write file %q", filepath.Join(infraPath, "infra.tf.json"))
	}

	log.Debugf("Infrastructure generated for deployment with id %s", deploymentID)
	return true, outputs, nil
}
