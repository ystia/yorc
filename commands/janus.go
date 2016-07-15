package commands

import (
    "fmt"
	//"log"
	//"novaforge.bull.com/starlings-janus/janus/rest"
	//"novaforge.bull.com/starlings-janus/janus/tasks"
	"os"
	//"os/signal"
	//"syscall"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"encoding/json"
)


var cfgFile string

type Command struct { 
	ShutdownCh chan struct{}
}

func Execute() {

	AddCommands()

	if err := RootCmd.Execute(); err!= nil {
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

func AddCommands() {
	
	RootCmd.AddCommand(serverCmd)
	RootCmd.AddCommand(versionCmd)
}


func init() {
        cobra.OnInitialize(initConfig)
		
		
        // Here you will define your flags and configuration settings. Cobra supports Persistent Flags, which, if defined here, will be global for your application.

		flags := RootCmd.Flags()
		
		//Flags definition for OpenStack
		flags.String("os-auth-url","http://10.197.132.150:5000/v2.0","will use the 1.1 *compute api*")
		flags.String("os-tenant-id","1f219ffcff2342e189ef47c0e9689dfb","The ID of the tenant")
		flags.String("os-tenant-name","starlings","The name of the tenant")
		flags.String("os-username","starlingsuser","The username to authenticate")
		flags.String("os-password","starlingsuser","The password to authenticate")
		flags.String("os-region-name","RegionOne","The region name")
		flags.String("cli-image-id","c773e8c9-0962-437f-8f50-833d46fdc8e4","The ID of the CLI image")
		flags.String("manager-flavor-id","3","The ID of the manager flavor")
		flags.String("manager-image-id","89ec515c-3251-4c2f-8402-bda280c31650","The ID of the manager image")
		flags.String("external-gateway","672a4127-41a1-482e-99c4-12ab31654ed2","The external gateway")
		flags.String("public-network-name","Public_Network","The public zone name")
		flags.String("prefix","KW","Prefix of the user")
		flags.String("keystone-user","starlingsuser","The keystone user")
		flags.String("keystone-password","starlingsuser","The keystone password")
		flags.String("keystone-tenant","starlings","The keystone tenant")
		flags.String("keystone-url","http://10.197.132.150:5000/v2.0","The keystone URL")
		flags.String("cli-availability-zone","r423-01.share.ops","The CLI availability zone")
		flags.String("manager-availability-zone","r423-01.share.ops","The manager availability zone")
		
		//Flags definition for Consul
		flags.String("consul-ip","localhost","The IP address of the Consul node")
		flags.String("consul-port","8500","The port to access to Consul")

		//Bind Flags for OpenStack
		viper.BindPFlag("os-auth-url",flags.Lookup("os-auth-url"))
		viper.BindPFlag("os-tenant-id",flags.Lookup("os-tenant-id"))
		viper.BindPFlag("os-tenant-name",flags.Lookup("os-tenant-name"))
		viper.BindPFlag("os-tenant-name",flags.Lookup("os-tenant-name"))
		viper.BindPFlag("os-username",flags.Lookup("os-username"))
		viper.BindPFlag("os-password",flags.Lookup("os-password"))
		viper.BindPFlag("os-region-name",flags.Lookup("os-region-name"))
		viper.BindPFlag("cli-image-id",flags.Lookup("cli-image-id"))
		viper.BindPFlag("manager-flavor-id",flags.Lookup("manager-flavor-id"))
		viper.BindPFlag("manager-image-id",flags.Lookup("manager-image-id"))
		viper.BindPFlag("external-gateway",flags.Lookup("external-gateway"))
		viper.BindPFlag("public-network-name",flags.Lookup("public-network-name"))
		viper.BindPFlag("prefix",flags.Lookup("prefix"))
		viper.BindPFlag("keystone-user",flags.Lookup("keystone-user"))
		viper.BindPFlag("keystone-password",flags.Lookup("keystone-password"))
		viper.BindPFlag("keystone-tenant",flags.Lookup("keystone-tenant"))
		viper.BindPFlag("keystone-url",flags.Lookup("keystone-url"))
		viper.BindPFlag("cli-availability-zone",flags.Lookup("cli-availability-zone"))
		viper.BindPFlag("manager-availability-zone",flags.Lookup("manager-availability-zone"))
		
		//Bind flags for Consul
		viper.BindPFlag("consul-ip",flags.Lookup("consul-ip"))
		viper.BindPFlag("consul-port",flags.Lookup("consul-port"))

        RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/janus/config.janus.json)")
        // Cobra also supports local flags, which will only run when this action is called directly.
        RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
        if cfgFile != "" { // enable ability to specify config file via flag
                viper.SetConfigFile(cfgFile)
        }

        viper.SetConfigName("config.janus") // name of config file (without extension)
		viper.AddConfigPath("/etc/janus")   // adding home directory as first search path
		//viper.AddConfigPath(".") 
		
		viper.SetEnvPrefix("os")
		viper.SetEnvPrefix("tf_var")
		viper.AutomaticEnv()          // read in environment variables that match
		
        // If a config file is found, read it in.
        if err := viper.ReadInConfig()
		err == nil {
                fmt.Println("Using config file:", viper.ConfigFileUsed())
				
				//AUTH_URL := viper.GetString("OS_AUTH_URL")
				//TENANT_ID := viper.GetString("OS_TENANT_ID")
				//TENANT_NAME := viper.GetString("OS_TENANT_NAME")
				//USERNAME := viper.GetString("OS_USERNAME")
				//PASSWORD := viper.GetString("OS_PASSWORD")
				//REGION_NAME := viper.GetString("OS_REGION_NAME")
				//CLI_IMAGE_ID := viper.GetString("TF_VAR_cli_image_id")
				//MANAGER_FLAVOR_ID := viper.GetString("TF_VAR_manager_flavor_id")
				//MANAGER_IMAGE_ID := viper.GetString("TF_VAR_manager_image_id")
				//EXTERNAL_GATEWAY := viper.GetString("TF_VAR_external_gateway")
				//PUBLIC_NETWORK_NAME := viper.GetString("TF_VAR_public_network_name")
				//PREFIX := viper.GetString("TF_VAR_prefix")
				//KEYSTONE_USER := viper.GetString("TF_VAR_keystone_user")
				//KEYSTONE_PASSWORD := viper.GetString("TF_VAR_keystone_password")
				//KEYSTONE_TENANT := viper.GetString("TF_VAR_keystone_tenant")
				//KEYSTONE_URL := viper.GetString("TF_VAR_keystone_url")
				//CLI_AVAILABILITY_ZONE := viper.GetString("TF_VAR_cli_availability_zone")
				//MANAGER_AVAILABILITY_ZONE := viper.GetString("TF_VAR_manager_availability_zone")
				
				//After retrieving the values from the configuration file of Janus, 
				//there is the creation of terraform file in json : <prefix>janus.tf.json 
				//fileHandle, err := os.Create("janus.tf.json")
				//if err != nil {
				//	fmt.Println(err)
				//}
				//writer := bufio.NewWriter(fo)
				//fmt.Fprintln(writer, "String I want to write")
				//writer.Flush()
				
				//CONSUL_ADDRESS := viper.GetString("consul_address")
				//CONSUL_SCHEME := viper.GetString("consul_scheme")
				//CONSUL_DATACENTER := viper.GetString("consul_datacenter")
				//CONSUL_TOKEN := viper.GetString("consul_token")
						
        } else {
				fmt.Println("Config not found... ")
		}
}