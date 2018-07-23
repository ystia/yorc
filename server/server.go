// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov/monitoring"
	"github.com/ystia/yorc/rest"
	"github.com/ystia/yorc/tasks/workflow"
)

// RunServer starts the Yorc server
func RunServer(configuration config.Configuration, shutdownCh chan struct{}) error {
	err := setupTelemetry(configuration)
	if err != nil {
		return err
	}

	vaultClient, err := buildVaultClient(configuration)
	if err != nil {
		return err
	}
	if vaultClient != nil {
		fm := template.FuncMap{
			"secret": vaultClient.GetSecret,
		}
		config.DefaultConfigTemplateResolver.SetTemplatesFunctions(fm)

		// Setup default vault client for TOSCA functions resolver
		deployments.DefaultVaultClient = vaultClient
	}
	var wg sync.WaitGroup
	client, err := configuration.GetConsulClient()
	if err != nil {
		return errors.Wrap(err, "Can't connect to Consul")
	}

	maxConsulPubRoutines := configuration.Consul.PubMaxRoutines
	if maxConsulPubRoutines <= 0 {
		maxConsulPubRoutines = config.DefaultConsulPubMaxRoutines
	}

	consulutil.InitConsulPublisher(maxConsulPubRoutines, client.KV())

	dispatcher := workflow.NewDispatcher(configuration, shutdownCh, client, &wg)
	go dispatcher.Run()
	var httpServer *rest.Server
	pm := newPluginManager()
	defer pm.cleanup()
	err = pm.loadPlugins(configuration)
	if err != nil {
		close(shutdownCh)
		goto WAIT
	}

	httpServer, err = rest.NewServer(configuration, client, shutdownCh)
	if err != nil {
		close(shutdownCh)
		goto WAIT
	}
	defer httpServer.Shutdown()

	// Register yorc service in Consul
	if err = consulutil.RegisterServerAsConsulService(configuration, client, shutdownCh); err != nil {
		return errors.Wrap(err, "Failed to register this yorc server as a Consul service")
	}
	// Start monitoring
	monitoring.Start(configuration, client)
	defer monitoring.Stop()

WAIT:
	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	for {
		var sig os.Signal
		shutdownChClosed := false
		select {
		case s := <-signalCh:
			sig = s
		case <-shutdownCh:
			sig = os.Interrupt
			shutdownChClosed = true
		}

		// Check if this is a SIGHUP
		if sig == syscall.SIGHUP {
			// TODO reload
		} else {
			if !shutdownChClosed {
				close(shutdownCh)
			}
			gracefulTimeout := configuration.ServerGracefulShutdownTimeout
			if gracefulTimeout == 0 {
				gracefulTimeout = config.DefaultServerGracefulShutdownTimeout
			}
			log.Printf("Waiting at least %v for a graceful server shutdown. Send another termination signal to exit immediately.", gracefulTimeout)
			gracefulCh := make(chan struct{})
			go func() {
				wg.Wait()
				close(gracefulCh)
			}()
			select {
			// Wait for another signal, a timeout or a notification that the graceful shutdown is done
			case <-signalCh:
			case <-gracefulCh:
			case <-time.After(gracefulTimeout):
			}
			return err
		}
	}
}
