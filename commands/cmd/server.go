package cmd

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"novaforge.bull.com/starlings-janus/janus/commands/jconfig"
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

		c := new(Command)
		configuration := jconfig.Configuration{}
		configuration = getConfig(configuration)

		var ConsulDC string = configuration.Consul_datacenter
		var ConsulToken string = configuration.Consul_token

		//Custom configuration for Consul
		lenConsulDC := len(ConsulDC)
		lenConsulToken := len(ConsulToken)
		if (lenConsulToken > 0) && (lenConsulDC > 0) {

			ConsulCustomConfig := api.DefaultConfig()
			ConsulCustomConfig.Datacenter = fmt.Sprintf("%s", ConsulDC)
			ConsulCustomConfig.Token = fmt.Sprintf("%s", ConsulToken)

			client, err := api.NewClient(ConsulCustomConfig)
			if err != nil {
				log.Printf("Can't connect to Consul")

			}

			dispatcher := tasks.NewDispatcher(3, c.ShutdownCh, client, configuration)
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
			dispatcher := tasks.NewDispatcher(3, c.ShutdownCh, client, configuration)
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

func setConfig() {
	//Here we will define your flags and configuration settings. Cobra supports Persistent Flags, which, if defined here, will be global for your application.
	//Viper configuration keys are case insensitive. Note that Viper uses the following precedence order (each item takes precedence over the item below it):
	//1)explicit call to Set  2)flag  3)env  4)config  5)key/value store  6)default

	//Flags definition for OpenStack
	RootCmd.PersistentFlags().StringP("auth-url", "a", "", "will use the 1.1 *compute api*")
	RootCmd.PersistentFlags().StringP("tenant-id", "b", "", "The ID of the tenant")
	RootCmd.PersistentFlags().StringP("tenant-name", "c", "", "The name of the tenant")
	RootCmd.PersistentFlags().StringP("user-name", "d", "", "The username to authenticate")
	RootCmd.PersistentFlags().StringP("password", "p", "", "The password to authenticate")
	RootCmd.PersistentFlags().StringP("region", "r", "", "The region name")
	RootCmd.PersistentFlags().StringP("external-gateway", "e", "", "The external gateway")
	RootCmd.PersistentFlags().StringP("public-network-name", "f", "", "The public network name")
	RootCmd.PersistentFlags().StringP("prefix", "k", "", "Prefix of the user")
	RootCmd.PersistentFlags().StringP("keystone-user", "l", "", "The keystone user")
	RootCmd.PersistentFlags().StringP("keystone-password", "m", "", "The keystone password")
	RootCmd.PersistentFlags().StringP("keystone-tenant", "n", "", "The keystone tenant")
	RootCmd.PersistentFlags().StringP("keystone-url", "o", "", "The keystone URL")
	RootCmd.PersistentFlags().StringP("path", "w", "", "The path to the configuration file")

	//Flags definition for Consul
	RootCmd.PersistentFlags().StringP("consul-token", "u", "", "The token by default")
	RootCmd.PersistentFlags().StringP("consul-datacenter", "v", "", "The datacenter of Consul node")

	//Bind Flags for OpenStack
	viper.BindPFlag("auth-url", RootCmd.PersistentFlags().Lookup("auth-url"))
	viper.BindPFlag("tenant-id", RootCmd.PersistentFlags().Lookup("tenant-id"))
	viper.BindPFlag("tenant-name", RootCmd.PersistentFlags().Lookup("tenant-name"))
	viper.BindPFlag("user-name", RootCmd.PersistentFlags().Lookup("user-name"))
	viper.BindPFlag("password", RootCmd.PersistentFlags().Lookup("password"))
	viper.BindPFlag("region", RootCmd.PersistentFlags().Lookup("region"))
	viper.BindPFlag("external-gateway", RootCmd.PersistentFlags().Lookup("external-gateway"))
	viper.BindPFlag("public-network-name", RootCmd.PersistentFlags().Lookup("public-network-name"))
	viper.BindPFlag("prefix", RootCmd.PersistentFlags().Lookup("prefix"))
	viper.BindPFlag("keystone-user", RootCmd.PersistentFlags().Lookup("keystone-user"))
	viper.BindPFlag("keystone-password", RootCmd.PersistentFlags().Lookup("keystone-password"))
	viper.BindPFlag("keystone-tenant", RootCmd.PersistentFlags().Lookup("keystone-tenant"))
	viper.BindPFlag("keystone-url", RootCmd.PersistentFlags().Lookup("keystone-url"))
	viper.BindPFlag("path", RootCmd.PersistentFlags().Lookup("path"))
	//Bind flags for Consul
	viper.BindPFlag("consul-token", RootCmd.PersistentFlags().Lookup("consul-token"))
	viper.BindPFlag("consul-datacenter", RootCmd.PersistentFlags().Lookup("consul-datacenter"))

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/janus/config.janus.json)")
	// Cobra also supports local flags, which will only run when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	viper.SetEnvPrefix("janus")
	viper.AutomaticEnv() // read in environment variables that match

	//Setting Defaults
	viper.SetDefault("public-network-name", "Public_Network")
	viper.SetDefault("prefix", "Janus-")
	viper.SetDefault("path", "/etc/janus/")
	viper.SetDefault("consul-datacenter", "dc1")
	viper.SetDefault("consul-token", "anonymous")

	viper.SetConfigName("config.janus") // name of config file (without extension)
	viper.AddConfigPath("/etc/janus/")  // adding home directory as first search path

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())

	} else {
		fmt.Println("Config not found... ")
	}

}

func getConfig(configuration jconfig.Configuration) jconfig.Configuration {

	configuration.Auth_url = viper.GetString("auth-url")
	configuration.Tenant_id = viper.GetString("tenant-id")
	configuration.Tenant_name = viper.GetString("tenant-name")
	configuration.User_name = viper.GetString("user-name")
	configuration.Password = viper.GetString("password")
	configuration.Region = viper.GetString("region")
	configuration.External_gateway = viper.GetString("external-gateway")
	configuration.Public_network_name = viper.GetString("public-network-name")
	configuration.Prefix = viper.GetString("prefix")
	configuration.Keystone_user = viper.GetString("keystone-user")
	configuration.Keystone_password = viper.GetString("keystone-password")
	configuration.Keystone_tenant = viper.GetString("keystone-tenant")
	configuration.Keystone_url = viper.GetString("keystone-url")
	configuration.Path = viper.GetString("path")
	configuration.Consul_datacenter = viper.GetString("consul-datacenter")
	configuration.Consul_token = viper.GetString("consul-token")

	return configuration
}
