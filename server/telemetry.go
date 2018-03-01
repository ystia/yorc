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
	"time"

	metrics "github.com/armon/go-metrics"
	"github.com/armon/go-metrics/prometheus"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/log"
)

func setupTelemetry(cfg config.Configuration) error {
	memSink := metrics.NewInmemSink(10*time.Second, time.Minute)
	metrics.DefaultInmemSignal(memSink)
	serviceName := cfg.Telemetry.ServiceName
	if serviceName == "" {
		serviceName = "yorc"
	}
	metricsConf := metrics.DefaultConfig(serviceName)

	metricsConf.EnableHostname = !cfg.Telemetry.DisableHostName
	metricsConf.EnableRuntimeMetrics = !cfg.Telemetry.DisableGoRuntimeMetrics

	var sinks metrics.FanoutSink
	if cfg.Telemetry.StatsdAddress != "" {
		log.Debugf("Setting up a statsd telemetry service on %q", cfg.Telemetry.StatsdAddress)
		statsdSink, err := metrics.NewStatsdSink(cfg.Telemetry.StatsdAddress)
		if err != nil {
			return errors.Wrap(err, "Failed to create Statsd telemetry service")
		}
		sinks = append(sinks, statsdSink)
	}

	if cfg.Telemetry.StatsiteAddress != "" {
		log.Debugf("Setting up a statsite telemetry service on %q", cfg.Telemetry.StatsiteAddress)
		statsitedSink, err := metrics.NewStatsiteSink(cfg.Telemetry.StatsiteAddress)
		if err != nil {
			return errors.Wrap(err, "Failed to create Statsite telemetry service")
		}
		sinks = append(sinks, statsitedSink)
	}

	if cfg.Telemetry.PrometheusEndpoint {
		log.Debug("Setting up a Prometheus telemetry service")
		prometheusSink, err := prometheus.NewPrometheusSink()
		if err != nil {
			return errors.Wrap(err, "Failed to create Prometheus telemetry service")
		}
		sinks = append(sinks, prometheusSink)
	}

	if len(sinks) > 0 {
		sinks = append(sinks, memSink)
		metrics.NewGlobal(metricsConf, sinks)
	} else {
		log.Debugln("Using InMemory only telemetry")
		metrics.NewGlobal(metricsConf, memSink)
	}

	return nil
}
