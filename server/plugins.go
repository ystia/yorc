package server

// Initialization imports
import (
	// Registering AWS delegate executor in the registry
	_ "novaforge.bull.com/starlings-janus/janus/prov/terraform/aws"
	// Registering openstack delegate executor in the registry
	_ "novaforge.bull.com/starlings-janus/janus/prov/terraform/openstack"
	// Registering ansible operation executor in the registry
	_ "novaforge.bull.com/starlings-janus/janus/prov/ansible"
	// Registering kubernetes operation executor in the registry
	_ "novaforge.bull.com/starlings-janus/janus/prov/kubernetes"
	// Registering builtin Tosca definition files
	_ "novaforge.bull.com/starlings-janus/janus/tosca"
)

import (
	"os"
	"path/filepath"

	gplugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/plugin"
	"novaforge.bull.com/starlings-janus/janus/registry"
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
					reg.AddToscaDefinition(defName, pluginID, defContent)
				}
			}
		} else {
			log.Printf("[Warning] Can't retrieve TOSCA definitions from plugin %q: %v. This is likely due to a outdated plugin.", pluginID, err)
			log.Debugf("%+v", err)
		}

		pm.pluginClients = append(pm.pluginClients, client)

		log.Printf("Plugin %q successfully loaded", pluginID)

	}

	return nil
}
