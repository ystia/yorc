package commands


import (
    "fmt"
	"github.com/hashicorp/consul/api"
	"log"
	"novaforge.bull.com/starlings-janus/janus/rest"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"os"
	"os/signal"
	"syscall"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)


var serverCmd = &cobra.Command{

			Use: "server",
			Short: "Perform the server command",
			Long: `Perform the server command`,
			Run: func(cmd *cobra.Command, args []string) {	
				
				c:= new(Command)
				/*
				if viper.IsSet("cl_auth-url") {
					AUTH_URL := viper.GetString("cl_auth-url") // case insensitive Setting & Getting
				} else { 
					if viper.IsSet("auth-url") {
						AUTH_URL := viper.GetString("auth-url") }
				}
				*/
				//WARNING: If we define default value for the flags, the following settings are useless

				if viper.IsSet("cl_consul_datacenter") {
					CONSUL_DATACENTER := viper.GetString("cl_consul_datacenter")
				} else {
						if viper.IsSet("consul_datacenter") {
							CONSUL_DATACENTER := viper.GetString("consul_datacenter") }
				}

				if viper.IsSet("cl_consul_token") {
					CONSUL_TOKEN := viper.GetString("cl_consul_token")
				} else {
						if viper.IsSet("consul_token") {
							CONSUL_TOKEN := viper.GetString("consul_token") }
				}

				//Custom configuration for Consul
				lenConsulDC := len(CONSUL_DATACENTER)
				lenConsulToken := len(CONSUL_TOKEN)
				if (lenConsulToken > 0) && (lenConsulDC > 0) {

					ConsulCustomConfig := api.DefaultConfig()
					ConsulCustomConfig.Datacenter = fmt.Sprintf("%s", CONSUL_DATACENTER)
					ConsulCustomConfig.Token = fmt.Sprintf("%s", CONSUL_TOKEN)

					client, err := api.NewClient(ConsulCustomConfig)
					if err != nil {
					log.Printf("Can't connect to Consul")
					
					}

					dispatcher := tasks.NewDispatcher(3, c.ShutdownCh, client)
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
						case <-c.ShutdownCh:
							sig = os.Interrupt
							shutdownChClosed = true
						}

						// Check if this is a SIGHUP
						if sig == syscall.SIGHUP {
							// TODO reload
						} else {
							if !shutdownChClosed {
								close(c.ShutdownCh)
							}
							
						}
					}
					
					
				} else {
					
					client, err := api.NewClient(api.DefaultConfig())
					if err != nil {
						log.Printf("Can't connect to Consul")
						
					}						
					dispatcher := tasks.NewDispatcher(3, c.ShutdownCh, client)
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
						case <-c.ShutdownCh:
							sig = os.Interrupt
							shutdownChClosed = true
						}

						// Check if this is a SIGHUP
						if sig == syscall.SIGHUP {
							// TODO reload
						} else {
							if !shutdownChClosed {
								close(c.ShutdownCh)
							}
							
						}
					}
						
				}
					
			},	
}
