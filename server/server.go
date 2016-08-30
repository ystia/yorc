package server

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/rest"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"os"
	"os/signal"
	"syscall"
)

func RunServer(configuration config.Configuration, shutdownCh chan struct{}) error {
	var ConsulDC string = configuration.CONSUL_DATACENTER
	var ConsulToken string = configuration.CONSUL_TOKEN

	ConsulCustomConfig := api.DefaultConfig()
	if configuration.CONSUL_ADDRESS != "" {
		ConsulCustomConfig.Address = configuration.CONSUL_ADDRESS
	}
	if ConsulDC != "" {
		ConsulCustomConfig.Datacenter = fmt.Sprintf("%s", ConsulDC)
	}
	if ConsulToken != "" {
		ConsulCustomConfig.Token = fmt.Sprintf("%s", ConsulToken)
	}
	client, err := api.NewClient(ConsulCustomConfig)
	if err != nil {
		log.Printf("Can't connect to Consul")
		return err
	}
	dispatcher := tasks.NewDispatcher(3, shutdownCh, client, configuration)
	go dispatcher.Run()
	httpServer, err := rest.NewServer(client)
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
