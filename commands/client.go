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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/log"
)

// ConfigureYorcClientCommand setups a Command of the CLI part of Yorc
func ConfigureYorcClientCommand(c *cobra.Command, v *viper.Viper, cfgFile *string, noColor *bool) {
	c.PersistentFlags().StringVarP(cfgFile, "config", "c", "", "Config file (default is /etc/yorc/yorc-client.[json|yaml])")
	c.PersistentFlags().StringP("yorc_api", "", "localhost:8800", "Specify the host and port used to join the Yorc' REST API")
	c.PersistentFlags().StringP("ca_file", "", "", "This provides a file path to a PEM-encoded certificate authority. This implies the use of HTTPS to connect to the Yorc REST API.")
	c.PersistentFlags().StringP("ca_path", "", "", "Path to a directory of PEM-encoded certificates authorities. This implies the use of HTTPS to connect to the Yorc REST API.")
	c.PersistentFlags().BoolVar(noColor, "no_color", false, "Disable coloring output")
	c.PersistentFlags().BoolP("ssl_enabled", "s", false, "Use HTTPS to connect to the Yorc REST API")
	c.PersistentFlags().BoolP("skip_tls_verify", "", false, "Controls whether a client verifies the server's certificate chain and host name. If set to true, TLS accepts any certificate presented by the server and any host name in that certificate. In this mode, TLS is susceptible to man-in-the-middle attacks. This should be used only for testing. This implies the use of HTTPS to connect to the Yorc REST API.")
	c.PersistentFlags().StringP("cert_file", "", "", "File path to a PEM-encoded client certificate used to authenticate to the Yorc API. This must be provided along with key-file. If one of key-file or cert-file is not provided then SSL authentication is disabled. If both cert-file and key-file are provided this implies the use of HTTPS to connect to the Yorc REST API.")
	c.PersistentFlags().StringP("key_file", "", "", "File path to a PEM-encoded client private key used to authenticate to the Yorc API. This must be provided along with cert-file. If one of key-file or cert-file is not provided then SSL authentication is disabled. If both cert-file and key-file are provided this implies the use of HTTPS to connect to the Yorc REST API.")

	v.BindPFlag("yorc_api", c.PersistentFlags().Lookup("yorc_api"))
	v.BindPFlag("ssl_enabled", c.PersistentFlags().Lookup("ssl_enabled"))
	v.BindPFlag("ca_file", c.PersistentFlags().Lookup("ca_file"))
	v.BindPFlag("ca_path", c.PersistentFlags().Lookup("ca_path"))
	v.BindPFlag("key_file", c.PersistentFlags().Lookup("key_file"))
	v.BindPFlag("cert_file", c.PersistentFlags().Lookup("cert_file"))
	v.BindPFlag("skip_tls_verify", c.PersistentFlags().Lookup("skip_tls_verify"))

	v.SetEnvPrefix("yorc")
	v.AutomaticEnv()
	v.BindEnv("yorc_api", "YORC_API")
	v.BindEnv("ssl_enabled")
	v.BindEnv("ca_file")
	v.BindEnv("ca_path")
	v.BindEnv("key_file")
	v.BindEnv("cert_file")
	v.BindEnv("skip_tls_verify")
	v.SetDefault("yorc_api", "localhost:8800")
	v.SetDefault("ssl_enabled", false)
	v.SetDefault("skip_tls_verify", false)

	//Configuration file directories
	v.SetConfigName("yorc-client") // name of config file (without extension)
	v.AddConfigPath("/etc/yorc/")  // adding home directory as first search path
	v.AddConfigPath(".")
}

// GetYorcClientConfig retrives Yorc client configuration
func GetYorcClientConfig(v *viper.Viper, cfgFile string) config.Client {
	if cfgFile != "" {
		// enable ability to specify config file via flag
		v.SetConfigFile(cfgFile)
	}
	// If a config file is found, read it in.
	if err := v.ReadInConfig(); err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		if cfgFile != "" || !ok {
			fmt.Println("Can't use config file:", err)
		}
	}
	clientCfg := config.Client{}
	err := v.Unmarshal(&clientCfg)
	if err != nil {
		log.Fatalf("Misconfiguration error: %v", err)
	}
	return clientCfg
}
