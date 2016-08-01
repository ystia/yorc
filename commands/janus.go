package commands

import (
    "fmt"
	"os"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

type Command struct { 
	ShutdownCh chan struct{}
}

func Execute() {

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

func init() {
        cobra.OnInitialize(initConfig)
		
	    RootCmd.AddCommand(serverCmd)
        RootCmd.AddCommand(versionCmd)

        // Here you will define your flags and configuration settings. Cobra supports Persistent Flags, which, if defined here, will be global for your application.

		flags := RootCmd.Flags()
		
		//Flags definition for OpenStack
		flags.String("auth-url","http://10.197.132.150:5000/v2.0","will use the 1.1 *compute api*")
		flags.String("tenant-id","1f219ffcff2342e189ef47c0e9689dfb","The ID of the tenant")
		flags.String("tenant-name","starlings","The name of the tenant")
		flags.String("username","starlingsuser","The username to authenticate")
		flags.String("password","starlingsuser","The password to authenticate")
		flags.String("region-name","RegionOne","The region name")
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
		flags.String("consul-token","anonymous","The token by default")
		flags.String("consul-datacenter","dc1","The datacenter of Consul node")

		//Bind Flags for OpenStack
		viper.BindPFlag("cl_auth_url",flags.Lookup("auth-url"))
		viper.BindPFlag("cl_tenant_id",flags.Lookup("tenant-id"))
		viper.BindPFlag("cl_tenant_name",flags.Lookup("tenant-name"))
		viper.BindPFlag("cl_tenant_name",flags.Lookup("tenant-name"))
		viper.BindPFlag("cl_username",flags.Lookup("username"))
		viper.BindPFlag("cl_password",flags.Lookup("password"))
		viper.BindPFlag("cl_region_name",flags.Lookup("region-name"))
		viper.BindPFlag("cl_cli_image_id",flags.Lookup("cli-image-id"))
		viper.BindPFlag("cl_manager_flavor_id",flags.Lookup("manager-flavor-id"))
		viper.BindPFlag("cl_manager_image_id",flags.Lookup("manager-image-id"))
		viper.BindPFlag("cl_external_gateway",flags.Lookup("external-gateway"))
		viper.BindPFlag("cl_public_network_name",flags.Lookup("public-network-name"))
		viper.BindPFlag("cl_prefix",flags.Lookup("prefix"))
		viper.BindPFlag("cl_keystone_user",flags.Lookup("keystone-user"))
		viper.BindPFlag("cl_keystone_password",flags.Lookup("keystone-password"))
		viper.BindPFlag("cl_keystone_tenant",flags.Lookup("keystone-tenant"))
		viper.BindPFlag("cl_keystone_url",flags.Lookup("keystone-url"))
		viper.BindPFlag("cl_cli_availability_zone",flags.Lookup("cli-availability-zone"))
		viper.BindPFlag("cl_manager_availability_zone",flags.Lookup("manager-availability-zone"))
		
		//Bind flags for Consul
		viper.BindPFlag("cl_consul_token",flags.Lookup("consul-token"))
		viper.BindPFlag("cl_consul_datacenter",flags.Lookup("consul-datacenter"))

        RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/janus/config.janus.json)")
        // Cobra also supports local flags, which will only run when this action is called directly.
        RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	   //It will check if the command line provide a value; otherwise, we will check into the configuration file.
	   //All the value from the command line start with the "cl_" prefix (cl for command line)
		if viper.IsSet("cl_auth-url") {
				AUTH_URL := viper.GetString("cl_auth-url") // case insensitive Setting & Getting
		} else { 
			if viper.IsSet("auth-url") {
				AUTH_URL := viper.GetString("auth-url") }
		}

		if viper.IsSet("cl_tenant_id") {
				TENANT_ID := viper.GetString("cl_tenant_id")
		} else {
			if viper.IsSet("tenant_id") {
				TENANT_ID := viper.GetString("tenant_id") }
		}

        if viper.IsSet("cl_tenant_name") {
				TENANT_NAME := viper.GetString("cl_tenant_name")
		} else {
			if viper.IsSet("tenant_name") {
				TENANT_NAME := viper.GetString("tenant_name") }
		}

        if viper.IsSet("cl_username") {
				USERNAME := viper.GetString("cl_username")
		} else { 
			if viper.IsSet("username") {
				USERNAME := viper.GetString("username") }
		}
		
		if viper.IsSet("cl_password") {
				PASSWORD := viper.GetString("cl_password")
		} else { 
			if viper.IsSet("password") {
				PASSWORD := viper.GetString("password") }
		}

		if viper.IsSet("cl_region_name") {
				REGION_NAME := viper.GetString("cl_region_name")
		} else {
			if viper.IsSet("region_name") {
				REGION_NAME := viper.GetString("region_name") }
		}

        if viper.IsSet("cl_cli_image_id") {
				CLI_IMAGE_ID := viper.GetString("cl_cli_image_id")
		} else {
			if viper.IsSet("cli_image_id") {
				CLI_IMAGE_ID := viper.GetString("cli_image_id") }
		}

        if viper.IsSet("cl_manager_flavor_id") {
				MANAGER_FLAVOR_ID := viper.GetString("cl_manager_flavor_id")
		} else { 
			if viper.IsSet("manager_flavor_id") {
				MANAGER_FLAVOR_ID := viper.GetString("manager_flavor_id") }
		}

        if viper.IsSet("cl_manager_image_id") {
				MANAGER_IMAGE_ID := viper.GetString("cl_manager_image_id")
		} else {
			 if viper.IsSet("manager_image_id") {
				MANAGER_IMAGE_ID := viper.GetString("manager_image_id") }
		}

        if viper.IsSet("cl_external_gateway") {
				EXTERNAL_GATEWAY := viper.GetString("cl_external_gateway")
		} else {
			 if viper.IsSet("external_gateway") {
				EXTERNAL_GATEWAY := viper.GetString("external_gateway") }
		}

        if viper.IsSet("cl_public_network_name") {
				PUBLIC_NETWORK_NAME := viper.GetString("cl_public_network_name")
		} else {
			if viper.IsSet("public_network_name") {
				PUBLIC_NETWORK_NAME := viper.GetString("public_network_name") }
		}

        if viper.IsSet("cl_prefix") {
				PREFIX := viper.GetString("cl_prefix")
		} else {
			 if viper.IsSet("prefix") {
				PREFIX := viper.GetString("prefix") }
		}

        if viper.IsSet("cl_keystone_user") {
				KEYSTONE_USER := viper.GetString("cl_keystone_user")
		} else {
			if viper.IsSet("keystone_user") {
				KEYSTONE_USER := viper.GetString("keystone_user") }
		}

		if viper.IsSet("cl_keystone_password") {
				KEYSTONE_PASSWORD := viper.GetString("cl_keystone_password")
		} else {
			if viper.IsSet("keystone_password") {
				KEYSTONE_PASSWORD := viper.GetString("keystone_password") }
		}

        if viper.IsSet("cl_keystone_tenant") {
				KEYSTONE_TENANT := viper.GetString("cl_keystone_tenant")
		} else { 
			if viper.IsSet("keystone_tenant") {
				KEYSTONE_TENANT := viper.GetString("keystone_tenant") }
		}

        if viper.IsSet("cl_keystone_url") {
				KEYSTONE_URL := viper.GetString("cl_keystone_url")
		} else {
			if viper.IsSet("keystone_url") {
				KEYSTONE_URL := viper.GetString("keystone_url") }
		}

        if viper.IsSet("cl_cli_availability_zone") {
				CLI_AVAILABILITY_ZONE := viper.GetString("cl_cli_availability_zone")
		} else {
			if viper.IsSet("cli_availability_zone") {
				CLI_AVAILABILITY_ZONE := viper.GetString("cli_availability_zone") }
		}

        if viper.IsSet("cl_manager_availability_zone") {
				MANAGER_AVAILABILITY_ZONE := viper.GetString("cl_manager_availability_zone")
		} else {
			if viper.IsSet("manager_availability_zone") {
				MANAGER_AVAILABILITY_ZONE := viper.GetString("manager_availability_zone") }
		}

        if viper.IsSet("cl_ssh_user_name") {
				SSH_USER_NAME := viper.GetString("cl_ssh_user_name")
		} else {
			if viper.IsSet("ssh_user_name") {
				SSH_USER_NAME := viper.GetString("ssh_user_name") }
		}

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
        if cfgFile != "" { // enable ability to specify config file via flag
                viper.SetConfigFile(cfgFile)
        }

        viper.SetConfigName("config.janus") // name of config file (without extension)
		viper.AddConfigPath("/etc/janus")   // adding home directory as first search path
		//viper.AddConfigPath(".") 
		
		viper.SetEnvPrefix()
		viper.AutomaticEnv()          // read in environment variables that match
		
        // If a config file is found, read it in.
        if err := viper.ReadInConfig()
		err == nil {
                fmt.Println("Using config file:", viper.ConfigFileUsed())
						
        } else {
				fmt.Println("Config not found... ")
		}

		/*
		viper.WatchConfig()
        viper.OnConfigChange(func(e fsnotify.Event) {
            fmt.Println("Config file changed:", e.Name)
        })
		*/

}

