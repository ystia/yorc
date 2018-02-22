package workflows

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ystia/yorc/commands/deployments"
)

var workflowsCmd = &cobra.Command{
	Use:     "workflows",
	Short:   "Perform commands on workflows",
	Aliases: []string{"wf"},
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Help()
		if err != nil {
			fmt.Print(err)
		}
	},
}

func init() {
	deployments.DeploymentsCmd.AddCommand(workflowsCmd)
}
