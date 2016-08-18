package commands

import (
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "janus",
	Short: "A new generation orchestrator",
	Long: `janus is the main command, used to start the http server.
Janus is a new generation orchestrator.  
It is cloud-agnostic, flexible and secure.
`,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

