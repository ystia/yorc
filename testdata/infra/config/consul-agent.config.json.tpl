{
      "advertise_addr": "${ip_address}",
      "client_addr": "0.0.0.0",
      "data_dir": "/var/consul",
      "server": false,
      "retry_join": ${consul_servers},
      "ui": ${consul_ui},
      "telemetry": {
            "statsd_address": "${statsd_ip}:8125"
      }
}