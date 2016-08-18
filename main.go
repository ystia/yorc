package main

import (
	"novaforge.bull.com/starlings-janus/janus/commands"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func main() {

	if err := commands.RootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
	log.Debug("Exiting main...")
}
