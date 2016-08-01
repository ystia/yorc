package commands

import (
        "fmt"
        "github.com/spf13/cobra"
)

const version = "Janus v0.1.0"

var versionCmd = &cobra.Command{

			Use: "version",
			Short: "Print the version",
			Long: `The version of Janus`,
			Run: func(cmd *cobra.Command, args []string) {
					fmt.Println(version)
			},	
}
