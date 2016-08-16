package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"novaforge.bull.com/starlings-janus/janus/jconfig"
	"os"
)

var cfgFile string

type Command struct {
	ShutdownCh chan struct{}
}

func Execute() {

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

var RootCmd = &cobra.Command{
	Use:   "janus",
	Short: "A new generation orchestrator",
	Long: `janus is the main command, used to start the http server.
Janus is a new generation orchestrator.  
It is cloud-agnostic, flexible and secure.
`,

	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println("Available Commands for Janus:")
		fmt.Println("  - server      Perform the server command")
		fmt.Println("  - version     Print the version")

	},
}

func init() {

	addCommand()
	setConfig()
}

func addCommand() {
	RootCmd.AddCommand(serverCmd)
	RootCmd.AddCommand(versionCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

}

func setConfig() {

	//Flags definition for OpenStack
	RootCmd.PersistentFlags().StringP("os_auth_url", "a", "", "will use the 1.1 *compute api*")
	RootCmd.PersistentFlags().StringP("os_tenant_id", "b", "", "The ID of the tenant")
	RootCmd.PersistentFlags().StringP("os_tenant_name", "c", "", "The name of the tenant")
	RootCmd.PersistentFlags().StringP("os_user_name", "d", "", "The username to authenticate")
	RootCmd.PersistentFlags().StringP("os_password", "p", "", "The password to authenticate")
	RootCmd.PersistentFlags().StringP("os_region", "r", "", "The region name")
	RootCmd.PersistentFlags().StringP("os_prefix", "k", "", "Prefix of the user")

	//Flags definition for Consul
	RootCmd.PersistentFlags().StringP("consul_token", "u", "", "The token by default")
	RootCmd.PersistentFlags().StringP("consul_datacenter", "v", "", "The datacenter of Consul node")

	//Bind Flags for OpenStack
	viper.BindPFlag("os_auth_url", RootCmd.PersistentFlags().Lookup("os_auth_url"))
	viper.BindPFlag("os_tenant_id", RootCmd.PersistentFlags().Lookup("os_tenant_id"))
	viper.BindPFlag("os_tenant_name", RootCmd.PersistentFlags().Lookup("os_tenant_name"))
	viper.BindPFlag("os_user_name", RootCmd.PersistentFlags().Lookup("os_user_name"))
	viper.BindPFlag("os_password", RootCmd.PersistentFlags().Lookup("os_password"))
	viper.BindPFlag("os_region", RootCmd.PersistentFlags().Lookup("os_region"))
	viper.BindPFlag("os_prefix", RootCmd.PersistentFlags().Lookup("os_prefix"))
	//Bind flags for Consul
	viper.BindPFlag("consul_token", RootCmd.PersistentFlags().Lookup("consul_token"))
	viper.BindPFlag("consul_datacenter", RootCmd.PersistentFlags().Lookup("consul_datacenter"))

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/janus/config.janus.json)")
	// Cobra also supports local flags, which will only run when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	//Environment Variables
	viper.SetEnvPrefix("os") // will be uppercased automatically - Become "OS_"
	viper.AutomaticEnv() // read in environment variables that match
	viper.BindEnv("os_auth_url", "OS_AUTH_URL")
	viper.BindEnv("os_tenant_id", "OS_TENANT_ID")
	viper.BindEnv("os_tenant_name", "OS_TENANT_NAME")
	viper.BindEnv("os_user_name", "OS_USERNAME")
	viper.BindEnv("os_password", "OS_PASSWORD")
	viper.BindEnv("os_region", "OS_REGION_NAME")
	viper.BindEnv("os_prefix", "OS_PREFIX")
	
	//Setting Defaults
	viper.SetDefault("os_prefix", "Janus-")
	viper.SetDefault("os_region", "RegionOne")
	viper.SetDefault("consul_datacenter", "dc1")
	viper.SetDefault("consul_token", "anonymous")

	//Configuration file directories
	viper.SetConfigName("config.janus") // name of config file (without extension)
	viper.AddConfigPath("/etc/janus/")  // adding home directory as first search path
	viper.AddConfigPath(".")	

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())

	} else {
		fmt.Println("Config not found... ")
	}

}

func getConfig(configuration jconfig.Configuration) jconfig.Configuration {

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
