package server

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"path/filepath"

	"github.com/hashicorp/consul/api"
	gplugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/plugin"
	"novaforge.bull.com/starlings-janus/janus/prov"
	"novaforge.bull.com/starlings-janus/janus/rest"
	"novaforge.bull.com/starlings-janus/janus/tasks/workflow"
)

// RunServer starts the Janus server
func RunServer(configuration config.Configuration, shutdownCh chan struct{}) error {
	client, err := configuration.GetConsulClient()
	if err != nil {
		log.Printf("Can't connect to Consul")
		return err
	}

	maxConsulPubRoutines := configuration.ConsulPubMaxRoutines
	if maxConsulPubRoutines <= 0 {
		maxConsulPubRoutines = config.DefaultConsulPubMaxRoutines
	}

	consulutil.InitConsulPublisher(maxConsulPubRoutines, client.KV())

	dispatcher := workflow.NewDispatcher(configuration.WorkersNumber, shutdownCh, client, configuration)
	go dispatcher.Run()
	httpServer, err := rest.NewServer(configuration, client, shutdownCh)
	if err != nil {
		close(shutdownCh)
		return err
	}
	defer httpServer.Shutdown()
	loadPlugins(configuration, client)
	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	for {
		var sig os.Signal
		shutdownChClosed := false
		select {
		case s := <-signalCh:
			sig = s
		case <-shutdownCh:
			sig = os.Interrupt
			shutdownChClosed = true
		}

		// Check if this is a SIGHUP
		if sig == syscall.SIGHUP {
			// TODO reload
		} else {
			if !shutdownChClosed {
				close(shutdownCh)
			}
			return nil
		}
	}
}

func loadPlugins(cfg config.Configuration, consulClient *api.Client) error {
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

	for _, pFile := range plugins {
		log.Debugf("Loading plugin %q...", pFile)
		client := gplugin.NewClient(&gplugin.ClientConfig{
			HandshakeConfig: plugin.HandshakeConfig,
			Plugins: map[string]gplugin.Plugin{
				"delegate": &plugin.DelegatePlugin{},
			},
			Cmd: exec.Command(pFile),
		})
		defer client.Kill()
		// Connect via RPC
		rpcClient, err := client.Client()
		if err != nil {
			log.Fatal(err)
		}

		// Request the plugin
		raw, err := rpcClient.Dispense("delegate")
		if err != nil {
			log.Fatal(err)
		}

		// We should have a Greeter now! This feels like a normal interface
		// implementation but is in fact over an RPC connection.
		delegate := raw.(prov.DelegateExecutor)
		err = delegate.ExecDelegate(context.Background(), cfg, "1", "2", "3", "4")
		if err != nil {
			log.Fatal(err)
		}
		log.Println("main result")
	}

	return nil
}
