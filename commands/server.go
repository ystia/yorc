package commands

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/helper/collections"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/server"
)

func init() {
	RootCmd.AddCommand(serverCmd)

	// Get the CLI args
	args := os.Args
	serverInitInfraExtraFlags(args)
	setConfig()
	cobra.OnInitialize(initConfig)
}

var cfgFile string

var serverExtraInfraParams []string

var serverCmd = &cobra.Command{
	Use:          "server",
	Short:        "Perform the server command",
	Long:         `Perform the server command`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		configuration := getConfig()
		log.Debugf("Configuration :%+v", configuration)
		shutdownCh := make(chan struct{})
		return server.RunServer(configuration, shutdownCh)
	},
}

func serverInitInfraExtraFlags(args []string) {
	serverExtraInfraParams = make([]string, 0)
	for i := range args {
		if strings.HasPrefix(args[i], "--infrastructure_") {
			var viperName, flagName string
			if strings.ContainsRune(args[i], '=') {
				// Handle the syntax --infrastructure_xxx_yyy = value
				flagParts := strings.Split(args[i], "=")
				flagName = strings.TrimLeft(flagParts[0], "-")
				viperName = strings.Replace(strings.Replace(flagName, "infrastructure_", "infrastructures.", 1), "_", ".", 1)
				if len(flagParts) == 1 {
					// Boolean flag
					serverCmd.PersistentFlags().Bool(flagName, false, "")
					viper.SetDefault(viperName, false)
				} else {
					serverCmd.PersistentFlags().String(flagName, "", "")
					viper.SetDefault(viperName, "")
				}
			} else {
				// Handle the syntax --infrastructure_xxx_yyy value
				flagName = strings.TrimLeft(args[i], "-")
				viperName = strings.Replace(strings.Replace(flagName, "infrastructure_", "infrastructures.", 1), "_", ".", 1)
				if len(args) > i+1 && !strings.HasPrefix(args[i+1], "--") {
					serverCmd.PersistentFlags().String(flagName, "", "")
					viper.SetDefault(viperName, "")
				} else {
					// Boolean flag
					serverCmd.PersistentFlags().Bool(flagName, false, "")
					viper.SetDefault(viperName, false)
				}
			}
			// Add viper flag
			viper.BindPFlag(viperName, serverCmd.PersistentFlags().Lookup(flagName))
			serverExtraInfraParams = append(serverExtraInfraParams, viperName)
		}
	}
	for _, envVar := range os.Environ() {
		if strings.HasPrefix(envVar, "JANUS_INFRA_") {
			envVarParts := strings.SplitN(envVar, "=", 2)
			viperName := strings.ToLower(strings.Replace(strings.Replace(envVarParts[0], "JANUS_INFRA_", "infrastructures.", 1), "_", ".", 1))
			viper.BindEnv(viperName, envVarParts[0])
			if !collections.ContainsString(serverExtraInfraParams, viperName) {
				serverExtraInfraParams = append(serverExtraInfraParams, viperName)
			}
		}
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		log.Println("Can't use config file:", err)
	}
}

func setConfig() {

	//Flags definition for Janus server
	serverCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is /etc/janus/config.janus.json)")
	serverCmd.PersistentFlags().String("plugins_directory", config.DefaultPluginDir, "The name of the plugins directory of the Janus server")
	serverCmd.PersistentFlags().StringP("working_directory", "w", "", "The name of the working directory of the Janus server")
	serverCmd.PersistentFlags().Int("workers_number", config.DefaultWorkersNumber, "Number of workers in the Janus server. If not set the default value will be used")
	serverCmd.PersistentFlags().Duration("graceful_shutdown_timeout", config.DefaultServerGracefulShutdownTimeout, "Timeout to  wait for a graceful shutdown of the Janus server. After this delay the server immediately exits.")
	serverCmd.PersistentFlags().Bool("keep_operation_remote_path", config.DefaultKeepOperationRemotePath, "Define wether the path created to store artifacts on the nodes will be removed at the end of workflow executions.")
	serverCmd.PersistentFlags().StringP("resources_prefix", "x", "", "Prefix created resources (like Computes and so on)")

	// Flags definition for Janus HTTP REST API
	serverCmd.PersistentFlags().Int("http_port", config.DefaultHTTPPort, "Port number for the Janus HTTP REST API. If omitted or set to '0' then the default port number is used, any positive integer will be used as it, and finally any negative value will let use a random port.")
	serverCmd.PersistentFlags().String("http_address", config.DefaultHTTPAddress, "Listening address for the Janus HTTP REST API.")
	serverCmd.PersistentFlags().String("key_file", "", "File path to a PEM-encoded private key. The key is used to enable SSL for the Janus HTTP REST API. This must be provided along with cert_file. If one of key_file or cert_file is not provided then SSL is disabled.")
	serverCmd.PersistentFlags().String("cert_file", "", "File path to a PEM-encoded certificate. The certificate is used to enable SSL for the Janus HTTP REST API. This must be provided along with key_file. If one of key_file or cert_file is not provided then SSL is disabled.")

	//Flags definition for Consul
	serverCmd.PersistentFlags().StringP("consul_address", "", "", "Address of the HTTP interface for Consul (format: <host>:<port>)")
	serverCmd.PersistentFlags().StringP("consul_token", "t", "", "The token by default")
	serverCmd.PersistentFlags().StringP("consul_datacenter", "d", "", "The datacenter of Consul node")
	serverCmd.PersistentFlags().String("consul_key_file", "", "The key file to use for talking to Consul over TLS")
	serverCmd.PersistentFlags().String("consul_cert_file", "", "The cert file to use for talking to Consul over TLS")
	serverCmd.PersistentFlags().String("consul_ca_cert", "", "CA cert to use for talking to Consul over TLS")
	serverCmd.PersistentFlags().String("consul_ca_path", "", "Path to a directory of CA certs to use for talking to Consul over TLS")
	serverCmd.PersistentFlags().Bool("consul_ssl", false, "Whether or not to use HTTPS")
	serverCmd.PersistentFlags().Bool("consul_ssl_verify", true, "Whether or not to disable certificate checking")

	serverCmd.PersistentFlags().Int("consul_publisher_max_routines", config.DefaultConsulPubMaxRoutines, "Maximum number of parallelism used to store TOSCA definitions in Consul. If you increase the default value you may need to tweak the ulimit max open files. If set to 0 or less the default value will be used")

	serverCmd.PersistentFlags().Bool("ansible_use_openssh", false, "Prefer OpenSSH over Paramiko a Python implementation of SSH (the default) to provision remote hosts")
	serverCmd.PersistentFlags().Bool("ansible_debug", false, "Prints massive debug information from Ansible")
	serverCmd.PersistentFlags().Int("ansible_connection_retries", 5, "Number of retries in case of Ansible SSH connection failure")
	serverCmd.PersistentFlags().String("operation_remote_base_dir", ".janus", "Name of the temporary directory used by Ansible on the nodes")

	//Bind flags for Consul
	viper.BindPFlag("consul_address", serverCmd.PersistentFlags().Lookup("consul_address"))
	viper.BindPFlag("consul_token", serverCmd.PersistentFlags().Lookup("consul_token"))
	viper.BindPFlag("consul_datacenter", serverCmd.PersistentFlags().Lookup("consul_datacenter"))
	viper.BindPFlag("consul_key_file", serverCmd.PersistentFlags().Lookup("consul_key_file"))
	viper.BindPFlag("consul_cert_file", serverCmd.PersistentFlags().Lookup("consul_cert_file"))
	viper.BindPFlag("consul_ca_cert", serverCmd.PersistentFlags().Lookup("consul_ca_cert"))
	viper.BindPFlag("consul_ca_path", serverCmd.PersistentFlags().Lookup("consul_ca_path"))
	viper.BindPFlag("consul_ssl", serverCmd.PersistentFlags().Lookup("consul_ssl"))
	viper.BindPFlag("consul_ssl_verify", serverCmd.PersistentFlags().Lookup("consul_ssl_verify"))

	viper.BindPFlag("consul_publisher_max_routines", serverCmd.PersistentFlags().Lookup("consul_publisher_max_routines"))

	//Bind Flags for Janus server
	viper.BindPFlag("working_directory", serverCmd.PersistentFlags().Lookup("working_directory"))
	viper.BindPFlag("plugins_directory", serverCmd.PersistentFlags().Lookup("plugins_directory"))
	viper.BindPFlag("workers_number", serverCmd.PersistentFlags().Lookup("workers_number"))
	viper.BindPFlag("server_graceful_shutdown_timeout", serverCmd.PersistentFlags().Lookup("graceful_shutdown_timeout"))
	viper.BindPFlag("keep_operation_remote_path", serverCmd.PersistentFlags().Lookup("keep_operation_remote_path"))
	viper.BindPFlag("resources_prefix", serverCmd.PersistentFlags().Lookup("resources_prefix"))

	//Bind Flags Janus HTTP REST API
	viper.BindPFlag("http_port", serverCmd.PersistentFlags().Lookup("http_port"))
	viper.BindPFlag("http_address", serverCmd.PersistentFlags().Lookup("http_address"))
	viper.BindPFlag("cert_file", serverCmd.PersistentFlags().Lookup("cert_file"))
	viper.BindPFlag("key_file", serverCmd.PersistentFlags().Lookup("key_file"))

	viper.BindPFlag("ansible_use_openssh", serverCmd.PersistentFlags().Lookup("ansible_use_openssh"))
	viper.BindPFlag("ansible_debug", serverCmd.PersistentFlags().Lookup("ansible_debug"))
	viper.BindPFlag("ansible_connection_retries", serverCmd.PersistentFlags().Lookup("ansible_connection_retries"))
	viper.BindPFlag("operation_remote_base_dir", serverCmd.PersistentFlags().Lookup("operation_remote_base_dir"))

	//Environment Variables
	viper.SetEnvPrefix("janus") // will be uppercased automatically - Become "JANUS_"
	viper.AutomaticEnv()        // read in environment variables that match
	viper.BindEnv("working_directory")
	viper.BindEnv("plugins_directory")
	viper.BindEnv("server_graceful_shutdown_timeout")
	viper.BindEnv("workers_number")
	viper.BindEnv("http_port")
	viper.BindEnv("http_address")
	viper.BindEnv("key_file")
	viper.BindEnv("cert_file")
	viper.BindEnv("resources_prefix")
	viper.BindEnv("consul_publisher_max_routines")
	viper.BindEnv("consul_address")
	viper.BindEnv("consul_key_file")
	viper.BindEnv("consul_cert_file")
	viper.BindEnv("consul_ca_cert")
	viper.BindEnv("consul_ca_path")
	viper.BindEnv("consul_ssl")
	viper.BindEnv("consul_ssl_verify")
	viper.BindEnv("keep_operation_remote_path")

	viper.BindEnv("ansible_use_openssh")
	viper.BindEnv("ansible_debug")
	viper.BindEnv("ansible_connection_retries")
	viper.BindEnv("operation_remote_base_dir")

	//Setting Defaults
	viper.SetDefault("working_directory", "work")
	viper.SetDefault("server_graceful_shutdown_timeout", config.DefaultServerGracefulShutdownTimeout)
	viper.SetDefault("plugins_directory", config.DefaultPluginDir)
	viper.SetDefault("http_port", config.DefaultHTTPPort)
	viper.SetDefault("http_address", config.DefaultHTTPAddress)
	viper.SetDefault("resources_prefix", "janus-")
	viper.SetDefault("consul_address", "") // Use consul api default
	viper.SetDefault("consul_datacenter", "dc1")
	viper.SetDefault("consul_token", "anonymous")
	viper.SetDefault("consul_publisher_max_routines", config.DefaultConsulPubMaxRoutines)
	viper.SetDefault("workers_number", config.DefaultWorkersNumber)
	viper.SetDefault("keep_operation_remote_path", config.DefaultKeepOperationRemotePath)

	viper.SetDefault("ansible_use_openssh", false)
	viper.SetDefault("ansible_debug", false)
	viper.SetDefault("ansible_connection_retries", 5)
	viper.SetDefault("operation_remote_base_dir", ".janus")

	//Configuration file directories
	viper.SetConfigName("config.janus") // name of config file (without extension)
	viper.AddConfigPath("/etc/janus/")  // adding home directory as first search path
	viper.AddConfigPath(".")

}

func getConfig() config.Configuration {
	configuration := config.Configuration{}
	configuration.AnsibleUseOpenSSH = viper.GetBool("ansible_use_openssh")
	configuration.AnsibleDebugExec = viper.GetBool("ansible_debug")
	configuration.AnsibleConnectionRetries = viper.GetInt("ansible_connection_retries")
	configuration.OperationRemoteBaseDir = viper.GetString("operation_remote_base_dir")
	configuration.WorkingDirectory = viper.GetString("working_directory")
	configuration.PluginsDirectory = viper.GetString("plugins_directory")
	configuration.WorkersNumber = viper.GetInt("workers_number")
	configuration.HTTPPort = viper.GetInt("http_port")
	configuration.HTTPAddress = viper.GetString("http_address")
	configuration.CertFile = viper.GetString("cert_file")
	configuration.KeyFile = viper.GetString("key_file")
	configuration.ResourcesPrefix = viper.GetString("resources_prefix")
	configuration.ConsulAddress = viper.GetString("consul_address")
	configuration.ConsulDatacenter = viper.GetString("consul_datacenter")
	configuration.ConsulToken = viper.GetString("consul_token")
	configuration.ConsulPubMaxRoutines = viper.GetInt("consul_publisher_max_routines")
	configuration.ConsulKey = viper.GetString("consul_key_file")
	configuration.ConsulCert = viper.GetString("consul_cert_file")
	configuration.ConsulCA = viper.GetString("consul_ca_cert")
	configuration.ConsulCAPath = viper.GetString("consul_ca_path")
	configuration.ConsulSSL = viper.GetBool("consul_ssl")
	configuration.ConsulSSLVerify = viper.GetBool("consul_ssl_verify")
	configuration.ServerGracefulShutdownTimeout = viper.GetDuration("server_graceful_shutdown_timeout")
	configuration.KeepOperationRemotePath = viper.GetBool("keep_operation_remote_path")

	configuration.Infrastructures = make(map[string]config.InfrastructureConfig)

	infras := viper.GetStringMap("infrastructures")
	for infraName, infraConf := range infras {
		infraConfMap, ok := infraConf.(map[string]interface{})
		if !ok {
			log.Fatalf("Invalid configuration format for infrastructure %q", infraName)
		}

		configuration.Infrastructures[infraName] = infraConfMap
	}

	for _, infraParam := range serverExtraInfraParams {
		addServerExtraInfraParams(&configuration, infraParam)
	}

	configuration.Telemetry.StatsdAddress = viper.GetString("telemetry.statsd_address")
	configuration.Telemetry.StatsiteAddress = viper.GetString("telemetry.statsite_address")
	configuration.Telemetry.ServiceName = viper.GetString("telemetry.service_name")
	configuration.Telemetry.PrometheusEndpoint = viper.GetBool("telemetry.expose_prometheus_endpoint")
	configuration.Telemetry.DisableHostName = viper.GetBool("telemetry.disable_hostname")
	configuration.Telemetry.DisableGoRuntimeMetrics = viper.GetBool("telemetry.disable_go_runtime_metrics")

	return configuration
}

func addServerExtraInfraParams(cfg *config.Configuration, infraParam string) {
	if cfg.Infrastructures == nil {
		cfg.Infrastructures = make(map[string]config.InfrastructureConfig)
	}
	paramParts := strings.Split(infraParam, ".")
	value := viper.Get(infraParam)
	params, ok := cfg.Infrastructures[paramParts[1]]
	if !ok {
		params = make(config.InfrastructureConfig)
		cfg.Infrastructures[paramParts[1]] = params
	}
	params[paramParts[2]] = value
}
