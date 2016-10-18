package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"novaforge.bull.com/starlings-janus/janus/rest"
	"os"
)

func init() {
	RootCmd.AddCommand(deploymentsCmd)
	setDeploymentsConfig()
}

var deploymentsCmd = &cobra.Command{
	Use:           "deployments",
	Aliases:       []string{"depls", "depl", "deps", "dep", "d"},
	Short:         "Perform commands on deployments",
	Long:          `Perform different commands on deployments`,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func setDeploymentsConfig() {

	deploymentsCmd.PersistentFlags().StringP("janus-api", "j", "localhost:8800", "specify the host and port used to join the Janus' REST API")

	viper.BindPFlag("janus_api", deploymentsCmd.PersistentFlags().Lookup("janus-api"))
	viper.SetEnvPrefix("janus")
	viper.BindEnv("janus_api", "JANUS_API")
	viper.SetDefault("janus_api", "localhost:8800")

}

func printRestErrors(errs rest.Errors) {
	if len(errs.Errors) > 0 {
		fmt.Println("Got errors when interacting with Janus:")
	}
	for _, e := range errs.Errors {
		fmt.Printf("Error: %q: %q\n", e.Title, e.Detail)
	}
}

func errExit(msg interface{}) {
	fmt.Println("Error:", msg)
	os.Exit(1)
}
