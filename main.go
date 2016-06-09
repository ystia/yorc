package main

import (
	"github.com/hashicorp/consul/api"
	"github.com/mitchellh/cli"
	"log"
	"novaforge.bull.com/starlings-janus/janus/rest"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"os"
	"os/signal"
	"syscall"
)

type Command struct {
	Ui         cli.Ui
	ShutdownCh chan struct{}
}

func (*Command) Help() string {
	return ""
}
func (*Command) Synopsis() string {
	return ""
}
func (c *Command) Run(args []string) int {

	// TODO make access to consul configurable
	// TODO deals with tokens
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Printf("Can't connect to Consul")
		return 1
	}

	dispatcher := tasks.NewDispatcher(3, c.ShutdownCh, client)
	go dispatcher.Run()
	httpServer, err := rest.NewServer(client)
	if err != nil {
		log.Print(err)
		return 1
	}
	defer httpServer.Shutdown()
	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	for {
		var sig os.Signal
		shutdownChClosed := false
		select {
		case s := <-signalCh:
			sig = s
		case <-c.ShutdownCh:
			sig = os.Interrupt
			shutdownChClosed = true
		}

		// Check if this is a SIGHUP
		if sig == syscall.SIGHUP {
			// TODO reload
		} else {
			if !shutdownChClosed {
				close(c.ShutdownCh)
			}
			return 1
		}
	}
	return 0
}

func main() {

	c := cli.NewCLI("janus", Version)
	c.Args = os.Args[1:]
	ui := &cli.BasicUi{Writer: os.Stdout}
	c.Commands = map[string]cli.CommandFactory{
		"server": func() (cli.Command, error) {
			return &Command{
				Ui:         ui,
				ShutdownCh: make(chan struct{}),
			}, nil
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		log.Println(err)
	}

	os.Exit(exitStatus)

}
