package server

import (
	"time"

	metrics "github.com/armon/go-metrics"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func setupTelemetry(cfg config.Configuration) error {
	memSink := metrics.NewInmemSink(10*time.Second, time.Minute)
	metrics.DefaultInmemSignal(memSink)
	serviceName := cfg.Telemetry.ServiceName
	if serviceName == "" {
		serviceName = "janus"
	}
	metricsConf := metrics.DefaultConfig(serviceName)
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

	if len(sinks) > 0 {
		sinks = append(sinks, memSink)
		metrics.NewGlobal(metricsConf, sinks)
	} else {
		log.Debugln("Using InMemory only telemetry")
		metrics.NewGlobal(metricsConf, memSink)
	}

	return nil
}
