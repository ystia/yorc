package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "unknown phantom version"

var gitCommit = "dev version"

func init() {
	var quiet bool
	versionCmd := &cobra.Command{

		Use:   "version",
		Short: "Print the version",
		Long:  `The version of Janus`,
		Run: func(cmd *cobra.Command, args []string) {
			if quiet {
				fmt.Println(version)
			} else {
				fmt.Println("Janus Server", version)
				fmt.Printf("Revision: %q\n", gitCommit)
			}

		},
	}
	versionCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, fmt.Sprintf("Print just the release number in machine readable format (ie: %s)", version))

	RootCmd.AddCommand(versionCmd)
}
