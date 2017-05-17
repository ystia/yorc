package server

import (
	"os"
	"os/signal"
	"syscall"

	"sync"

	"time"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/rest"
	"novaforge.bull.com/starlings-janus/janus/tasks/workflow"
)

// RunServer starts the Janus server
func RunServer(configuration config.Configuration, shutdownCh chan struct{}) error {
	var wg sync.WaitGroup
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

	dispatcher := workflow.NewDispatcher(configuration, shutdownCh, client, &wg)
	go dispatcher.Run()
	httpServer, err := rest.NewServer(configuration, client, shutdownCh)
	if err != nil {
		close(shutdownCh)
		return err
	}
	defer httpServer.Shutdown()

	pm := newPluginManager(shutdownCh, &wg)

	err = pm.loadPlugins(configuration)
	if err != nil {
		close(shutdownCh)
		return err
	}
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
			gracefulTimeout := configuration.ServerGracefulShutdownTimeout
			if gracefulTimeout == 0 {
				gracefulTimeout = config.DefaultServerGracefulShutdownTimeout
			}
			log.Printf("Waiting at least %v for a graceful server shutdown. Send another termination signal to exit immediately.", gracefulTimeout)
			gracefulCh := make(chan struct{})
			go func() {
				wg.Wait()
				close(gracefulCh)
			}()
			select {
			// In case of double Ctrl+C then exit immediately
			case <-signalCh:
			case <-gracefulCh:
			case <-time.After(gracefulTimeout):

			}
			return nil
		}
	}
}
