package rest

import (
	"net"

	"crypto/tls"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
)

func wrapListenerTLS(listener net.Listener, cfg config.Configuration) (net.Listener, error) {
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to load TLS certificates")
	}
	tlsConf := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	return tls.NewListener(listener, tlsConf), nil
}
