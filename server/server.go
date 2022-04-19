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

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov/monitoring"
	"github.com/ystia/yorc/v4/prov/scheduling/scheduler"
	"github.com/ystia/yorc/v4/rest"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/tasks/workflow"
)

func initVaultClient(configuration config.Configuration) error {
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
	return nil
}

func initConsulClient(configuration config.Configuration) (*api.Client, error) {
	client, err := configuration.GetConsulClient()
	if err != nil {
		return nil, errors.Wrap(err, "Can't connect to Consul")
	}

	maxConsulPubRoutines := configuration.Consul.PubMaxRoutines
	if maxConsulPubRoutines <= 0 {
		maxConsulPubRoutines = config.DefaultConsulPubMaxRoutines
	}

	consulutil.InitConsulPublisher(maxConsulPubRoutines, client.KV())

	// Load main stores used for deployments, logs, events
	err = storage.LoadStores(configuration)
	if err != nil {
		return nil, err
	}

	err = registerBuiltinTOSCATypes()
	if err != nil {
		return nil, err
	}

	err = setupConsulDBSchema(configuration, client)
	// return error if any
	return client, err
}

func initLocationManager(configuration config.Configuration) error {
	if configuration.LocationsFilePath != "" {
		locationMgr, err := locations.GetManager(configuration)
		if err != nil {
			return errors.Wrap(err, "Failed to initialize locations")
		}

		// Initialize locations where deployments will take place
		done, err := locationMgr.InitializeLocations(configuration.LocationsFilePath)
		if err != nil {
			return errors.Wrap(err, "Failed to initialize locations")

		}

		if done {
			log.Printf("Initialzed location from %s", configuration.LocationsFilePath)
		} else {
			log.Debugf("Locations already initialized")
		}
	}
	return nil
}

// RunServer starts the Yorc server
func RunServer(configuration config.Configuration, shutdownCh chan struct{}) error {
	err := setupTelemetry(configuration)
	if err != nil {
		return err
	}

	err = initVaultClient(configuration)
	if err != nil {
		return err
	}

	client, err := initConsulClient(configuration)
	if err != nil {
		return err
	}

	err = initLocationManager(configuration)
	if err != nil {
		return err
	}

	pm := newPluginManager()
	defer pm.cleanup()
	err = pm.loadPlugins(configuration)
	if err != nil {
		return err
	}

	httpServer, err := rest.NewServer(configuration, client, shutdownCh)
	if err != nil {
		return err
	}
	defer httpServer.Shutdown()

	// Register yorc service in Consul
	if err = consulutil.RegisterServerAsConsulService(configuration, client, shutdownCh); err != nil {
		return err
	}

	var wg sync.WaitGroup
	// Dispatcher needs
	go workflow.NewDispatcher(configuration, shutdownCh, client, &wg).Run()

	// Start monitoring
	monitoring.Start(configuration, client)
	defer monitoring.Stop()

	// Start scheduler
	scheduler.Start(configuration, client)
	defer scheduler.Stop()

	healthCheckTimeInterval := configuration.PluginsHealthCheckTimeInterval
	if healthCheckTimeInterval <= 0 {
		healthCheckTimeInterval = config.DefaultPluginsHealthCheckTimeInterval
	}
	tickerPingPlugins := time.NewTicker(healthCheckTimeInterval)
	defer tickerPingPlugins.Stop()
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
		case <-tickerPingPlugins.C:
			err := pm.pingPlugins()
			if err != nil {
				log.Printf("[ERROR] Failed to ping plugins: %s", err.Error())
				log.Printf("Cleaning up plugins")
				pm.cleanup()
				log.Printf("Reloading plugins")
				err = pm.loadPlugins(configuration)
				if err != nil {
					log.Printf("[ERROR] Failed to load plugins: %+v", errors.WithStack(err))
					return err
				}
			}
			continue
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
