package commands

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/spf13/cobra"
	"log"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/rest"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"os"
	"os/signal"
	"syscall"
)

func init() {

	cobra.OnInitialize(initConfig)

}

var serverCmd = &cobra.Command{

	Use:   "server",
	Short: "Perform the server command",
	Long:  `Perform the server command`,
	Run: func(cmd *cobra.Command, args []string) {

		configuration := config.Configuration{}
		configuration = getConfig(configuration)

		var ConsulDC string = configuration.CONSUL_DATACENTER
		var ConsulToken string = configuration.CONSUL_TOKEN

		ConsulCustomConfig := api.DefaultConfig()
		ConsulCustomConfig.Datacenter = fmt.Sprintf("%s", ConsulDC)
		ConsulCustomConfig.Token = fmt.Sprintf("%s", ConsulToken)

		client, err := api.NewClient(ConsulCustomConfig)
		if err != nil {
			log.Printf("Can't connect to Consul")

		}
		shutdownCh := make(chan struct{})
		dispatcher := tasks.NewDispatcher(3, shutdownCh, client, configuration)
		go dispatcher.Run()
		httpServer, err := rest.NewServer(client)
		if err != nil {
			log.Print(err)

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
				return
			}
		}
	},
}
