package server

// Initialization imports
import (
	// Registering openstack delegate executor in the registry
	_ "novaforge.bull.com/starlings-janus/janus/prov/terraform/openstack"
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
		pluginID := filepath.Base(pFile)
		client := plugin.NewClient(pFile)
		pm.pluginClients = append(pm.pluginClients, client)
		// Connect via RPC
		rpcClient, err := client.Client()
		if err != nil {
			return errors.Wrapf(err, "Failed to load plugin %q", pFile)
		}

		// Request the delegate plugin
		raw, err := rpcClient.Dispense(plugin.DelegatePluginName)
		if err != nil {
			return errors.Wrapf(err, "Failed to load plugin %q", pFile)
		}

		delegateExecutor := raw.(plugin.DelegateExecutor)
		supportedTypes, err := delegateExecutor.GetSupportedTypes()
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve supported type for delegate")
		}
		if len(supportedTypes) > 0 {
			log.Debugf("Registering supported node types %v into registry for plugin %q", supportedTypes, pluginID)
			reg.RegisterDelegates(supportedTypes, delegateExecutor, pluginID)
		}
		// Request the delegate plugin
		raw, err = rpcClient.Dispense(plugin.DefinitionsPluginName)
		if err != nil {
			return errors.Wrapf(err, "Failed to load plugin %q", pFile)
		}
		definitionPlugin := raw.(plugin.Definitions)
		definitions, err := definitionPlugin.GetDefinitions()
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve plugin specific TOSCA definitions")
		}
		for defName, defContent := range definitions {
			log.Debugf("Registering TOSCA definition %q into registry for plugin %q", defName, pluginID)
			reg.AddToscaDefinition(defName, pluginID, defContent)
		}

	}

	return nil
}
