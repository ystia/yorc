package server

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/rest"
	"novaforge.bull.com/starlings-janus/janus/tasks"
)

func RunServer(configuration config.Configuration, shutdownCh chan struct{}) error {
	consulDC := configuration.ConsulDatacenter
	consulToken := configuration.ConsulToken

	consulCustomConfig := api.DefaultConfig()
	if configuration.ConsulAddress != "" {
		consulCustomConfig.Address = configuration.ConsulAddress
	}
	if consulDC != "" {
		consulCustomConfig.Datacenter = fmt.Sprintf("%s", consulDC)
	}
	if consulToken != "" {
		consulCustomConfig.Token = fmt.Sprintf("%s", consulToken)
	}
	client, err := api.NewClient(consulCustomConfig)
	if err != nil {
		log.Printf("Can't connect to Consul")
		return err
	}

	maxConsulPubRoutines := configuration.ConsulPubMaxRoutines
	if maxConsulPubRoutines <= 0 {
		maxConsulPubRoutines = config.DefaultConsulPubMaxRoutines
	}

	consulutil.InitConsulPublisher(maxConsulPubRoutines, client.KV())

	dispatcher := tasks.NewDispatcher(3, shutdownCh, client, configuration)
	go dispatcher.Run()
	httpServer, err := rest.NewServer(configuration, client, shutdownCh)
	if err != nil {
		close(shutdownCh)
		return err
	}
	defer httpServer.Shutdown()
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
