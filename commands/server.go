package commands

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/server"
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
		shutdownCh := make(chan struct{})
		return server.RunServer(configuration, shutdownCh)
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
	serverCmd.PersistentFlags().StringP("os_private_network_name", "m", "", "Name of the private network")
	serverCmd.PersistentFlags().StringP("os_public_network_name", "e", "", "Name of the public network")

	//Flags definition for Consul
	serverCmd.PersistentFlags().StringP("consul_address", "", "", "Address of the HTTP interface for Consul (format: <host>:<port>)")
	serverCmd.PersistentFlags().StringP("consul_token", "t", "", "The token by default")
	serverCmd.PersistentFlags().StringP("consul_datacenter", "d", "", "The datacenter of Consul node")

	serverCmd.PersistentFlags().Int("rest_consul_publisher_max_routines", config.DEFAULT_REST_CONSUL_PUB_MAX_ROUTINES, "Maximum number of paralellism used by the REST API to store TOSCA definitions in Consul. If you increase the default value you may need to tweak the ulimit max open files. If set to 0 or less the default value will be used")

	//Bind Flags for OpenStack
	viper.BindPFlag("os_auth_url", serverCmd.PersistentFlags().Lookup("os_auth_url"))
	viper.BindPFlag("os_tenant_id", serverCmd.PersistentFlags().Lookup("os_tenant_id"))
	viper.BindPFlag("os_tenant_name", serverCmd.PersistentFlags().Lookup("os_tenant_name"))
	viper.BindPFlag("os_user_name", serverCmd.PersistentFlags().Lookup("os_user_name"))
	viper.BindPFlag("os_password", serverCmd.PersistentFlags().Lookup("os_password"))
	viper.BindPFlag("os_region", serverCmd.PersistentFlags().Lookup("os_region"))
	viper.BindPFlag("os_prefix", serverCmd.PersistentFlags().Lookup("os_prefix"))
	viper.BindPFlag("os_private_network_name", serverCmd.PersistentFlags().Lookup("os_private_network_name"))
	viper.BindPFlag("os_public_network_name", serverCmd.PersistentFlags().Lookup("os_public_network_name"))
	//Bind flags for Consul
	viper.BindPFlag("consul_address", serverCmd.PersistentFlags().Lookup("consul_address"))
	viper.BindPFlag("consul_token", serverCmd.PersistentFlags().Lookup("consul_token"))
	viper.BindPFlag("consul_datacenter", serverCmd.PersistentFlags().Lookup("consul_datacenter"))

	viper.BindPFlag("rest_consul_publisher_max_routines", serverCmd.PersistentFlags().Lookup("rest_consul_publisher_max_routines"))

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
	viper.BindEnv("os_private_network_name", "OS_PRIVATE_NETWORK_NAME")
	viper.BindEnv("os_public_network_name", "OS_PUBLIC_NETWORK_NAME")
	viper.BindEnv("rest_consul_publisher_max_routines")
	viper.BindEnv("consul_address")

	//Setting Defaults
	viper.SetDefault("os_prefix", "janus-")
	viper.SetDefault("os_region", "RegionOne")
	viper.SetDefault("consul_address", "") // Use consul api default
	viper.SetDefault("consul_datacenter", "dc1")
	viper.SetDefault("consul_token", "anonymous")
	viper.SetDefault("rest_consul_publisher_max_routines", config.DEFAULT_REST_CONSUL_PUB_MAX_ROUTINES)

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
	configuration.OS_PRIVATE_NETWORK_NAME = viper.GetString("os_private_network_name")
	configuration.OS_PUBLIC_NETWORK_NAME = viper.GetString("os_public_network_name")
	configuration.CONSUL_ADDRESS = viper.GetString("consul_address")
	configuration.CONSUL_DATACENTER = viper.GetString("consul_datacenter")
	configuration.CONSUL_TOKEN = viper.GetString("consul_token")
	configuration.REST_CONSUL_PUB_MAX_ROUTINES = viper.GetInt("rest_consul_publisher_max_routines")

	return configuration
}
