package slurm

import (
	"novaforge.bull.com/starlings-janus/janus/helper/sshutil"
)

type provider struct {
	username string
	name     string
	url      string
	port     string
	password string
	session  sshutil.Session
}

type infrastructure struct {
	nodes    []nodeAllocation
	provider *provider
}

type nodeAllocation struct {
	cpu          string
	memory       string
	gres         string
	partition    string
	name         string
	instanceName string
}
