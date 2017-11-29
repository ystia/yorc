package vault

import (
	"novaforge.bull.com/starlings-janus/janus/config"
)

type Client interface {
}

type ClientBuilder interface {
	BuildClient(cfg config.Configuration) (Client, error)
}
