package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	RootCmd.AddCommand(hostsPoolCmd)
	setHostsPoolConfig()
}

var hostsPoolCmd = &cobra.Command{
	Use:           "hostspool",
	Aliases:       []string{"hostpool", "hostsp", "hpool", "hp"},
	Short:         "Perform commands on hosts pool",
	Long:          `Allow to add, update and delete hosts pool`,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Help()
		if err != nil {
			fmt.Print(err)
		}
	},
}

func setHostsPoolConfig() {
	hostsPoolCmd.PersistentFlags().StringP("janus-api", "j", "localhost:8800", "specify the host and port used to join the Janus' REST API")
	hostsPoolCmd.PersistentFlags().StringP("ca-file", "", "", "This provides a file path to a PEM-encoded certificate authority. This implies the use of HTTPS to connect to the Janus REST API.")
	hostsPoolCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable coloring output")
	hostsPoolCmd.PersistentFlags().BoolP("secured", "s", false, "Use HTTPS to connect to the Janus REST API")
	hostsPoolCmd.PersistentFlags().BoolP("skip-tls-verify", "", false, "skip-tls-verify controls whether a client verifies the server's certificate chain and host name. If set to true, TLS accepts any certificate presented by the server and any host name in that certificate. In this mode, TLS is susceptible to man-in-the-middle attacks. This should be used only for testing. This implies the use of HTTPS to connect to the Janus REST API.")

	viper.BindPFlag("janus_api", hostsPoolCmd.PersistentFlags().Lookup("janus-api"))
	viper.BindPFlag("secured", hostsPoolCmd.PersistentFlags().Lookup("secured"))
	viper.BindPFlag("ca_file", hostsPoolCmd.PersistentFlags().Lookup("ca-file"))
	viper.BindPFlag("skip_tls_verify", hostsPoolCmd.PersistentFlags().Lookup("skip-tls-verify"))
	viper.SetEnvPrefix("janus")
	viper.BindEnv("janus_api", "JANUS_API")
	viper.BindEnv("secured")
	viper.BindEnv("ca_file")
	viper.BindEnv("skip_tls_verify")
	viper.SetDefault("janus_api", "localhost:8800")
	viper.SetDefault("secured", false)
	viper.SetDefault("skip_tls_verify", false)
}
