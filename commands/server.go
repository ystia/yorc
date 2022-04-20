// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/server"
)

func init() {
	RootCmd.AddCommand(serverCmd)

	// Get the CLI args
	args := os.Args

	serverInitExtraFlags(args)
	setConfig()
}

const (
	// EnvironmentVariablePrefix is the prefix used in Yorc commands parameters
	// passed as environment variables
	EnvironmentVariablePrefix = "YORC"
)

var (
	// TfConsulPluginVersion is the Terraform Consul plugin lowest supported version
	TfConsulPluginVersion           = "tf Consul plugin version"
	tfConsulPluginVersionConstraint = versionToConstraint("~>", TfConsulPluginVersion, "minor")

	// TfAWSPluginVersion is the Terraform AWS plugin lowest supported version
	TfAWSPluginVersion           = "tf AWS plugin version"
	tfAWSPluginVersionConstraint = versionToConstraint("~>", TfAWSPluginVersion, "minor")

	// TfOpenStackPluginVersion is the Terraform OpenStack plugin lowest supported version
	TfOpenStackPluginVersion           = "tf OpenStack plugin version"
	tfOpenStackPluginVersionConstraint = versionToConstraint("~>", TfOpenStackPluginVersion, "minor")

	// TfGooglePluginVersion is the Terraform Google plugin lowest supported version
	TfGooglePluginVersion           = "tf Google plugin version"
	tfGooglePluginVersionConstraint = versionToConstraint("~>", TfGooglePluginVersion, "minor")
)

var ansibleConfiguration = map[string]interface{}{
	"ansible.use_openssh":                  false,
	"ansible.debug":                        false,
	"ansible.connection_retries":           5,
	"ansible.operation_remote_base_dir":    ".yorc",
	"ansible.keep_operation_remote_path":   config.DefaultKeepOperationRemotePath,
	"ansible.archive_artifacts":            config.DefaultArchiveArtifacts,
	"ansible.cache_facts":                  config.DefaultCacheFacts,
	"ansible.keep_generated_recipes":       false,
	"ansible.job_monitoring_time_interval": config.DefaultAnsibleJobMonInterval,
}

var consulConfiguration = map[string]interface{}{
	"consul.address":                "",
	"consul.token":                  "anonymous",
	"consul.datacenter":             "dc1",
	"consul.key_file":               "",
	"consul.cert_file":              "",
	"consul.ca_cert":                "",
	"consul.ca_path":                "",
	"consul.ssl":                    false,
	"consul.ssl_verify":             true,
	"consul.publisher_max_routines": config.DefaultConsulPubMaxRoutines,
	"consul.tls_handshake_timeout":  config.DefaultConsulTLSHandshakeTimeout,
}

var terraformConfiguration = map[string]interface{}{
	"terraform.plugins_dir":                         "",
	"terraform.consul_plugin_version_constraint":    tfConsulPluginVersionConstraint,
	"terraform.aws_plugin_version_constraint":       tfAWSPluginVersionConstraint,
	"terraform.google_plugin_version_constraint":    tfGooglePluginVersionConstraint,
	"terraform.openstack_plugin_version_constraint": tfOpenStackPluginVersionConstraint,
	"terraform.keep_generated_files":                false,
}

var cfgFile string

var resolvedServerExtraParams []*serverExtraParams

type serverExtraParams struct {
	argPrefix   string
	envPrefix   string
	viperPrefix string
	viperNames  []string
	subSplit    int
	storeFn     serverExtraParamStoreFn
	readConfFn  serverExtraParamReadConf
}

type serverExtraParamStoreFn func(cfg *config.Configuration, param string)
type serverExtraParamReadConf func(cfg *config.Configuration)

var serverCmd = &cobra.Command{
	Use:          "server",
	Short:        "Perform the server command",
	Long:         `Perform the server command`,
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initConfig()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Println("Using config file:", viper.ConfigFileUsed())
		shutdownCh := make(chan struct{})
		return RunServer(shutdownCh)
	},
}

// RunServer starts a Yorc Server
func RunServer(shutdownCh chan struct{}) error {
	configuration := GetConfig()
	log.Debugf("Configuration :%+v", configuration)
	return server.RunServer(configuration, shutdownCh)
}

func serverInitExtraFlags(args []string) {
	InitExtraFlags(args, serverCmd)
}

// InitExtraFlags inits vault flags
func InitExtraFlags(args []string, cmd *cobra.Command) {

	resolvedServerExtraParams = []*serverExtraParams{
		&serverExtraParams{
			argPrefix:   "vault_",
			envPrefix:   "YORC_VAULT_",
			viperPrefix: "vault.",
			viperNames:  make([]string, 0),
			storeFn:     addServerExtraVaultParam,
			readConfFn:  readVaultViperConfig,
		},
	}

	for _, sep := range resolvedServerExtraParams {
		for i := range args {
			if strings.HasPrefix(args[i], "--"+sep.argPrefix) {
				var viperName, flagName string
				if strings.ContainsRune(args[i], '=') {
					// Handle the syntax --vault_xxx_yyy = value
					flagParts := strings.Split(args[i], "=")
					flagName = strings.TrimLeft(flagParts[0], "-")
					viperName = strings.Replace(strings.Replace(flagName, sep.argPrefix, sep.viperPrefix, 1), "_", ".", sep.subSplit)
					if len(flagParts) == 1 {
						// Boolean flag
						cmd.PersistentFlags().Bool(flagName, false, "")
						viper.SetDefault(viperName, false)
					} else {
						cmd.PersistentFlags().String(flagName, "", "")
						viper.SetDefault(viperName, "")
					}
				} else {
					// Handle the syntax --vault_xxx_yyy value
					flagName = strings.TrimLeft(args[i], "-")
					viperName = strings.Replace(strings.Replace(flagName, sep.argPrefix, sep.viperPrefix, 1), "_", ".", sep.subSplit)
					if len(args) > i+1 && !strings.HasPrefix(args[i+1], "--") {

						// Arguments ending wih a plural 's' are considered to
						// be slices
						if strings.HasSuffix(args[i], "s") && !strings.HasSuffix(args[i], "credentials") {
							// May have already been defined as string slice
							// flags can appear several times
							if cmd.PersistentFlags().Lookup(flagName) == nil {
								cmd.PersistentFlags().StringSlice(flagName, []string{}, "")
								viper.SetDefault(viperName, []string{})
							}
						} else {
							cmd.PersistentFlags().String(flagName, "", "")
							viper.SetDefault(viperName, "")
						}
					} else {
						// Boolean flag
						cmd.PersistentFlags().Bool(flagName, false, "")
						viper.SetDefault(viperName, false)
					}
				}
				// Add viper flag
				viper.BindPFlag(viperName, cmd.PersistentFlags().Lookup(flagName))
				sep.viperNames = append(sep.viperNames, viperName)
			}
		}
		for _, envVar := range os.Environ() {
			if strings.HasPrefix(envVar, sep.envPrefix) {
				envVarParts := strings.SplitN(envVar, "=", 2)
				viperName := strings.ToLower(strings.Replace(strings.Replace(envVarParts[0], sep.envPrefix, sep.viperPrefix, 1), "_", ".", sep.subSplit))
				viper.BindEnv(viperName, envVarParts[0])
				if !collections.ContainsString(sep.viperNames, viperName) {
					sep.viperNames = append(sep.viperNames, viperName)
				}
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
	if err := viper.ReadInConfig(); err != nil {
		log.Println("Can't use config file:", err)
	} else {
		// Watch config to take into account config changes
		viper.WatchConfig()
		viper.OnConfigChange(func(e fsnotify.Event) {
			log.Printf("Reloading config on config file %s change\n", e.Name)
			viper.ReadInConfig()
		})
	}

}

func setConfig() {
	host, err := os.Hostname()
	if err != nil {
		host = "server_0"
		log.Printf("Failed to get system hostname: %v. We will try to use the default id (%q) for this instance.", err, host)
	}

	//Flags definition for Yorc server
	serverCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is /etc/yorc/config.yorc.json)")
	serverCmd.PersistentFlags().String("plugins_directory", config.DefaultPluginDir, "The name of the plugins directory of the Yorc server")
	serverCmd.PersistentFlags().Duration("plugins_health_check_time_interval", config.DefaultPluginsHealthCheckTimeInterval, "Plugins health check time interval")
	serverCmd.PersistentFlags().StringP("working_directory", "w", "", "The name of the working directory of the Yorc server")
	serverCmd.PersistentFlags().Int("workers_number", config.DefaultWorkersNumber, "Number of workers in the Yorc server. If not set the default value will be used")
	serverCmd.PersistentFlags().Duration("graceful_shutdown_timeout", config.DefaultServerGracefulShutdownTimeout, "Timeout to  wait for a graceful shutdown of the Yorc server. After this delay the server immediately exits.")
	serverCmd.PersistentFlags().StringP("resources_prefix", "x", "", "Prefix created resources (like Computes and so on)")
	serverCmd.PersistentFlags().Duration("wf_step_graceful_termination_timeout", config.DefaultWfStepGracefulTerminationTimeout, "Timeout to wait for a graceful termination of a workflow step during concurrent workflow step failure. After this delay the step is set on error.")
	serverCmd.PersistentFlags().Duration("purged_deployments_eviction_timeout", config.DefaultPurgedDeploymentsEvictionTimeout, "When a deployment is purged an event is keep to trace that the purge was actually done, this timeout controls the retention time of such events.")
	serverCmd.PersistentFlags().String("server_id", host, "The server ID used to identify the server node in a cluster.")
	serverCmd.PersistentFlags().Bool("disable_ssh_agent", false, "Allow disabling ssh-agent use for SSH authentication on provisioned computes. Default is false. If true, compute credentials must provide a path to a private key file instead of key content.")
	serverCmd.PersistentFlags().String("locations_file_path", "", "File path to locations configuration. This configuration is taken in account for the first time the server starts.")
	serverCmd.PersistentFlags().Int("concurrency_limit_for_upgrades", config.DefaultUpgradesConcurrencyLimit, "Limit of concurrency used in Upgrade processes. If not set the default value will be used")
	serverCmd.PersistentFlags().Duration("ssh_connection_timeout", config.DefaultSSHConnectionTimeout, "Timeout to establish SSH connection from Yorc SSH client. If not set the default value will be used")
	serverCmd.PersistentFlags().Duration("ssh_connection_retry_backoff", config.DefaultSSHConnectionRetryBackoff, "Backoff duration before retrying an ssh connection. This may be superseded by a location attribute if supported.")
	serverCmd.PersistentFlags().Uint64("ssh_connection_max_retries", config.DefaultSSHConnectionMaxRetries, "Maximum number of retries (attempts are retries + 1) before giving-up to connect. This may be superseded by a location attribute if supported.")

	serverCmd.PersistentFlags().Duration("tasks_dispatcher_long_poll_wait_time", config.DefaultTasksDispatcherLongPollWaitTime, "Wait time when long polling for executions tasks to dispatch to workers")
	serverCmd.PersistentFlags().Duration("tasks_dispatcher_lock_wait_time", config.DefaultTasksDispatcherLockWaitTime, "Wait time for acquiring a lock for an execution task")
	serverCmd.PersistentFlags().Duration("tasks_dispatcher_metrics_refresh_time", config.DefaultTasksDispatcherMetricsRefreshTime, "Tasks dispatcher metrics refresh time")

	// Flags definition for Yorc HTTP REST API
	serverCmd.PersistentFlags().Int("http_port", config.DefaultHTTPPort, "Port number for the Yorc HTTP REST API. If omitted or set to '0' then the default port number is used, any positive integer will be used as it, and finally any negative value will let use a random port.")
	serverCmd.PersistentFlags().String("http_address", config.DefaultHTTPAddress, "Listening address for the Yorc HTTP REST API.")
	serverCmd.PersistentFlags().String("key_file", "", "File path to a PEM-encoded private key. The key is used to enable SSL for the Yorc HTTP REST API. This must be provided along with cert_file. If one of key_file or cert_file is not provided then SSL is disabled.")
	serverCmd.PersistentFlags().String("ca_file", "", "File path to a PEM-encoded CA certificate to use for talking to yorc over TLS")
	serverCmd.PersistentFlags().String("ca_path", "", "Path to a directory of CA certs to use for talking to yorc over TLS")
	serverCmd.PersistentFlags().String("cert_file", "", "File path to a PEM-encoded certificate. The certificate is used to enable SSL for the Yorc HTTP REST API. This must be provided along with key_file. If one of key_file or cert_file is not provided then SSL is disabled.")
	serverCmd.PersistentFlags().Bool("ssl_verify", false, "Whether or not enable client certificate checking by the server")

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
	serverCmd.PersistentFlags().Duration("consul_tls_handshake_timeout", config.DefaultConsulTLSHandshakeTimeout, "Maximum duration to wait for a TLS handshake with Consul")

	serverCmd.PersistentFlags().Bool("ansible_use_openssh", false, "Prefer OpenSSH over Paramiko a Python implementation of SSH (the default) to provision remote hosts")
	serverCmd.PersistentFlags().Bool("ansible_debug", false, "Prints massive debug information from Ansible")
	serverCmd.PersistentFlags().Int("ansible_connection_retries", 5, "Number of retries in case of Ansible SSH connection failure")
	serverCmd.PersistentFlags().String("operation_remote_base_dir", ".yorc", "Name of the temporary directory used by Ansible on the nodes")
	serverCmd.PersistentFlags().Bool("keep_operation_remote_path", config.DefaultKeepOperationRemotePath, "Define wether the path created to store artifacts on the nodes will be removed at the end of workflow executions.")
	serverCmd.PersistentFlags().Bool("ansible_archive_artifacts", config.DefaultArchiveArtifacts, "Define wether artifacts should be ./archived before being copied on remote nodes (requires tar to be installed on remote nodes).")
	serverCmd.PersistentFlags().Bool("ansible_cache_facts", config.DefaultCacheFacts, "Define wether Ansible facts (useful variables about remote hosts) should be cached.")
	serverCmd.PersistentFlags().Bool("ansible_keep_generated_recipes", false, "Define if Yorc should not delete generated Ansible recipes")
	serverCmd.PersistentFlags().Duration("ansible_job_monitoring_time_interval", config.DefaultAnsibleJobMonInterval, "Default duration for monitoring time interval for jobs handled by Ansible")

	//Flags definition for Terraform
	serverCmd.PersistentFlags().Bool("terraform_keep_generated_files", false, "Define if Yorc should not delete generated Terraform infrastructures files")

	//Flags definition for Terraform
	serverCmd.PersistentFlags().StringP("terraform_plugins_dir", "", "", "The directory where to find Terraform plugins")
	serverCmd.PersistentFlags().StringP("terraform_consul_plugin_version_constraint", "", tfConsulPluginVersionConstraint, "Terraform Consul plugin version constraint.")
	serverCmd.PersistentFlags().StringP("terraform_aws_plugin_version_constraint", "", tfAWSPluginVersionConstraint, "Terraform AWS plugin version constraint.")
	serverCmd.PersistentFlags().StringP("terraform_openstack_plugin_version_constraint", "", tfOpenStackPluginVersionConstraint, "Terraform OpenStack plugin version constraint.")
	serverCmd.PersistentFlags().StringP("terraform_google_plugin_version_constraint", "", tfGooglePluginVersionConstraint, "Terraform Google plugin version constraint.")

	//Bind Consul persistent flags
	for key := range consulConfiguration {
		viper.BindPFlag(key, serverCmd.PersistentFlags().Lookup(toFlatKey(key)))
	}

	//Bind Flags for Yorc server
	viper.BindPFlag("working_directory", serverCmd.PersistentFlags().Lookup("working_directory"))
	viper.BindPFlag("plugins_directory", serverCmd.PersistentFlags().Lookup("plugins_directory"))
	viper.BindPFlag("plugins_health_check_time_interval", serverCmd.PersistentFlags().Lookup("plugins_health_check_time_interval"))
	viper.BindPFlag("workers_number", serverCmd.PersistentFlags().Lookup("workers_number"))
	viper.BindPFlag("server_graceful_shutdown_timeout", serverCmd.PersistentFlags().Lookup("graceful_shutdown_timeout"))
	viper.BindPFlag("resources_prefix", serverCmd.PersistentFlags().Lookup("resources_prefix"))
	viper.BindPFlag("wf_step_graceful_termination_timeout", serverCmd.PersistentFlags().Lookup("wf_step_graceful_termination_timeout"))
	viper.BindPFlag("purged_deployments_eviction_timeout", serverCmd.PersistentFlags().Lookup("purged_deployments_eviction_timeout"))
	viper.BindPFlag("server_id", serverCmd.PersistentFlags().Lookup("server_id"))
	viper.BindPFlag("disable_ssh_agent", serverCmd.PersistentFlags().Lookup("disable_ssh_agent"))
	viper.BindPFlag("locations_file_path", serverCmd.PersistentFlags().Lookup("locations_file_path"))
	viper.BindPFlag("concurrency_limit_for_upgrades", serverCmd.PersistentFlags().Lookup("concurrency_limit_for_upgrades"))
	viper.BindPFlag("ssh_connection_timeout", serverCmd.PersistentFlags().Lookup("ssh_connection_timeout"))
	viper.BindPFlag("ssh_connection_retry_backoff", serverCmd.PersistentFlags().Lookup("ssh_connection_retry_backoff"))
	viper.BindPFlag("ssh_connection_max_retries", serverCmd.PersistentFlags().Lookup("ssh_connection_max_retries"))

	viper.BindPFlag("tasks.dispatcher.long_poll_wait_time", serverCmd.PersistentFlags().Lookup("tasks_dispatcher_long_poll_wait_time"))
	viper.BindPFlag("tasks.dispatcher.lock_wait_time", serverCmd.PersistentFlags().Lookup("tasks_dispatcher_lock_wait_time"))
	viper.BindPFlag("tasks.dispatcher.metrics_refresh_time", serverCmd.PersistentFlags().Lookup("tasks_dispatcher_metrics_refresh_time"))

	//Bind Flags Yorc HTTP REST API
	viper.BindPFlag("http_port", serverCmd.PersistentFlags().Lookup("http_port"))
	viper.BindPFlag("http_address", serverCmd.PersistentFlags().Lookup("http_address"))
	viper.BindPFlag("cert_file", serverCmd.PersistentFlags().Lookup("cert_file"))
	viper.BindPFlag("key_file", serverCmd.PersistentFlags().Lookup("key_file"))
	viper.BindPFlag("ca_file", serverCmd.PersistentFlags().Lookup("ca_file"))
	viper.BindPFlag("ca_path", serverCmd.PersistentFlags().Lookup("ca_path"))
	viper.BindPFlag("ssl_verify", serverCmd.PersistentFlags().Lookup("ssl_verify"))

	//Bind Ansible persistent flags
	for key := range ansibleConfiguration {
		viper.BindPFlag(key, serverCmd.PersistentFlags().Lookup(toFlatKey(key)))
	}

	//Bind Terraform persistent flags
	for key := range terraformConfiguration {
		viper.BindPFlag(key, serverCmd.PersistentFlags().Lookup(toFlatKey(key)))
	}

	//Environment Variables
	viper.SetEnvPrefix(EnvironmentVariablePrefix)
	viper.AutomaticEnv() // read in environment variables that match
	viper.BindEnv("working_directory")
	viper.BindEnv("plugins_directory")
	viper.BindEnv("plugins_health_check_time_interval")
	viper.BindEnv("server_graceful_shutdown_timeout")
	viper.BindEnv("workers_number")
	viper.BindEnv("http_port")
	viper.BindEnv("http_address")
	viper.BindEnv("key_file")
	viper.BindEnv("ca_file")
	viper.BindEnv("ca_path")
	viper.BindEnv("cert_file")
	viper.BindEnv("SSL_verify")
	viper.BindEnv("resources_prefix")
	viper.BindEnv("server_id")
	viper.BindEnv("disable_ssh_agent")
	viper.BindEnv("locations_file_path")
	viper.BindEnv("concurrency_limit_for_upgrades")
	viper.BindEnv("ssh_connection_timeout")
	viper.BindEnv("ssh_connection_retry_backoff")
	viper.BindEnv("ssh_connection_max_retries")

	//Bind Consul environment variables flags
	for key := range consulConfiguration {
		viper.BindEnv(key, toEnvVar(key))
	}

	viper.BindEnv("wf_step_graceful_termination_timeout")
	viper.BindEnv("purged_deployments_eviction_timeout")
	viper.BindEnv("tasks.dispatcher.long_poll_wait_time")
	viper.BindEnv("tasks.dispatcher.lock_wait_time")
	viper.BindEnv("tasks.dispatcher.metrics_refresh_time")

	//Bind Ansible environment variables flags
	for key := range ansibleConfiguration {
		viper.BindEnv(key, toEnvVar(key))
	}

	//Bind Terraform environment variables flags
	for key := range terraformConfiguration {
		viper.BindEnv(key, toEnvVar(key))
	}

	//Setting Defaults
	viper.SetDefault("working_directory", "work")
	viper.SetDefault("server_graceful_shutdown_timeout", config.DefaultServerGracefulShutdownTimeout)
	viper.SetDefault("plugins_directory", config.DefaultPluginDir)
	viper.SetDefault("plugins_health_check_time_interval", config.DefaultPluginsHealthCheckTimeInterval)
	viper.SetDefault("http_port", config.DefaultHTTPPort)
	viper.SetDefault("http_address", config.DefaultHTTPAddress)
	viper.SetDefault("resources_prefix", "yorc-")
	viper.SetDefault("workers_number", config.DefaultWorkersNumber)
	viper.SetDefault("wf_step_graceful_termination_timeout", config.DefaultWfStepGracefulTerminationTimeout)
	viper.SetDefault("purged_deployments_eviction_timeout", config.DefaultPurgedDeploymentsEvictionTimeout)
	viper.SetDefault("server_id", host)
	viper.SetDefault("disable_ssh_agent", false)
	viper.SetDefault("concurrency_limit_for_upgrades", config.DefaultUpgradesConcurrencyLimit)
	viper.SetDefault("ssh_connection_timeout", config.DefaultSSHConnectionTimeout)
	viper.SetDefault("ssh_connection_retry_backoff", config.DefaultSSHConnectionRetryBackoff)
	viper.SetDefault("ssh_connection_max_retries", config.DefaultSSHConnectionMaxRetries)

	viper.SetDefault("tasks.dispatcher.long_poll_wait_time", config.DefaultTasksDispatcherLongPollWaitTime)
	viper.SetDefault("tasks.dispatcher.lock_wait_time", config.DefaultTasksDispatcherLockWaitTime)
	viper.SetDefault("tasks.dispatcher.metrics_refresh_time", config.DefaultTasksDispatcherMetricsRefreshTime)

	// Consul configuration default settings
	for key, value := range consulConfiguration {
		viper.SetDefault(key, value)
	}

	// Ansible configuration default settings
	for key, value := range ansibleConfiguration {
		viper.SetDefault(key, value)
	}

	// Terraform configuration default settings
	for key, value := range terraformConfiguration {
		viper.SetDefault(key, value)
	}

	//Configuration file directories
	viper.SetConfigName("config.yorc") // name of config file (without extension)
	viper.AddConfigPath("/etc/yorc/")  // adding home directory as first search path
	viper.AddConfigPath(".")

}

// GetConfig gets configuration from viper
func GetConfig() config.Configuration {
	configuration := config.Configuration{}
	err := viper.Unmarshal(&configuration)
	if err != nil {
		log.Fatalf("Misconfiguration error: %v", err)
	}

	if configuration.Vault == nil {
		configuration.Vault = make(config.DynamicMap)
	}
	for _, sep := range resolvedServerExtraParams {
		sep.readConfFn(&configuration)
		for _, infraParam := range sep.viperNames {
			sep.storeFn(&configuration, infraParam)
		}
	}

	return configuration
}

func readVaultViperConfig(cfg *config.Configuration) {
	vaultCfg := viper.GetStringMap("vault")
	for k, v := range vaultCfg {
		cfg.Vault.Set(k, v)
	}
}

func addServerExtraVaultParam(cfg *config.Configuration, vaultParam string) {
	paramParts := strings.Split(vaultParam, ".")
	value := viper.Get(vaultParam)
	cfg.Vault.Set(paramParts[1], value)
}

// Returns the flat key corresponding to a nested key.
// For example, for a nested key consul.token, this function will return
// consul_token
func toFlatKey(nestedKey string) string {

	var flatKey string

	// Specific code for keys that don't follow the naming scheme
	if nestedKey == "ansible.operation_remote_base_dir" ||
		nestedKey == "ansible.keep_operation_remote_path" {

		flatKey = strings.Replace(nestedKey, "ansible.", "", 1)
	} else {
		flatKey = strings.Replace(nestedKey, ".", "_", 1)
	}

	return flatKey

}

// Returns the name of the environment variable corresponding to a viper
// nested key. For example, using the prefix yorc,
// nested key consul.token will be associated to YORC_CONSUL_TOKEN environment
// variable
func toEnvVar(key string) string {

	name := EnvironmentVariablePrefix + "_" + toFlatKey(key)
	return strings.ToUpper(name)
}

// For a defined constraint symbol, a defined version as major.minor.patch ans a defined constraint level, it returns the version constraint
func versionToConstraint(constraint, version, level string) string {
	switch strings.ToLower(level) {
	case "major":
		// returns the constraint + major digit a
		tab := strings.Split(version, ".")
		if len(tab) > 1 {
			return fmt.Sprintf("%s %s", strings.TrimSpace(constraint), tab[0])
		}
		return fmt.Sprintf("%s %s", strings.TrimSpace(constraint), version)
	case "minor":
		// returns the constraint + major and minor digits a.b
		tab := strings.Split(version, ".")
		if len(tab) > 2 {
			return fmt.Sprintf("%s %s.%s", strings.TrimSpace(constraint), tab[0], tab[1])
		}
		return fmt.Sprintf("%s %s", strings.TrimSpace(constraint), version)
	case "patch":
		// returns the constraint + major,minor and patch digits a.b.c
		tab := strings.Split(version, ".")
		if len(tab) > 3 {
			return fmt.Sprintf("%s %s.%s.%s", strings.TrimSpace(constraint), tab[0], tab[1], tab[2])
		}
		return fmt.Sprintf("%s %s", strings.TrimSpace(constraint), version)
	default:
		panic("unable to build related constraint for constraint:%q, version:%q and level:%q. Unknown level")
	}

}
