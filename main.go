package main

import (
	"github.com/ystia/yorc/commands"
	_ "github.com/ystia/yorc/commands/deployments"
	_ "github.com/ystia/yorc/commands/deployments/tasks"
	_ "github.com/ystia/yorc/commands/deployments/workflows"
	_ "github.com/ystia/yorc/commands/hostspool"
	"github.com/ystia/yorc/log"
)

func main() {
	if err := commands.RootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
	log.Debug("Exiting main...")
}
