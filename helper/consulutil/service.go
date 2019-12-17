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

package consulutil

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/log"
)

// YorcService is the service name for yorc as a Consul service
const YorcService = "yorc"

// RegisterServerAsConsulService allows to register the Yorc server as a Consul service
func RegisterServerAsConsulService(cfg config.Configuration, cc *api.Client, chShutdown chan struct{}) error {
	log.Printf("Register Yorc as Consul Service for node %q", cfg.ServerID)
	var service *api.AgentServiceRegistration

	nodeName, err := cc.Agent().NodeName()
	if err != nil {
		return errors.Wrap(err, ConsulGenericErrMsg)
	}

	var sslEnabled bool
	sslEnabled = ((&cfg.KeyFile != nil) && len(cfg.KeyFile) != 0) || ((&cfg.CertFile != nil) && len(cfg.CertFile) != 0)

	var httpString string
	if sslEnabled {
		httpString = fmt.Sprintf("https://localhost:%d/server/health", cfg.HTTPPort)
		log.Debugf("Register Yorc service with HTTPS check: %s", httpString)
	} else {
		httpString = fmt.Sprintf("http://localhost:%d/server/health", cfg.HTTPPort)
		log.Debugf("Register Yorc service with HTTP check: %s", httpString)
	}

	service = &api.AgentServiceRegistration{
		Name: YorcService,
		Tags: []string{"server", cfg.ServerID},
		Port: cfg.HTTPPort,
		Check: &api.AgentServiceCheck{
			Name:          "HTTP check on yorc port",
			Interval:      "5s",
			HTTP:          httpString,
			Method:        "GET",
			TLSSkipVerify: false,
			Header:        map[string][]string{"Accept": []string{"application/json"}},
			Status:        api.HealthCritical,
		},
	}

	go func() {
		for {
			select {
			case <-chShutdown:
				UnregisterServerAsConsulService(cfg, cc)
				return
			}
		}
	}()
	err = cc.Agent().ServiceRegister(service)
	if err != nil {
		return errors.Wrap(err, "Failed to register this yorc server as a Consul service")
	}
	return waitForServerHealthCheck(cfg, cc, nodeName, chShutdown)
}

func waitForServerHealthCheck(cfg config.Configuration, cc *api.Client, nodeName string, chShutdown chan struct{}) error {
	waitIndex := uint64(0)
	timeout := 15 * time.Second
	log.Printf("let up to %s to Yorc service to reach the ready state...", timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		var ok bool
		var err error
		ok, waitIndex, err = checkServiceHealth(ctx, cfg, cc, nodeName, waitIndex, chShutdown)
		if ok || err != nil {
			return err
		}
		select {
		case <-chShutdown:
			return nil
		case <-ctx.Done():
			return errors.Errorf("failed to reach %s state for Yorc service registered in Consul", api.HealthPassing)
		default:
		}
	}

}

func checkServiceHealth(ctx context.Context, cfg config.Configuration, cc *api.Client, nodeName string, waitIndex uint64, chShutdown chan struct{}) (bool, uint64, error) {
	qOpts := &api.QueryOptions{
		WaitIndex: waitIndex,
		WaitTime:  5 * time.Second,
	}
	qOpts = qOpts.WithContext(ctx)

	services, qMeta, err := cc.Health().Service(YorcService, cfg.ServerID, false, qOpts)
	if err != nil {
		return false, 0, errors.Wrap(err, "Failed to check this yorc server Consul service status")
	}

	for _, s := range services {
		for _, c := range s.Checks {
			log.Debugf("Service: %q, Node: %q, CheckID: %q, Status: %q", s.Service.ID, c.Node, c.CheckID, c.Status)
			if c.Node == nodeName && c.CheckID == "service:yorc" && c.Status == api.HealthPassing {
				log.Printf("Yorc service is up & running")
				return true, qMeta.LastIndex, nil
			}
		}
	}
	log.Debugf("Service: %q no passing status for this node", YorcService)
	return false, qMeta.LastIndex, nil
}

// UnregisterServerAsConsulService allows to unregister the Yorc server as a Consul service
func UnregisterServerAsConsulService(cfg config.Configuration, cc *api.Client) {
	log.Printf("Unregister Yorc as Consul Service for node %q", cfg.ServerID)
	err := cc.Agent().ServiceDeregister(YorcService)
	if err != nil {
		log.Printf("Failed to unregister Yorc as Consul service for node %q due to:%+v", cfg.ServerID, err)
	}
}
