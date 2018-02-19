package main

import (
	"novaforge.bull.com/starlings-janus/janus/commands"
	_ "novaforge.bull.com/starlings-janus/janus/commands/deployments"
	_ "novaforge.bull.com/starlings-janus/janus/commands/deployments/tasks"
	_ "novaforge.bull.com/starlings-janus/janus/commands/deployments/workflows"
	_ "novaforge.bull.com/starlings-janus/janus/commands/hostspool"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func main() {
	if err := commands.RootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
	log.Debug("Exiting main...")
}
