package commands

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/rest"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	RootCmd.AddCommand(serverCmd)
	setConfig()
	cobra.OnInitialize(initConfig)
}

var cfgFile string

var serverCmd = &cobra.Command{

	Use:   "server",
	Short: "Perform the server command",
	Long:  `Perform the server command`,
	RunE: func(cmd *cobra.Command, args []string) error {

		configuration := getConfig()

		var ConsulDC string = configuration.CONSUL_DATACENTER
		var ConsulToken string = configuration.CONSUL_TOKEN

		ConsulCustomConfig := api.DefaultConfig()
		ConsulCustomConfig.Datacenter = fmt.Sprintf("%s", ConsulDC)
		ConsulCustomConfig.Token = fmt.Sprintf("%s", ConsulToken)

		client, err := api.NewClient(ConsulCustomConfig)
		if err != nil {
			log.Printf("Can't connect to Consul")
			return err
		}
		shutdownCh := make(chan struct{})
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
	},
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Debugln("Using config file:", viper.ConfigFileUsed())
	} else {
		log.Debugln("Config not found... ")
	}
}

func setConfig() {

	//Flags definition for OpenStack
	serverCmd.PersistentFlags().StringP("os_auth_url", "a", "", "will use the 1.1 *compute api*")
	serverCmd.PersistentFlags().StringP("os_tenant_id", "i", "", "The ID of the tenant")
	serverCmd.PersistentFlags().StringP("os_tenant_name", "n", "", "The name of the tenant")
	serverCmd.PersistentFlags().StringP("os_user_name", "u", "", "The username to authenticate")
	serverCmd.PersistentFlags().StringP("os_password", "p", "", "The password to authenticate")
	serverCmd.PersistentFlags().StringP("os_region", "r", "", "The region name")
	serverCmd.PersistentFlags().StringP("os_prefix", "x", "", "Prefix of the user")

	//Flags definition for Consul
	serverCmd.PersistentFlags().StringP("consul_token", "t", "", "The token by default")
	serverCmd.PersistentFlags().StringP("consul_datacenter", "d", "", "The datacenter of Consul node")

	//Bind Flags for OpenStack
	viper.BindPFlag("os_auth_url", serverCmd.PersistentFlags().Lookup("os_auth_url"))
	viper.BindPFlag("os_tenant_id", serverCmd.PersistentFlags().Lookup("os_tenant_id"))
	viper.BindPFlag("os_tenant_name", serverCmd.PersistentFlags().Lookup("os_tenant_name"))
	viper.BindPFlag("os_user_name", serverCmd.PersistentFlags().Lookup("os_user_name"))
	viper.BindPFlag("os_password", serverCmd.PersistentFlags().Lookup("os_password"))
	viper.BindPFlag("os_region", serverCmd.PersistentFlags().Lookup("os_region"))
	viper.BindPFlag("os_prefix", serverCmd.PersistentFlags().Lookup("os_prefix"))
	//Bind flags for Consul
	viper.BindPFlag("consul_token", serverCmd.PersistentFlags().Lookup("consul_token"))
	viper.BindPFlag("consul_datacenter", serverCmd.PersistentFlags().Lookup("consul_datacenter"))

	serverCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is /etc/janus/config.janus.json)")

	//Environment Variables
	viper.SetEnvPrefix("janus") // will be uppercased automatically - Become "JANUS_"
	viper.AutomaticEnv()        // read in environment variables that match
	viper.BindEnv("os_auth_url", "OS_AUTH_URL")
	viper.BindEnv("os_tenant_id", "OS_TENANT_ID")
	viper.BindEnv("os_tenant_name", "OS_TENANT_NAME")
	viper.BindEnv("os_user_name", "OS_USERNAME")
	viper.BindEnv("os_password", "OS_PASSWORD")
	viper.BindEnv("os_region", "OS_REGION_NAME")
	viper.BindEnv("os_prefix", "OS_PREFIX")

	//Setting Defaults
	viper.SetDefault("os_prefix", "janus-")
	viper.SetDefault("os_region", "RegionOne")
	viper.SetDefault("consul_datacenter", "dc1")
	viper.SetDefault("consul_token", "anonymous")

	//Configuration file directories
	viper.SetConfigName("config.janus") // name of config file (without extension)
	viper.AddConfigPath("/etc/janus/")  // adding home directory as first search path
	viper.AddConfigPath(".")

}

func getConfig() config.Configuration {
	configuration := config.Configuration{}
	configuration.OS_AUTH_URL = viper.GetString("os_auth_url")
	configuration.OS_TENANT_ID = viper.GetString("os_tenant_id")
	configuration.OS_TENANT_NAME = viper.GetString("os_tenant_name")
	configuration.OS_USER_NAME = viper.GetString("os_user_name")
	configuration.OS_PASSWORD = viper.GetString("os_password")
	configuration.OS_REGION = viper.GetString("os_region")
	configuration.OS_PREFIX = viper.GetString("os_prefix")
	configuration.CONSUL_DATACENTER = viper.GetString("consul_datacenter")
	configuration.CONSUL_TOKEN = viper.GetString("consul_token")

	return configuration
}
