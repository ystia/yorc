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

package server

// Initialization imports
import (
	// Registering AWS delegate executor in the registry
	_ "github.com/ystia/yorc/v4/prov/terraform/aws"
	// Registering Google Cloud delegate executor in the registry
	_ "github.com/ystia/yorc/v4/prov/terraform/google"
	// Registering openstack delegate executor in the registry
	_ "github.com/ystia/yorc/v4/prov/terraform/openstack"
	// Registering ansible operation executor in the registry
	_ "github.com/ystia/yorc/v4/prov/ansible"
	// Registering kubernetes operation executor in the registry
	_ "github.com/ystia/yorc/v4/prov/kubernetes"
	// Registering slurm delegate executor in the registry
	_ "github.com/ystia/yorc/v4/prov/slurm"
	// Registering hosts pool delegate executor in the registry
	_ "github.com/ystia/yorc/v4/prov/hostspool"
	// Registering builtin Tosca definition files
	_ "github.com/ystia/yorc/v4/tosca"
	// Registering builtin HashiCorp Vault Client Builder
	_ "github.com/ystia/yorc/v4/vault/hashivault"
	// Registering builtin activity hooks
	"context"
	"os"
	"path/filepath"

	"github.com/ystia/yorc/v4/deployments/store"
	_ "github.com/ystia/yorc/v4/prov/validation"

	gplugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/plugin"
	"github.com/ystia/yorc/v4/registry"
)

type pluginManager struct {
	pluginClients []*gplugin.Client
}

func newPluginManager() *pluginManager {
	pm := &pluginManager{
		pluginClients: make([]*gplugin.Client, 0),
	}
	return pm
}

func (pm *pluginManager) cleanup() {
	for _, client := range pm.pluginClients {
		client.Kill()
	}
	pm.pluginClients = nil
}

func (pm *pluginManager) loadPlugins(cfg config.Configuration) error {
	pluginsPath := cfg.PluginsDirectory
	if pluginsPath == "" {
		pluginsPath = config.DefaultPluginDir
	}
	pluginPath, err := filepath.Abs(pluginsPath)
	if err != nil {
		return errors.Wrap(err, "Failed to explore plugins directory")
	}
	pluginsFiles, err := filepath.Glob(filepath.Join(pluginPath, "*"))
	if err != nil {
		return errors.Wrap(err, "Failed to explore plugins directory")
	}
	plugins := make([]string, 0)
	for _, pFile := range pluginsFiles {
		fInfo, err := os.Stat(pFile)
		if err != nil {
			return errors.Wrap(err, "Failed to explore plugins directory")
		}
		if !fInfo.IsDir() && fInfo.Mode().Perm()&0111 != 0 {
			plugins = append(plugins, pFile)
		}
	}
	reg := registry.GetRegistry()
	ctx := context.Background()
	for _, pFile := range plugins {
		log.Debugf("Loading plugin %q...", pFile)
		// OK the idea here is to _try_ to load the plugin if we can't we give up with this plugin and try the others
		// There is no reason to stop the server loading if we can't load a plugin.
		pluginID := filepath.Base(pFile)
		client := plugin.NewClient(pFile)
		// Connect via RPC
		rpcClient, err := client.Client()
		if err != nil {
			log.Printf("[Warning] Failed to load %q as a plugin: %v. Skipping it and continue loading plugins.", pFile, err)
			log.Debugf("Error details: %+v", err)
			continue
		}

		// Request the configManager plugin
		raw, err := rpcClient.Dispense(plugin.ConfigManagerPluginName)
		if err != nil {
			log.Printf("[Warning] Failed to load %q as a plugin: %v. Skipping it and continue loading plugins.", pFile, err)
			log.Debugf("Error details: %+v", err)
			client.Kill()
			continue
		}
		cfgManager := raw.(plugin.ConfigManager)
		err = cfgManager.SetupConfig(cfg)
		if err != nil {
			log.Printf("[Warning] Failed to load %q as a plugin: %v. Skipping it and continue loading plugins.", pFile, err)
			log.Debugf("Error details: %+v", err)
			client.Kill()
			continue
		}

		// Request the delegate plugin
		raw, err = rpcClient.Dispense(plugin.DelegatePluginName)
		if err == nil {
			delegateExecutor := raw.(plugin.DelegateExecutor)
			supportedTypes, err := delegateExecutor.GetSupportedTypes()
			if err != nil {
				log.Printf("[Warning] Failed to retrieve delegate executor supported type for plugin %q.", pluginID)
				log.Debugf("%+v", err)
			}
			if len(supportedTypes) > 0 {
				log.Debugf("Registering supported node types %v into registry for plugin %q", supportedTypes, pluginID)
				reg.RegisterDelegates(supportedTypes, delegateExecutor, pluginID)
			}
		} else {
			log.Printf("[Warning] Can't retrieve delegate executor from plugin %q: %v. This is likely due to a outdated plugin.", pluginID, err)
			log.Debugf("%+v", err)
		}

		// Request the operation plugin
		raw, err = rpcClient.Dispense(plugin.OperationPluginName)
		if err == nil {
			operationExecutor := raw.(plugin.OperationExecutor)
			supportedArtTypes, err := operationExecutor.GetSupportedArtifactTypes()
			if err != nil {
				log.Printf("[Warning] Failed to retrieve operation executor supported implementation artifacts for plugin %q.", pluginID)
				log.Debugf("%+v", err)
			}
			if len(supportedArtTypes) > 0 {
				log.Debugf("Registering supported implementation artifact types %v into registry for plugin %q", supportedArtTypes, pluginID)
				reg.RegisterOperationExecutor(supportedArtTypes, operationExecutor, pluginID)
			}
		} else {
			log.Printf("[Warning] Can't retrieve operation executor from plugin %q: %v. This is likely due to a outdated plugin.", pluginID, err)
			log.Debugf("%+v", err)
		}

		// Request the action plugin
		raw, err = rpcClient.Dispense(plugin.ActionPluginName)
		if err == nil {
			actionOperator := raw.(plugin.ActionOperator)
			actionTypes, err := actionOperator.GetActionTypes()
			if err != nil {
				log.Printf("[Warning] Failed to retrieve action types for plugin %q.", pluginID)
				log.Debugf("%+v", err)
			}
			if len(actionTypes) > 0 {
				log.Debugf("Registering action types %v into registry for plugin %q", actionTypes, pluginID)
				reg.RegisterActionOperator(actionTypes, actionOperator, pluginID)
			}
		} else {
			log.Printf("[Warning] Can't retrieve action operator from plugin %q: %v. This is likely due to a outdated plugin.", pluginID, err)
			log.Debugf("%+v", err)
		}

		// Request the definitions plugin
		raw, err = rpcClient.Dispense(plugin.DefinitionsPluginName)
		if err == nil {
			definitionPlugin := raw.(plugin.Definitions)
			definitions, err := definitionPlugin.GetDefinitions()
			if err != nil {
				log.Printf("[Warning] Failed to retrieve TOSCA definitions for plugin %q.", pluginID)
				log.Debugf("%+v", err)
			}
			if len(definitions) > 0 {
				for defName, defContent := range definitions {
					log.Debugf("Registering TOSCA definition %q into registry for plugin %q", defName, pluginID)
					store.CommonDefinition(ctx, defName, pluginID, defContent)
				}
			}
		} else {
			log.Printf("[Warning] Can't retrieve TOSCA definitions from plugin %q: %v. This is likely due to a outdated plugin.", pluginID, err)
			log.Debugf("%+v", err)
		}

		// Request the infra usage collector plugin
		raw, err = rpcClient.Dispense(plugin.InfraUsageCollectorPluginName)
		if err == nil {
			infraUsageCollectorPlugin := raw.(plugin.InfraUsageCollector)
			infras, err := infraUsageCollectorPlugin.GetSupportedInfras()
			if err != nil {
				log.Printf("[Warning] Failed to retrieve supported infrastructure for plugin %q.", pluginID)
				log.Debugf("%+v", err)
			}
			if len(infras) > 0 {
				for _, infra := range infras {
					log.Debugf("Registering infrastructure usage collector %q into registry for plugin %q", infra, pluginID)
					reg.RegisterInfraUsageCollector(infra, infraUsageCollectorPlugin, pluginID)
				}
			}
		} else {
			log.Printf("[Warning] Can't get collector supported infra from plugin %q: %v. This is likely due to a outdated plugin.", pluginID, err)
			log.Debugf("%+v", err)
		}

		pm.pluginClients = append(pm.pluginClients, client)

		log.Printf("Plugin %q successfully loaded", pluginID)

	}

	return nil
}

func (pm *pluginManager) pingPlugins() error {
	var err error
	for _, pluginClient := range pm.pluginClients {
		var c gplugin.ClientProtocol
		c, err = pluginClient.Client()
		if err != nil {
			break
		}

		err = c.Ping()
		if err != nil {
			break
		}
	}

	return err
}
