package server

import (
	"os"
	"os/signal"
	"syscall"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
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

	pm := newPluginManager(shutdownCh)

	pm.loadPlugins(configuration)
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
