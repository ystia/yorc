package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

const janusAPIDefaultErrorMsg string = "Failed to contact Janus API"

// RootCmd is the root of janus commands tree
var RootCmd = &cobra.Command{
	Use:   "janus",
	Short: "A new generation orchestrator",
	Long: `janus is the main command, used to start the http server.
Janus is a new generation orchestrator.  
It is cloud-agnostic, flexible and secure.
`,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Help()
		if err != nil {
			fmt.Print(err)
		}
	},
}
