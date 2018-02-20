package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// RootCmd is the root of yorc commands tree
var RootCmd = &cobra.Command{
	Use:   "yorc",
	Short: "A new generation orchestrator",
	Long: `yorc is the main command, used to start the http server.
Yorc is a new generation orchestrator.  
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
