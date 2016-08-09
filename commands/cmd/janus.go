package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var cfgFile string
var path string

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
