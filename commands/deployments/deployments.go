package deployments

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"novaforge.bull.com/starlings-janus/janus/commands"
)

func init() {
	commands.RootCmd.AddCommand(DeploymentsCmd)
	setDeploymentsConfig()
}

// NoColor returns true if no-color option is set
var NoColor bool

// DeploymentsCmd is the deployments-based command
var DeploymentsCmd = &cobra.Command{
	Use:           "deployments",
	Aliases:       []string{"depls", "depl", "deps", "dep", "d"},
	Short:         "Perform commands on deployments",
	Long:          `Perform different commands on deployments`,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Help()
		if err != nil {
			fmt.Print(err)
		}
	},
}

func setDeploymentsConfig() {

	DeploymentsCmd.PersistentFlags().StringP("janus-api", "j", "localhost:8800", "specify the host and port used to join the Janus' REST API")
	DeploymentsCmd.PersistentFlags().StringP("ca-file", "", "", "This provides a file path to a PEM-encoded certificate authority. This implies the use of HTTPS to connect to the Janus REST API.")
	DeploymentsCmd.PersistentFlags().BoolVar(&NoColor, "no-color", false, "Disable coloring output")
	DeploymentsCmd.PersistentFlags().BoolP("secured", "s", false, "Use HTTPS to connect to the Janus REST API")
	DeploymentsCmd.PersistentFlags().BoolP("skip-tls-verify", "", false, "skip-tls-verify controls whether a client verifies the server's certificate chain and host name. If set to true, TLS accepts any certificate presented by the server and any host name in that certificate. In this mode, TLS is susceptible to man-in-the-middle attacks. This should be used only for testing. This implies the use of HTTPS to connect to the Janus REST API.")

	viper.BindPFlag("janus_api", DeploymentsCmd.PersistentFlags().Lookup("janus-api"))
	viper.BindPFlag("secured", DeploymentsCmd.PersistentFlags().Lookup("secured"))
	viper.BindPFlag("ca_file", DeploymentsCmd.PersistentFlags().Lookup("ca-file"))
	viper.BindPFlag("skip_tls_verify", DeploymentsCmd.PersistentFlags().Lookup("skip-tls-verify"))
	viper.SetEnvPrefix("janus")
	viper.BindEnv("janus_api", "JANUS_API")
	viper.BindEnv("secured")
	viper.BindEnv("ca_file")
	viper.BindEnv("skip_tls_verify")
	viper.SetDefault("janus_api", "localhost:8800")
	viper.SetDefault("secured", false)
	viper.SetDefault("skip_tls_verify", false)

}
