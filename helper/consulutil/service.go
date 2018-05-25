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
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/ystia/yorc/config"
)

// RegisterServerAsConsulService allows to register the Yorc server as a Consul service
func RegisterServerAsConsulService(cfg config.Configuration, cc *api.Client) error {
	service := &api.AgentServiceRegistration{
		Name: "yorc",
		Tags: []string{"server", cfg.ServerID},
		Port: cfg.HTTPPort,
		Check: &api.AgentServiceCheck{
			Name:     "TCP check on yorc port",
			Interval: "5s",
			TCP:      fmt.Sprintf("localhost:%d", cfg.HTTPPort),
			Status:   api.HealthPassing,
		},
	}
	return cc.Agent().ServiceRegister(service)
}
